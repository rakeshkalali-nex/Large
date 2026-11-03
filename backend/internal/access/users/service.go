// Package users owns retained staff identities and Administrator-only access changes.
package users

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
)

type User struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Status      string   `json:"status"`
	Version     int64    `json:"version"`
	ProjectIDs  []string `json:"projectIds"`
}
type Input struct {
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	ProjectIDs  []string `json:"projectIds"`
	Version     int64    `json:"version"`
}
type Transition struct {
	Version int64  `json:"version"`
	Reason  string `json:"reason"`
}
type Option struct {
	ID   string `json:"id,omitempty"`
	Code string `json:"code,omitempty"`
	Name string `json:"name"`
}
type Service struct{ Sessions *sessions.Service }

func New(d contracts.Dependencies) *Service { return &Service{sessions.New(d)} }

func load(ctx context.Context, tx contracts.Transaction, id string) (User, error) {
	u := User{ProjectIDs: []string{}}
	err := tx.QueryRowContext(ctx, `SELECT id,display_name,email,role_code,status,version FROM users WHERE id=?`, id).Scan(&u.ID, &u.DisplayName, &u.Email, &u.Role, &u.Status, &u.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return u, policy.NotFound()
	}
	if err != nil {
		return u, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT project_id FROM user_projects WHERE user_id=? ORDER BY project_id`, id)
	if err != nil {
		return u, err
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err = rows.Scan(&p); err != nil {
			return u, err
		}
		u.ProjectIDs = append(u.ProjectIDs, p)
	}
	return u, rows.Err()
}

// Reads share the serialized authorization boundary so revocation cannot race a list.
func (s *Service) List(ctx context.Context, token, search string) ([]User, error) {
	out := []User{}
	if len(search) > 320 {
		return out, transactions.Invalid()
	}
	err := s.Sessions.WithinAuthorizedWrite(ctx, token, "users.manage", "", func(ctx context.Context, tx contracts.Transaction) error {
		rows, e := tx.QueryContext(ctx, `SELECT id FROM users WHERE instr(lower(display_name),lower(?))>0 OR instr(normalized_email,lower(?))>0 ORDER BY display_name,id`, search, search)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			u, e := load(ctx, tx, id)
			if e != nil {
				return e
			}
			out = append(out, u)
		}
		return nil
	})
	return out, err
}
func (s *Service) Options(ctx context.Context, token string, projects bool) ([]Option, error) {
	out := []Option{}
	err := s.Sessions.WithinAuthorizedWrite(ctx, token, "users.manage", "", func(ctx context.Context, tx contracts.Transaction) error {
		query := `SELECT code,name FROM roles ORDER BY name`
		if projects {
			query = `SELECT id,name FROM projects ORDER BY name,id`
		}
		rows, e := tx.QueryContext(ctx, query)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var o Option
			var id string
			if e = rows.Scan(&id, &o.Name); e != nil {
				return e
			}
			if projects {
				o.ID = id
			} else {
				o.Code = id
			}
			out = append(out, o)
		}
		return rows.Err()
	})
	return out, err
}
func validate(in *Input) error {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Email = strings.TrimSpace(in.Email)
	address, e := mail.ParseAddress(in.Email)
	if e != nil || address.Address != in.Email || len(in.Email) > 254 || in.DisplayName == "" || utf8.RuneCountInString(in.DisplayName) > 120 || !utf8.ValidString(in.DisplayName) || in.ProjectIDs == nil || len(in.ProjectIDs) > 1000 {
		return transactions.Invalid()
	}
	for _, r := range in.DisplayName {
		if unicode.IsControl(r) {
			return transactions.Invalid()
		}
	}
	seen := map[string]bool{}
	for _, id := range in.ProjectIDs {
		if id == "" || seen[id] {
			return transactions.Invalid()
		}
		seen[id] = true
	}
	return nil
}
func (s *Service) Save(ctx context.Context, token, id, key string, in Input) (User, error) {
	if e := validate(&in); e != nil {
		return User{}, e
	}
	return s.mutate(ctx, token, id, key, "save", in, func(ctx context.Context, tx contracts.Transaction, actor string, old User) (User, error) {
		var n int
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM roles WHERE code=?`, in.Role).Scan(&n); e != nil {
			return User{}, e
		}
		if n != 1 {
			return User{}, transactions.Invalid()
		}
		for _, p := range in.ProjectIDs {
			if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM projects WHERE id=?`, p).Scan(&n); e != nil {
				return User{}, e
			}
			if n != 1 {
				return User{}, transactions.Invalid()
			}
		}
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE normalized_email=? AND id<>?`, strings.ToLower(in.Email), old.ID).Scan(&n); e != nil {
			return User{}, e
		}
		if n != 0 {
			return User{}, transactions.Conflict("EMAIL_IN_USE")
		}
		stamp := transactions.Timestamp(s.Sessions.Clock.Now())
		if old.ID == "" {
			old.ID = transactions.NewID()
			_, e := tx.ExecContext(ctx, `INSERT INTO users(id,display_name,email,normalized_email,role_code,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, old.ID, in.DisplayName, in.Email, strings.ToLower(in.Email), in.Role, actor, stamp, stamp)
			if e != nil {
				return User{}, e
			}
		} else {
			if e := transactions.CheckVersion(old.Version, in.Version); e != nil {
				return User{}, e
			}
			if old.Role == "administrator" && old.Status == "active" && in.Role != "administrator" {
				if e := finalAdmin(ctx, tx); e != nil {
					return User{}, e
				}
			}
			_, e := tx.ExecContext(ctx, `UPDATE users SET display_name=?,email=?,normalized_email=?,role_code=?,version=version+1,access_revision=access_revision+1,updated_at=? WHERE id=?`, in.DisplayName, in.Email, strings.ToLower(in.Email), in.Role, stamp, old.ID)
			if e != nil {
				return User{}, e
			}
			if e := revoke(ctx, tx, old.ID, stamp, !strings.EqualFold(old.Email, in.Email)); e != nil {
				return User{}, e
			}
		}
		if _, e := tx.ExecContext(ctx, `DELETE FROM user_projects WHERE user_id=?`, old.ID); e != nil {
			return User{}, e
		}
		for _, p := range in.ProjectIDs {
			if _, e := tx.ExecContext(ctx, `INSERT INTO user_projects(user_id,project_id,created_by,created_at) VALUES(?,?,?,?)`, old.ID, p, actor, stamp); e != nil {
				return User{}, e
			}
		}
		// Retain assignment history as project-scoped audit evidence, not deleted rows.
		before, after := map[string]bool{}, map[string]bool{}
		for _, p := range old.ProjectIDs {
			before[p] = true
		}
		for _, p := range in.ProjectIDs {
			after[p] = true
		}
		for _, set := range []map[string]bool{before, after} {
			for p := range set {
				if before[p] == after[p] {
					continue
				}
				action := "user.unassign"
				if after[p] {
					action = "user.assign"
				}
				if e := s.Sessions.Audit.Append(ctx, tx, contracts.AuditEvent{ActorID: actor, RequestID: transactions.NewID(), RecordType: "user", RecordID: old.ID, ProjectID: &p, Action: action, OccurredAt: s.Sessions.Clock.Now()}); e != nil {
					return User{}, e
				}
			}
		}
		return load(ctx, tx, old.ID)
	})
}
func finalAdmin(ctx context.Context, tx contracts.Transaction) error {
	var n int
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE role_code='administrator' AND status='active' AND setup_required=0`).Scan(&n); e != nil {
		return e
	}
	if n <= 1 {
		return transactions.Conflict("FINAL_ACTIVE_ADMINISTRATOR")
	}
	return nil
}
func revoke(ctx context.Context, tx contracts.Transaction, id, stamp string, tokens bool) error {
	if _, e := tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, stamp, id); e != nil {
		return e
	}
	if tokens {
		_, e := tx.ExecContext(ctx, `UPDATE password_tokens SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL AND consumed_at IS NULL`, stamp, id)
		return e
	}
	return nil
}
func (s *Service) ChangeStatus(ctx context.Context, token, id, key, action string, in Transition) (User, error) {
	if id == "" || in.Version < 1 || strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 1000 || (action != "deactivate" && action != "reactivate") {
		return User{}, transactions.Invalid()
	}
	return s.mutate(ctx, token, id, key, action, in, func(ctx context.Context, tx contracts.Transaction, actor string, old User) (User, error) {
		if e := transactions.CheckVersion(old.Version, in.Version); e != nil {
			return User{}, e
		}
		if action == "deactivate" && old.Status == "deactivated" || action == "reactivate" && old.Status != "deactivated" {
			return User{}, transactions.Conflict("INVALID_TRANSITION")
		}
		if old.Role == "administrator" && old.Status == "active" {
			if e := finalAdmin(ctx, tx); e != nil {
				return User{}, e
			}
		}
		stamp := transactions.Timestamp(s.Sessions.Clock.Now())
		status := "deactivated"
		var deactivated any = stamp
		if action == "reactivate" {
			status = "setup-required"
			deactivated = nil
		}
		_, e := tx.ExecContext(ctx, `UPDATE users SET status=?,setup_required=1,password_hash=NULL,deactivated_at=?,version=version+1,access_revision=access_revision+1,updated_at=? WHERE id=?`, status, deactivated, stamp, id)
		if e != nil {
			return User{}, e
		}
		if e = revoke(ctx, tx, id, stamp, true); e != nil {
			return User{}, e
		}
		return load(ctx, tx, id)
	})
}
func (s *Service) mutate(ctx context.Context, token, id, key, action string, input any, work func(context.Context, contracts.Transaction, string, User) (User, error)) (User, error) {
	var out User
	err := s.Sessions.WithinAuthorizedWrite(ctx, token, "users.manage", "", func(ctx context.Context, tx contracts.Transaction) error {
		var actor string
		if e := tx.QueryRowContext(ctx, `SELECT user_id FROM sessions WHERE token_digest=?`, sessions.Digest(token)).Scan(&actor); e != nil {
			return e
		}
		digest, e := transactions.Digest(struct {
			ID    string
			Input any
		}{id, input})
		if e != nil {
			return e
		}
		request := transport.RequestID(ctx)
		if request == "" {
			request = transactions.NewID()
		}
		identity := contracts.MutationIdentity{ActorID: actor, RequestID: request, Operation: "users." + action, IdempotencyKey: key, RequestDigest: digest}
		store := transactions.Store{Now: s.Sessions.Clock.Now}
		prior, e := store.Lookup(ctx, tx, identity)
		if e != nil {
			return e
		}
		if prior != nil {
			out, e = load(ctx, tx, prior.ResourceID)
			return e
		}
		old := User{}
		if id != "" {
			old, e = load(ctx, tx, id)
			if e != nil {
				return e
			}
		}
		out, e = work(ctx, tx, actor, old)
		if e != nil {
			return e
		}
		verb := "update"
		if id == "" {
			verb = "create"
		}
		var reason *string
		if transition, ok := input.(Transition); ok {
			reason = &transition.Reason
			verb = "deactivate"
			if action == "reactivate" {
				verb = "restore"
			}
		}
		changes := []contracts.SafeChange{{Field: "status", Before: nil, After: &out.Status}, {Field: "role_code", Before: nil, After: &out.Role}}
		if old.ID != "" {
			changes[0].Before = &old.Status
			changes[1].Before = &old.Role
		}
		if e = s.Sessions.Audit.Append(ctx, tx, contracts.AuditEvent{ActorID: actor, RequestID: request, RecordID: out.ID, RecordType: "user", Action: "user." + verb, Reason: reason, Changes: changes, OccurredAt: s.Sessions.Clock.Now()}); e != nil {
			return e
		}
		return store.Record(ctx, tx, identity, contracts.CommittedResult{Status: 200, ResourceType: "user", ResourceID: out.ID, Version: out.Version})
	})
	if err != nil {
		return User{}, err
	}
	return out, nil
}
