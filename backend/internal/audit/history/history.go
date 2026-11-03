// Package history exposes immutable, least-data audit projections.
package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
)

type Change struct {
	Field  string  `json:"field"`
	Before *string `json:"before"`
	After  *string `json:"after"`
}
type Event struct {
	ID          string    `json:"id"`
	ActorID     *string   `json:"actorId"`
	ActorName   *string   `json:"actorName"`
	OccurredAt  string    `json:"occurredAt"`
	Action      string    `json:"action"`
	ProjectID   *string   `json:"projectId"`
	ProjectName *string   `json:"projectName"`
	RecordType  string    `json:"recordType"`
	RecordID    string    `json:"recordId"`
	RecordLink  *string   `json:"recordLink"`
	RequestID   string    `json:"requestId"`
	Reason      *string   `json:"reason"`
	Changes     []Change  `json:"changes"`
	Retention   Retention `json:"retention"`
}
type Page struct {
	NextCursor *string `json:"nextCursor"`
	Limit      int     `json:"limit"`
}
type cursor struct {
	Binding string
	At      string
	ID      string
}
type Service struct{ D contracts.Dependencies }

func invalid() error {
	return &transactions.Error{Status: 422, Code: "VALIDATION_FAILED", Message: "Audit filters or cursor are invalid."}
}
func RegisterRoutes(mux *http.ServeMux, d contracts.Dependencies) {
	s := Service{d}
	mux.HandleFunc("GET /api/v1/audit-events", s.read)
	mux.HandleFunc("GET /api/v1/audit-events/{id}", s.read)
}
func (s Service) read(w http.ResponseWriter, r *http.Request) {
	events, page, err := s.Read(r.Context(), r.URL.Query(), r.PathValue("id"))
	if err != nil {
		transactions.WriteError(w, r, err)
		return
	}
	if r.PathValue("id") != "" {
		transport.JSON(w, r, 200, events[0])
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": events, "page": page, "requestId": transport.RequestID(r.Context())})
}

// Read authorizes and queries within one committed snapshot; no browser role is accepted.
func (s Service) Read(ctx context.Context, values url.Values, id string) ([]Event, Page, error) {
	if s.D.DB == nil || s.D.Clock == nil {
		return nil, Page{}, &transactions.Error{Status: 503, Code: "SERVICE_UNAVAILABLE", Message: "Audit history is unavailable."}
	}
	tx, err := s.D.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, Page{}, err
	}
	defer tx.Rollback()
	p := policy.Service{DB: s.D.DB, Clock: s.D.Clock}
	if err = p.AuthorizeTx(ctx, tx, "audit.scoped", ""); err != nil {
		return nil, Page{}, err
	}
	principal := policy.PrincipalFrom(ctx)
	var role string
	var accessRevision int64
	if err = tx.QueryRowContext(ctx, "SELECT role_code,access_revision FROM users WHERE id=?", principal.UserID).Scan(&role, &accessRevision); err != nil {
		return nil, Page{}, err
	}
	q, err := transport.ParseListQuery(values, []string{"occurredAt"}, []string{"q", "actorId", "action", "projectId", "recordType", "recordId", "from", "to"})
	if err != nil {
		return nil, Page{}, invalid()
	}
	if !values.Has("direction") {
		q.Direction = "desc"
	}
	for key, v := range q.Filters {
		if len(v) > 200 {
			return nil, Page{}, invalid()
		}
		if (key == "from" || key == "to") && v != "" {
			t, e := time.Parse(time.RFC3339Nano, v)
			if e != nil || !strings.HasSuffix(v, "Z") {
				return nil, Page{}, invalid()
			}
			q.Filters[key] = t.UTC().Format("2006-01-02T15:04:05.000Z")
		}
	}
	if q.Filters["from"] != "" && q.Filters["to"] != "" && q.Filters["from"] > q.Filters["to"] {
		return nil, Page{}, invalid()
	}
	where := "1=1"
	args := []any{}
	if role != "administrator" {
		where += " AND EXISTS(SELECT 1 FROM user_projects up WHERE up.project_id=a.project_id AND up.user_id=?)"
		args = append(args, principal.UserID)
		if role == "finance-officer" {
			where += " AND a.record_type IN ('invoice','payment','refund','expense','budget','budget-revision','purchase-order','purchase-order-amendment','goods-receipt','work-order','work-order-amendment','stock-adjustment','reconciliation','booking','cancellation')"
		} else {
			where += " AND a.record_type IN ('site-log','daily-log','inspection','defect','purchase-order','purchase-order-amendment','work-order','work-order-amendment','expense','budget','budget-revision','stock-adjustment','reconciliation') AND (a.action IN ('approve','reject','return','submit','confirm') OR substr(a.action,instr(a.action,'.')+1) IN ('approve','reject','return','submit','confirm'))"
		}
	}
	columns := map[string]string{"actorId": "a.actor_id", "action": "a.action", "projectId": "a.project_id", "recordType": "a.record_type", "recordId": "a.record_id"}
	for _, key := range []string{"actorId", "action", "projectId", "recordType", "recordId"} {
		if v := q.Filters[key]; v != "" {
			where += " AND " + columns[key] + "=?"
			args = append(args, v)
		}
	}
	for _, key := range []string{"from", "to"} {
		if v := q.Filters[key]; v != "" {
			op := ">="
			if key == "to" {
				op = "<="
			}
			where += " AND a.occurred_at" + op + "?"
			args = append(args, v)
		}
	}
	if v := q.Filters["q"]; v != "" {
		where += " AND (instr(lower(a.action),lower(?))>0 OR instr(lower(a.record_id),lower(?))>0 OR instr(lower(a.request_id),lower(?))>0 OR instr(lower(coalesce(u.display_name,'')),lower(?))>0)"
		args = append(args, v, v, v, v)
	}
	if id != "" {
		where += " AND a.id=?"
		args = append(args, id)
	}
	bindingRaw, _ := json.Marshal([]any{principal.UserID, principal.SessionDigest, role, accessRevision, q.Filters, q.Direction, q.Limit})
	binding := fmt.Sprintf("%x", sha256.Sum256(bindingRaw))
	if q.Cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(q.Cursor)
		var c cursor
		if e != nil || json.Unmarshal(raw, &c) != nil || c.Binding != binding || c.At == "" || c.ID == "" {
			return nil, Page{}, invalid()
		}
		op := "<"
		if q.Direction == "asc" {
			op = ">"
		}
		where += " AND (a.occurred_at" + op + "? OR (a.occurred_at=? AND a.id" + op + "?))"
		args = append(args, c.At, c.At, c.ID)
	}
	args = append(args, q.Limit+1)
	rows, err := tx.QueryContext(ctx, `SELECT a.id,a.actor_id,u.display_name,a.occurred_at,a.action,a.project_id,p.name,a.record_type,a.record_id,a.request_id,a.reason,a.changes_json FROM audit_events a LEFT JOIN users u ON u.id=a.actor_id LEFT JOIN projects p ON p.id=a.project_id WHERE `+where+" ORDER BY a.occurred_at "+q.Direction+",a.id "+q.Direction+" LIMIT ?", args...)
	if err != nil {
		return nil, Page{}, err
	}
	out := []Event{}
	for rows.Next() {
		var e Event
		var raw string
		if err = rows.Scan(&e.ID, &e.ActorID, &e.ActorName, &e.OccurredAt, &e.Action, &e.ProjectID, &e.ProjectName, &e.RecordType, &e.RecordID, &e.RequestID, &e.Reason, &raw); err != nil {
			rows.Close()
			return nil, Page{}, err
		}
		e.Changes = safeChanges(raw, role)
		if e.Reason != nil && unsafeText.MatchString(*e.Reason) {
			e.Reason = nil
		}
		e.RecordLink = recordLink(e, role)
		out = append(out, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, Page{}, err
	}
	page := Page{Limit: q.Limit}
	if len(out) > q.Limit {
		out = out[:q.Limit]
		last := out[len(out)-1]
		raw, _ := json.Marshal(cursor{binding, last.OccurredAt, last.ID})
		next := base64.RawURLEncoding.EncodeToString(raw)
		page.NextCursor = &next
	}
	if id != "" && len(out) == 0 {
		return nil, Page{}, policy.NotFound()
	}
	for i := range out {
		out[i].Retention, err = retentionFor(ctx, tx, out[i].ID)
		if err != nil {
			return nil, Page{}, err
		}
	}
	return out, page, tx.Commit()
}

var unsafeText = regexp.MustCompile(`(?i)password|passwd|secret|token|cookie|authorization|bearer\s|api[_ -]?key|session[_ -]?id|-----BEGIN|panic:|sqlite|sqlstate|stack trace|traceback|goroutine|[A-Za-z0-9_-]{40,}`)

func safeValue(field string, value *string) bool {
	if value == nil {
		return true
	}
	v := *value
	enums := map[string]string{
		"status":         "draft active archived setup-required deactivated on-hold completed planned in-progress cancelled submitted returned approved rejected posted reversed part-received received superseded revised scheduled pending-review converted expired cancellation-requested unconfirmed confirmed open resolved reopened quarantine validated committed failed in-review reviewed executed released",
		"role_code":      "administrator project-manager site-engineer procurement-officer finance-officer sales-executive management-viewer",
		"classification": "operational financial sales restricted", "method": "cash cheque bank-transfer other", "priority": "low normal high critical", "archived": "true false", "enabled": "true false",
	}
	if choices, ok := enums[field]; ok {
		for _, choice := range strings.Fields(choices) {
			if v == choice {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(field, "date") {
		d, e := time.Parse("2006-01-02", v)
		return e == nil && d.Format("2006-01-02") == v
	}
	n, e := strconv.ParseInt(v, 10, 64)
	return e == nil && strconv.FormatInt(n, 10) == v
}

func safeChanges(raw, role string) []Change {
	out := []Change{}
	var changes []Change
	if len(raw) > 65536 || json.Unmarshal([]byte(raw), &changes) != nil {
		return out
	}
	fields := " status role_code classification method priority archived enabled start_date end_date date due_date version revision access_revision amount_minor total_minor quantity quantity_delta completion_bps weight_bps "
	for _, c := range changes {
		if !strings.Contains(fields, " "+c.Field+" ") {
			continue
		}
		if role == "project-manager" && (c.Field == "amount_minor" || c.Field == "total_minor") {
			continue
		}
		ok := true
		for _, v := range []*string{c.Before, c.After} {
			if !safeValue(c.Field, v) {
				ok = false
			}
		}
		if ok {
			out = append(out, c)
		}
	}
	return out
}
func recordLink(e Event, role string) *string {
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`).MatchString(e.RecordID) {
		return nil
	}
	routes := map[string]string{"project": "/projects/", "site-log": "/site-logs/", "daily-log": "/site-logs/", "purchase-order": "/purchase-orders/", "work-order": "/work-orders/", "booking": "/bookings/", "property": "/properties/", "lead": "/leads/", "contractor": "/contractors/"}
	if role == "finance-officer" && (e.RecordType == "project" || e.RecordType == "site-log" || e.RecordType == "daily-log" || e.RecordType == "property" || e.RecordType == "lead") {
		return nil
	}
	if path := routes[e.RecordType]; path != "" {
		v := path + url.PathEscape(e.RecordID)
		return &v
	}
	return nil
}
