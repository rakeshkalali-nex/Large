package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
)

// Stewardship projections intentionally omit credentials and internal storage paths.
type stewardCorrections struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}
type stewardCandidate struct {
	ID         string   `json:"id"`
	RecordType string   `json:"recordType"`
	RecordID   string   `json:"recordId"`
	EligibleAt string   `json:"eligibleAt"`
	Action     string   `json:"action"`
	HoldIDs    []string `json:"holdIds"`
	ExecutedAt *string  `json:"executedAt"`
}

func (s Service) registerStewardship(mux *http.ServeMux) {
	for _, resource := range []string{"privacy-requests", "retention-holds", "retention-reviews"} {
		resource := resource
		capability := "retention.manage"
		if resource == "privacy-requests" {
			capability = "privacy.manage"
		}
		mux.HandleFunc("GET /api/v1/"+resource, s.handle(capability, func(w http.ResponseWriter, r *http.Request) {
			p, e := stewardPaging(r.URL.RawQuery)
			if e != nil {
				respond(w, r, nil, e)
				return
			}
			v, e := stewardRead(r.Context(), s.D.DB, resource, "", p)
			respond(w, r, v, e)
		}))
		if resource != "retention-holds" {
			mux.HandleFunc("GET /api/v1/"+resource+"/{id}", s.handle(capability, func(w http.ResponseWriter, r *http.Request) {
				v, e := stewardRead(r.Context(), s.D.DB, resource, r.PathValue("id"))
				respond(w, r, v, e)
			}))
		}
	}
	mux.HandleFunc("POST /api/v1/privacy-requests", s.handle("privacy.manage", s.stewardCreatePrivacy))
	mux.HandleFunc("POST /api/v1/privacy-requests/{id}/review", s.handle("privacy.manage", s.stewardReviewPrivacy))
	mux.HandleFunc("POST /api/v1/privacy-requests/{id}/execute", s.handle("privacy.manage", s.stewardExecutePrivacy))
	mux.HandleFunc("POST /api/v1/retention-holds", s.handle("retention.manage", s.stewardCreateHold))
	mux.HandleFunc("POST /api/v1/retention-holds/{id}/release", s.handle("retention.manage", s.stewardReleaseHold))
	mux.HandleFunc("POST /api/v1/retention-reviews", s.handle("retention.manage", s.stewardCreateReview))
	mux.HandleFunc("POST /api/v1/retention-reviews/{id}/execute", s.handle("retention.manage", s.stewardExecuteReview))
}

// JSON is assembled from a closed projection, never SELECT * or user supplied SQL.
type stewardPage struct{ Limit, Offset int }

func stewardPaging(raw string) (stewardPage, error) {
	p := stewardPage{Limit: 100}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return p, invalid()
	}
	for key, vs := range values {
		if (key != "limit" && key != "offset") || len(vs) != 1 {
			return p, invalid()
		}
		n, e := strconv.Atoi(vs[0])
		if e != nil || n < 0 {
			return p, invalid()
		}
		if key == "limit" {
			if n < 1 || n > 100 {
				return p, invalid()
			}
			p.Limit = n
		} else {
			p.Offset = n
		}
	}
	return p, nil
}
func stewardRead(ctx context.Context, q contracts.Transaction, resource, id string, pages ...stewardPage) (any, error) {
	table, projection := "", ""
	switch resource {
	case "privacy-requests":
		table = "privacy_requests"
		projection = `json_object('id',id,'version',version,'subjectType',subject_type,'subjectId',subject_id,'kind',kind,'reason',reason,'status',status,'createdAt',created_at,'completedAt',completed_at,'review',CASE WHEN reviewed_by IS NULL THEN NULL ELSE json_object('reviewedBy',reviewed_by,'reviewedAt',reviewed_at,'reason',review_reason,'corrections',json_object('name',correction_name,'email',correction_email,'phone',correction_phone)) END,'accessResult',json(CASE WHEN kind='access' AND status='completed' THEN CASE subject_type WHEN 'user' THEN (SELECT json_object('name',display_name,'email',email,'phone',NULL) FROM users WHERE id=subject_id) WHEN 'lead' THEN (SELECT json_object('name',contact_name,'email',contact_email,'phone',contact_phone) FROM leads WHERE id=subject_id) WHEN 'customer' THEN (SELECT json_object('name',name,'email',(SELECT email FROM customer_contacts WHERE customer_id=customers.id ORDER BY id LIMIT 1),'phone',(SELECT phone FROM customer_contacts WHERE customer_id=customers.id ORDER BY id LIMIT 1)) FROM customers WHERE id=subject_id) WHEN 'supplier-contact' THEN (SELECT json_object('name',name,'email',email,'phone',phone) FROM supplier_contacts WHERE id=subject_id) WHEN 'contractor-contact' THEN (SELECT json_object('name',name,'email',email,'phone',phone) FROM contractor_contacts WHERE id=subject_id) END ELSE NULL END))`
	case "retention-holds":
		table = "retention_holds"
		projection = `json_object('id',id,'version',version,'projectId',project_id,'recordType',record_type,'recordId',record_id,'reason',reason,'reviewAt',review_at,'status',status,'releasedAt',released_at,'releaseReason',release_reason)`
	case "retention-reviews":
		table = "retention_reviews"
		projection = `json_object('id',id,'version',version,'projectId',project_id,'asOfDate',as_of_date,'status',status,'executedAt',executed_at,'executionReason',execution_reason,'candidates',json((SELECT json_group_array(json_object('id',c.id,'recordType',c.record_type,'recordId',c.record_id,'eligibleAt',c.eligible_at,'action',c.action,'executedAt',c.executed_at,'holdIds',json((SELECT json_group_array(hold_id) FROM retention_candidate_holds WHERE candidate_id=c.id)))) FROM retention_review_candidates c WHERE c.review_id=retention_reviews.id)))`
	default:
		return nil, invalid()
	}
	if id != "" {
		var raw string
		err := q.QueryRowContext(ctx, "SELECT "+projection+" FROM "+table+" WHERE id=?", id).Scan(&raw)
		if err == sql.ErrNoRows {
			return nil, stewardMissing()
		}
		return json.RawMessage(raw), err
	}
	p := stewardPage{Limit: 100}
	if len(pages) > 0 {
		p = pages[0]
	}
	if p.Limit < 1 || p.Limit > 100 || p.Offset < 0 {
		return nil, invalid()
	}
	rows, err := q.QueryContext(ctx, "SELECT "+projection+" FROM "+table+" ORDER BY created_at DESC,id LIMIT ? OFFSET ?", p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []json.RawMessage{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		result = append(result, json.RawMessage(raw))
	}
	return result, rows.Err()
}
func stewardMissing() error {
	return &transactions.Error{Status: 404, Code: "NOT_FOUND", Message: "Record not found."}
}
func stewardReason(v string) bool { return strings.TrimSpace(v) != "" && len(v) <= 2000 }
func stewardNullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func stewardChanged(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return transactions.Conflict("STALE_VERSION")
	}
	return nil
}
func (s Service) stewardReply(w http.ResponseWriter, r *http.Request, resource, id string, err error) {
	if err == nil && mutationReplay(r.Context()) != nil {
		respond(w, r, nil, nil)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = stewardMissing()
	}
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	v, e := stewardRead(r.Context(), s.D.DB, resource, id)
	respond(w, r, v, e)
}

func stewardSubjectTable(kind string) string {
	switch kind {
	case "user":
		return "users"
	case "lead":
		return "leads"
	case "customer":
		return "customers"
	case "supplier-contact":
		return "supplier_contacts"
	case "contractor-contact":
		return "contractor_contacts"
	}
	return ""
}
func stewardExists(ctx context.Context, tx contracts.Transaction, kind, id string) error {
	table := stewardSubjectTable(kind)
	if table == "" {
		switch kind {
		case "project":
			table = "projects"
		case "session":
			table = "sessions"
		case "password-token":
			table = "password_tokens"
		case "audit-event":
			table = "audit_events"
		case "document":
			table = "documents"
		case "supplier":
			table = "suppliers"
		case "contractor":
			table = "contractors"
		case "invoice":
			table = "invoices"
		case "payment":
			table = "payments"
		case "booking":
			table = "bookings"
		case "purchase-order":
			table = "purchase_orders"
		case "work-order":
			table = "work_orders"
		}
	}
	if table == "" || id == "" {
		return invalid()
	}
	var one int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM "+table+" WHERE id=?", id).Scan(&one)
	if err == sql.ErrNoRows {
		return stewardMissing()
	}
	return err
}

// Includes project inheritance and parent contact holds. Global actors/contacts may
// participate in more than one project; every matching hold must block mutation.
func stewardHolds(ctx context.Context, tx contracts.Transaction, kind, id string) ([]string, error) {
	projects := "SELECT NULL WHERE 0"
	args := []any{kind, id}
	switch kind {
	case "project":
		projects = "SELECT id FROM projects WHERE id=?"
	case "lead", "customer", "document", "invoice", "payment", "booking", "purchase-order", "work-order":
		tables := map[string]string{"lead": "leads", "customer": "customers", "document": "documents", "invoice": "invoices", "payment": "payments", "booking": "bookings", "purchase-order": "purchase_orders", "work-order": "work_orders"}
		projects = "SELECT project_id FROM " + tables[kind] + " WHERE id=?"
	case "user":
		projects = "SELECT project_id FROM user_projects WHERE user_id=? UNION SELECT project_id FROM audit_events WHERE actor_id=?"
	case "supplier-contact":
		projects = "SELECT project_id FROM purchase_orders WHERE supplier_id=(SELECT supplier_id FROM supplier_contacts WHERE id=?)"
	case "contractor-contact":
		projects = "SELECT project_id FROM contractor_projects WHERE contractor_id=(SELECT contractor_id FROM contractor_contacts WHERE id=?)"
	case "session", "password-token":
		table := "sessions"
		if kind == "password-token" {
			table = "password_tokens"
		}
		projects = "SELECT project_id FROM user_projects WHERE user_id=(SELECT user_id FROM " + table + " WHERE id=?)"
	case "audit-event":
		projects = "SELECT project_id FROM audit_events WHERE id=?"
	}
	for i := 0; i < strings.Count(projects, "?"); i++ {
		args = append(args, id)
	}
	query := `SELECT id FROM retention_holds WHERE status='active' AND ((record_type=? AND record_id=?) OR (record_type='project' AND record_id IN (` + projects + `)))`
	if kind == "supplier-contact" || kind == "contractor-contact" {
		parent := strings.TrimSuffix(kind, "-contact")
		query += ` OR (status='active' AND record_type=? AND record_id=(SELECT ` + parent + `_id FROM ` + parent + `_contacts WHERE id=?))`
		args = append(args, parent, id)
	}
	if kind == "session" || kind == "password-token" {
		table := "sessions"
		if kind == "password-token" {
			table = "password_tokens"
		}
		query += ` OR (status='active' AND record_type='user' AND record_id=(SELECT user_id FROM ` + table + ` WHERE id=?))`
		args = append(args, id)
	}
	if kind == "audit-event" {
		query += ` OR (status='active' AND EXISTS(SELECT 1 FROM audit_events a WHERE a.id=? AND a.record_type=retention_holds.record_type AND a.record_id=retention_holds.record_id))`
		args = append(args, id)
	}
	// A held financial source protects its counterparty contact and the original
	// lead as well. Match authoritative relationships, not caller project metadata.
	if kind == "customer" || kind == "lead" {
		customers := `SELECT id FROM customers WHERE id=?`
		if kind == "lead" {
			customers = `SELECT id FROM customers WHERE source_lead_id=? UNION SELECT customer_id FROM bookings WHERE source_lead_id=?`
		}
		links := `SELECT 'customer' AS kind,id FROM customers WHERE id IN (` + customers + `) UNION SELECT 'booking',id FROM bookings WHERE customer_id IN (` + customers + `) UNION SELECT 'invoice',id FROM invoices WHERE customer_id IN (` + customers + `) UNION SELECT 'payment',id FROM payments WHERE customer_id IN (` + customers + `)`
		if kind == "lead" {
			links += ` UNION SELECT 'booking',id FROM bookings WHERE source_lead_id=?`
		}
		if kind == "customer" {
			links += ` UNION SELECT 'lead',source_lead_id FROM customers WHERE id=?`
		}
		query += ` OR (status='active' AND EXISTS(SELECT 1 FROM (` + links + `) linked WHERE linked.kind=retention_holds.record_type AND linked.id=retention_holds.record_id))`
		for i := 0; i < strings.Count(links, "?"); i++ {
			args = append(args, id)
		}
	}
	if kind == "supplier-contact" || kind == "contractor-contact" {
		parent := strings.TrimSuffix(kind, "-contact")
		parentID := `SELECT ` + parent + `_id FROM ` + parent + `_contacts WHERE id=?`
		links := `SELECT 'invoice' AS kind,id FROM invoices WHERE ` + parent + `_id=(` + parentID + `) UNION SELECT 'payment',id FROM payments WHERE ` + parent + `_id=(` + parentID + `)`
		if parent == "supplier" {
			links += ` UNION SELECT 'purchase-order',id FROM purchase_orders WHERE supplier_id=(` + parentID + `)`
		} else {
			links += ` UNION SELECT 'work-order',id FROM work_orders WHERE contractor_id=(` + parentID + `)`
		}
		query += ` OR (status='active' AND EXISTS(SELECT 1 FROM (` + links + `) linked WHERE linked.kind=retention_holds.record_type AND linked.id=retention_holds.record_id))`
		for i := 0; i < strings.Count(links, "?"); i++ {
			args = append(args, id)
		}
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var v string
		if err = rows.Scan(&v); err != nil {
			return nil, err
		}
		ids = append(ids, v)
	}
	return ids, rows.Err()
}
func stewardNoHold(ctx context.Context, tx contracts.Transaction, kind, id string) error {
	holds, err := stewardHolds(ctx, tx, kind, id)
	if err != nil {
		return err
	}
	if len(holds) > 0 {
		return transactions.Conflict("RETENTION_HOLD")
	}
	return nil
}

func (s Service) stewardCreatePrivacy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		Kind        string `json:"kind"`
		Reason      string `json:"reason"`
	}
	if !transport.DecodeJSON(w, r, &in) {
		return
	}
	if stewardSubjectTable(in.SubjectType) == "" || !stewardReason(in.Reason) || (in.Kind != "access" && in.Kind != "correction" && in.Kind != "anonymization") {
		respond(w, r, nil, invalid())
		return
	}
	id := transactions.NewID()
	err := s.write(r.Context(), "privacy.manage", "", "privacy-request.create", "privacy-request", id, func(ctx context.Context, tx contracts.Transaction) error {
		if err := stewardExists(ctx, tx, in.SubjectType, in.SubjectID); err != nil {
			return err
		}
		actor, err := s.actor(ctx, tx)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO privacy_requests(id,subject_type,subject_id,kind,reason,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, id, in.SubjectType, in.SubjectID, in.Kind, in.Reason, actor, transactions.Timestamp(s.now()), transactions.Timestamp(s.now()))
		return err
	})
	s.stewardReply(w, r, "privacy-requests", id, err)
}
func stewardValidateCorrections(kind string, c *stewardCorrections) error {
	if c == nil || (c.Name == nil && c.Email == nil && c.Phone == nil) {
		return invalid()
	}
	if c.Name != nil && (strings.TrimSpace(*c.Name) == "" || len(*c.Name) > 200) {
		return invalid()
	}
	if c.Email != nil {
		if len(*c.Email) > 254 {
			return invalid()
		}
		if *c.Email != "" {
			a, e := mail.ParseAddress(*c.Email)
			if e != nil || a.Address != *c.Email {
				return invalid()
			}
		}
	}
	if c.Phone != nil && len(*c.Phone) > 50 {
		return invalid()
	}
	if kind == "user" && (c.Phone != nil || (c.Email != nil && *c.Email == "")) {
		return invalid()
	}
	return nil
}
func (s Service) stewardReviewPrivacy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version     int64               `json:"version"`
		Decision    string              `json:"decision"`
		Reason      string              `json:"reason"`
		Corrections *stewardCorrections `json:"corrections"`
	}
	if !transport.DecodeJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	action := "privacy-request.approve"
	if in.Decision == "reject" {
		action = "privacy-request.reject"
	}
	err := s.write(r.Context(), "privacy.manage", "", action, "privacy-request", id, func(ctx context.Context, tx contracts.Transaction) error {
		if !stewardReason(in.Reason) || (in.Decision != "approve" && in.Decision != "reject") {
			return invalid()
		}
		var kind, subject, sid, status string
		var version int64
		if err := tx.QueryRowContext(ctx, `SELECT kind,subject_type,subject_id,status,version FROM privacy_requests WHERE id=?`, id).Scan(&kind, &subject, &sid, &status, &version); err != nil {
			return err
		}
		if err := transactions.CheckVersion(version, in.Version); err != nil {
			return err
		}
		if status != "submitted" {
			return transactions.Conflict("INVALID_TRANSITION")
		}
		var name, email, phone *string
		if in.Decision == "approve" && kind == "correction" {
			if err := stewardValidateCorrections(subject, in.Corrections); err != nil {
				return err
			}
			name, email, phone = in.Corrections.Name, in.Corrections.Email, in.Corrections.Phone
		} else if in.Corrections != nil {
			return invalid()
		}
		if in.Decision == "approve" && kind != "access" {
			if err := stewardNoHold(ctx, tx, subject, sid); err != nil {
				return err
			}
		}
		actor, err := s.actor(ctx, tx)
		if err != nil {
			return err
		}
		next := "approved"
		if in.Decision == "reject" {
			next = "rejected"
		}
		return stewardChanged(tx.ExecContext(ctx, `UPDATE privacy_requests SET status=?,reviewed_by=?,reviewed_at=?,review_reason=?,correction_name=?,correction_email=?,correction_phone=?,version=version+1,updated_at=? WHERE id=? AND version=?`, next, actor, transactions.Timestamp(s.now()), in.Reason, name, email, phone, transactions.Timestamp(s.now()), id, in.Version))
	})
	s.stewardReply(w, r, "privacy-requests", id, err)
}

// Contact-only updates retain identities, immutable attribution, transaction rows,
// and attachments. Staff anonymization is restricted to deactivated accounts.
func stewardContact(ctx context.Context, tx contracts.Transaction, kind, id string, c stewardCorrections, anonymize bool, now time.Time) error {
	if err := stewardExists(ctx, tx, kind, id); err != nil {
		return err
	}
	if err := stewardNoHold(ctx, tx, kind, id); err != nil {
		return err
	}
	if anonymize {
		name, email, phone := "Retained contact", "", ""
		c = stewardCorrections{&name, &email, &phone}
	}
	table := stewardSubjectTable(kind)
	nameCol, emailCol, phoneCol := "name", "email", "phone"
	where := "id=?"
	tail := []any{id}
	meta := ""
	switch kind {
	case "user":
		nameCol = "display_name"
		phoneCol = ""
		if anonymize {
			var status string
			if err := tx.QueryRowContext(ctx, "SELECT status FROM users WHERE id=?", id).Scan(&status); err != nil {
				return err
			}
			if status != "deactivated" {
				return transactions.Conflict("ACTIVE_SUBJECT")
			}
			email := "retained-" + id + "@example.invalid"
			c.Email = &email
			c.Phone = nil
		}
		meta = ",version=version+1,access_revision=access_revision+1,updated_at=?"
		tail = append([]any{transactions.Timestamp(now)}, tail...)
	case "lead":
		nameCol, emailCol, phoneCol = "contact_name", "contact_email", "contact_phone"
		meta = ",version=version+1,updated_at=?"
		tail = append([]any{transactions.Timestamp(now)}, tail...)
	case "customer":
		// The customer's business name is retained as transaction attribution.
		table = "customer_contacts"
		where = "customer_id=?"
	}
	sets := []string{}
	args := []any{}
	if c.Name != nil {
		sets = append(sets, nameCol+"=?")
		args = append(args, *c.Name)
	}
	if c.Email != nil {
		sets = append(sets, emailCol+"=?", "normalized_email=?")
		args = append(args, *c.Email, strings.ToLower(strings.TrimSpace(*c.Email)))
	}
	if c.Phone != nil && phoneCol != "" {
		sets = append(sets, phoneCol+"=?", "normalized_phone=?")
		args = append(args, *c.Phone, strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(*c.Phone))
	}
	if len(sets) == 0 {
		return invalid()
	}
	args = append(args, tail...)
	_, err := tx.ExecContext(ctx, "UPDATE "+table+" SET "+strings.Join(sets, ",")+meta+" WHERE "+where, args...)
	return err
}
func (s Service) stewardExecutePrivacy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int64 `json:"version"`
	}
	if !transport.DecodeJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	err := s.write(r.Context(), "privacy.manage", "", "privacy-request.execute", "privacy-request", id, func(ctx context.Context, tx contracts.Transaction) error {
		var subject, sid, kind, status string
		var version int64
		var c stewardCorrections
		if err := tx.QueryRowContext(ctx, `SELECT subject_type,subject_id,kind,status,version,correction_name,correction_email,correction_phone FROM privacy_requests WHERE id=?`, id).Scan(&subject, &sid, &kind, &status, &version, &c.Name, &c.Email, &c.Phone); err != nil {
			return err
		}
		if err := transactions.CheckVersion(version, in.Version); err != nil {
			return err
		}
		if status != "approved" {
			return transactions.Conflict("INVALID_TRANSITION")
		}
		if kind != "access" {
			if err := stewardContact(ctx, tx, subject, sid, c, kind == "anonymization", s.now()); err != nil {
				return err
			}
		}
		return stewardChanged(tx.ExecContext(ctx, `UPDATE privacy_requests SET status='completed',completed_at=?,updated_at=?,version=version+1 WHERE id=? AND version=?`, transactions.Timestamp(s.now()), transactions.Timestamp(s.now()), id, in.Version))
	})
	s.stewardReply(w, r, "privacy-requests", id, err)
}

func (s Service) stewardCreateHold(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RecordType string  `json:"recordType"`
		RecordID   string  `json:"recordId"`
		Reason     string  `json:"reason"`
		ReviewAt   *string `json:"reviewAt"`
	}
	if !transport.DecodeJSON(w, r, &in) {
		return
	}
	id := transactions.NewID()
	err := s.write(r.Context(), "retention.manage", "", "legal-hold.create", "legal-hold", id, func(ctx context.Context, tx contracts.Transaction) error {
		if !stewardReason(in.Reason) {
			return invalid()
		}
		if in.ReviewAt != nil {
			if _, e := time.Parse(time.RFC3339, *in.ReviewAt); e != nil {
				return invalid()
			}
		}
		if err := stewardExists(ctx, tx, in.RecordType, in.RecordID); err != nil {
			return err
		}
		actor, err := s.actor(ctx, tx)
		if err != nil {
			return err
		}
		var project any
		if in.RecordType == "project" {
			project = in.RecordID
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO retention_holds(id,project_id,record_type,record_id,reason,review_at,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, id, project, in.RecordType, in.RecordID, in.Reason, in.ReviewAt, actor, transactions.Timestamp(s.now()), transactions.Timestamp(s.now()))
		return err
	})
	s.stewardReply(w, r, "retention-holds", id, err)
}
func (s Service) stewardReleaseHold(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int64  `json:"version"`
		Reason  string `json:"reason"`
	}
	if !transport.DecodeJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	err := s.write(r.Context(), "retention.manage", "", "legal-hold.release", "legal-hold", id, func(ctx context.Context, tx contracts.Transaction) error {
		if !stewardReason(in.Reason) || in.Version < 1 {
			return invalid()
		}
		actor, err := s.actor(ctx, tx)
		if err != nil {
			return err
		}
		return stewardChanged(tx.ExecContext(ctx, `UPDATE retention_holds SET status='released',released_by=?,released_at=?,release_reason=?,updated_at=?,version=version+1 WHERE id=? AND version=? AND status='active'`, actor, transactions.Timestamp(s.now()), in.Reason, transactions.Timestamp(s.now()), id, in.Version))
	})
	s.stewardReply(w, r, "retention-holds", id, err)
}

// All timestamps are read from authoritative rows at preview AND execution. The
// immutable schema intentionally keeps business/audit evidence even after expiry.
func stewardEligible(ctx context.Context, tx contracts.Transaction, kind, id string) (time.Time, string, error) {
	query, years, days, action := "", 0, 0, "retain"
	switch kind {
	case "lead":
		query = `SELECT last_activity_at FROM leads l WHERE id=? AND stage<>'booked' AND NOT EXISTS(SELECT 1 FROM customers c WHERE c.source_lead_id=l.id)`
		years = 2
		action = "anonymize"
	case "user":
		query = `SELECT deactivated_at FROM users WHERE id=? AND status='deactivated'`
		days = 90
		action = "anonymize"
	case "session":
		query = `SELECT min(absolute_expires_at,idle_expires_at,coalesce(revoked_at,absolute_expires_at)) FROM sessions WHERE id=?`
		days = 30
		action = "purge"
	case "password-token":
		query = `SELECT min(expires_at,coalesce(consumed_at,expires_at),coalesce(revoked_at,expires_at)) FROM password_tokens WHERE id=?`
		days = 30
		action = "purge"
	case "project":
		query = `SELECT closed_at FROM projects WHERE id=? AND status IN ('completed','archived')`
		years = 7
	case "audit-event":
		var class string
		if err := tx.QueryRowContext(ctx, "SELECT retention_class FROM audit_events WHERE id=?", id).Scan(&class); err != nil {
			return time.Time{}, "", err
		}
		if class == "authentication" {
			query = `SELECT occurred_at FROM audit_events WHERE id=?`
			years = 1
		} else {
			query = `SELECT p.closed_at FROM audit_events a JOIN projects p ON p.id=a.project_id WHERE a.id=? AND p.status IN ('completed','archived')`
			years = 7
		}
	case "document", "invoice", "payment", "booking", "purchase-order", "work-order":
		tables := map[string]string{"document": "documents", "invoice": "invoices", "payment": "payments", "booking": "bookings", "purchase-order": "purchase_orders", "work-order": "work_orders"}
		query = "SELECT p.closed_at FROM " + tables[kind] + " r JOIN projects p ON p.id=r.project_id WHERE r.id=? AND p.status IN ('completed','archived')"
		years = 7
	default:
		return time.Time{}, "", invalid()
	}
	var stamp sql.NullString
	if err := tx.QueryRowContext(ctx, query, id).Scan(&stamp); err != nil {
		return time.Time{}, "", err
	}
	if !stamp.Valid {
		return time.Time{}, "", sql.ErrNoRows
	}
	at, err := time.Parse(time.RFC3339Nano, stamp.String)
	if err != nil {
		return time.Time{}, "", err
	}
	return at.AddDate(years, 0, days), action, nil
}
func stewardCandidates(ctx context.Context, tx contracts.Transaction, asOf time.Time, project string) ([]stewardCandidate, error) {
	query := `SELECT 'lead',id,project_id FROM leads UNION ALL SELECT 'user',id,NULL FROM users UNION ALL SELECT 'session',id,NULL FROM sessions UNION ALL SELECT 'password-token',id,NULL FROM password_tokens UNION ALL SELECT 'project',id,id FROM projects UNION ALL SELECT 'audit-event',id,project_id FROM audit_events UNION ALL SELECT 'document',id,project_id FROM documents UNION ALL SELECT 'invoice',id,project_id FROM invoices UNION ALL SELECT 'payment',id,project_id FROM payments UNION ALL SELECT 'booking',id,project_id FROM bookings UNION ALL SELECT 'purchase-order',id,project_id FROM purchase_orders UNION ALL SELECT 'work-order',id,project_id FROM work_orders`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	type ref struct{ kind, id string }
	refs := []ref{}
	for rows.Next() {
		var k, id string
		var p sql.NullString
		if err = rows.Scan(&k, &id, &p); err != nil {
			rows.Close()
			return nil, err
		}
		if project == "" || (p.Valid && p.String == project) {
			refs = append(refs, ref{k, id})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []stewardCandidate{}
	for _, r := range refs {
		at, action, e := stewardEligible(ctx, tx, r.kind, r.id)
		if e == sql.ErrNoRows {
			continue
		}
		if e != nil {
			return nil, e
		}
		if at.After(asOf) {
			continue
		}
		holds, e := stewardHolds(ctx, tx, r.kind, r.id)
		if e != nil {
			return nil, e
		}
		if len(holds) > 0 {
			action = "retain"
		}
		out = append(out, stewardCandidate{ID: transactions.NewID(), RecordType: r.kind, RecordID: r.id, EligibleAt: transactions.Timestamp(at), Action: action, HoldIDs: holds})
	}
	return out, nil
}
func (s Service) stewardCreateReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AsOfDate  string  `json:"asOfDate"`
		ProjectID *string `json:"projectId"`
	}
	if !transport.DecodeJSON(w, r, &in) {
		return
	}
	id := transactions.NewID()
	err := s.write(r.Context(), "retention.manage", "", "retention-review.create", "retention-review", id, func(ctx context.Context, tx contracts.Transaction) error {
		date, err := time.Parse("2006-01-02", in.AsOfDate)
		if err != nil || date.After(s.now()) {
			return invalid()
		}
		project := ""
		if in.ProjectID != nil {
			project = *in.ProjectID
			if err = stewardExists(ctx, tx, "project", project); err != nil {
				return err
			}
		}
		candidates, err := stewardCandidates(ctx, tx, date, project)
		if err != nil {
			return err
		}
		actor, err := s.actor(ctx, tx)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO retention_reviews(id,project_id,as_of_date,status,created_by,created_at,updated_at) VALUES(?,?,?,'reviewed',?,?,?)`, id, stewardNullable(project), in.AsOfDate, actor, transactions.Timestamp(s.now()), transactions.Timestamp(s.now()))
		if err != nil {
			return err
		}
		for _, c := range candidates {
			if _, err = tx.ExecContext(ctx, `INSERT INTO retention_review_candidates(id,review_id,record_type,record_id,eligible_at,action) VALUES(?,?,?,?,?,?)`, c.ID, id, c.RecordType, c.RecordID, c.EligibleAt, c.Action); err != nil {
				return err
			}
			for _, h := range c.HoldIDs {
				if _, err = tx.ExecContext(ctx, `INSERT INTO retention_candidate_holds(candidate_id,hold_id) VALUES(?,?)`, c.ID, h); err != nil {
					return err
				}
			}
		}
		return nil
	})
	s.stewardReply(w, r, "retention-reviews", id, err)
}
func (s Service) stewardExecuteReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version   int64    `json:"version"`
		RecordIDs []string `json:"recordIds"`
		Reason    string   `json:"reason"`
	}
	if !transport.DecodeJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	err := s.write(r.Context(), "retention.manage", "", "retention-review.execute", "retention-review", id, func(ctx context.Context, tx contracts.Transaction) error {
		if !stewardReason(in.Reason) || len(in.RecordIDs) == 0 {
			return invalid()
		}
		var version int64
		var status, date string
		if err := tx.QueryRowContext(ctx, `SELECT version,status,as_of_date FROM retention_reviews WHERE id=?`, id).Scan(&version, &status, &date); err != nil {
			return err
		}
		if err := transactions.CheckVersion(version, in.Version); err != nil {
			return err
		}
		if status != "reviewed" {
			return transactions.Conflict("INVALID_TRANSITION")
		}
		asOf, err := time.Parse("2006-01-02", date)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, rid := range in.RecordIDs {
			if seen[rid] {
				return invalid()
			}
			seen[rid] = true
			// IDs are source record IDs per the published contract. Ambiguous matches fail.
			var count int
			if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM retention_review_candidates WHERE review_id=? AND record_id=?`, id, rid).Scan(&count); err != nil {
				return err
			}
			if count != 1 {
				return invalid()
			}
			var cid, kind, oldAction, oldAt string
			if err = tx.QueryRowContext(ctx, `SELECT id,record_type,action,eligible_at FROM retention_review_candidates WHERE review_id=? AND record_id=? AND executed_at IS NULL`, id, rid).Scan(&cid, &kind, &oldAction, &oldAt); err != nil {
				return err
			}
			at, action, e := stewardEligible(ctx, tx, kind, rid)
			if e != nil {
				return transactions.Conflict("RETENTION_CHANGED")
			}
			if at.After(asOf) || at.After(s.now()) || oldAt != transactions.Timestamp(at) {
				return transactions.Conflict("RETENTION_CHANGED")
			}
			if oldAction != "retain" {
				if err = stewardNoHold(ctx, tx, kind, rid); err != nil {
					return err
				}
				if action != oldAction {
					return transactions.Conflict("RETENTION_CHANGED")
				}
				switch action {
				case "anonymize":
					if err = stewardContact(ctx, tx, kind, rid, stewardCorrections{}, true, s.now()); err != nil {
						return err
					}
				case "purge":
					table := "sessions"
					if kind == "password-token" {
						table = "password_tokens"
					}
					if err = stewardChanged(tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE id=?", rid)); err != nil {
						return err
					}
				}
			}
			if _, err = tx.ExecContext(ctx, `UPDATE retention_review_candidates SET executed_at=? WHERE id=?`, transactions.Timestamp(s.now()), cid); err != nil {
				return err
			}
		}
		actor, err := s.actor(ctx, tx)
		if err != nil {
			return err
		}
		return stewardChanged(tx.ExecContext(ctx, `UPDATE retention_reviews SET status='executed',executed_by=?,executed_at=?,execution_reason=?,updated_at=?,version=version+1 WHERE id=? AND version=?`, actor, transactions.Timestamp(s.now()), in.Reason, transactions.Timestamp(s.now()), id, in.Version))
	})
	s.stewardReply(w, r, "retention-reviews", id, err)
}
