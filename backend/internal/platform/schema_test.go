package platform_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"buildestate/backend/internal/platform/database"
)

func schemaDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "schema.sqlite"))
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(ctx, db); err != nil { t.Fatal(err) }
	if err := database.Migrate(ctx, db); err != nil { t.Fatalf("repeat migration: %v", err) }
	return db
}

func schemaExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil { t.Fatalf("SQL failed: %s: %v", query, err) }
}

func schemaReject(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query); err == nil { t.Fatalf("invalid write accepted: %s", query) }
}

func seedSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	schemaExec(t, db, `INSERT INTO users(id,display_name,email,normalized_email,role_code) VALUES ('u','Operator','schema@example.invalid','schema@example.invalid','administrator')`)
	for _, id := range []string{"p", "other"} {
		schemaExec(t, db, `INSERT INTO projects(id,code,name,address,project_type,start_date,end_date,manager_id,created_by) VALUES (?,?,?,'Address','residential','2026-01-01','2027-01-01','u','u')`, id,id,id)
		schemaExec(t, db, `INSERT INTO sites(id,project_id,code,name,address,created_by) VALUES (?,?,'S','Site','Address','u')`, "s-"+id,id)
	}
	schemaExec(t, db, `INSERT INTO properties(id,project_id,site_id,unit_code,block,property_type,area,area_unit,asking_price_minor,description,construction_status,created_by) VALUES ('unit','p','s-p','U1','A','apartment',100000,'sq-m',100000,'Unit','planned','u')`)
	schemaExec(t, db, `INSERT INTO customers(id,project_id,name,created_by) VALUES ('buyer','p','Buyer','u')`)
}

func TestSchemaBaselineTablesAndForeignKeys(t *testing.T) {
	db := schemaDB(t)
	tables := strings.Fields(`users roles user_projects sessions password_tokens rate_limit_buckets projects sites project_overrides milestones tasks plan_revisions site_logs site_log_workers review_decisions inspections inspection_items issues issue_events materials material_site_minimums suppliers purchase_orders purchase_order_lines purchase_order_revisions goods_receipts goods_receipt_lines stock_transactions stock_entries stock_adjustments contractors contractor_projects work_orders work_order_milestones work_order_revisions budgets budget_revisions budget_lines cost_categories expenses cost_sources cost_recognitions invoices invoice_lines invoice_sources duplicate_reviews payments allocations financial_reversals refund_proposals properties property_price_history property_images leads lead_activities site_visits customers reservations bookings unit_claims installments cancellation_terms cancellation_bands cancellations sale_milestones payment_verifications handover_items handovers documents upload_batches audit_events idempotency_records company_settings reference_values policy_acknowledgments privacy_requests retention_holds retention_reviews reconciliations reconciliation_lines reconciliation_reviews report_snapshots backup_runs`)
	for _, table := range tables {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_schema WHERE type='table' AND name=?`, table).Scan(&name); err != nil { t.Errorf("missing baseline table %s: %v",table,err) }
	}
	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 { t.Fatalf("foreign keys: %d, %v",foreignKeys,err) }
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil { t.Fatal(err) }
	defer rows.Close()
	if rows.Next() { t.Fatal("foreign key violation after migration") }
	if err := rows.Err(); err != nil { t.Fatal(err) }
}

func TestSchemaClaimsAreExclusiveAndProjectMatched(t *testing.T) {
	db := schemaDB(t); seedSchema(t,db)
	for _, id := range []string{"r1","r2"} {
		schemaExec(t,db,`INSERT INTO reservations(id,project_id,property_id,customer_id,expires_at,created_by) VALUES (?,'p','unit','buyer','2027-01-01T00:00:00.000Z','u')`,id)
	}
	schemaExec(t,db,`INSERT INTO unit_claims(property_id,project_id,reservation_id,acquired_at) VALUES ('unit','p','r1','2026-01-01T00:00:00.000Z')`)
	schemaReject(t,db,`INSERT INTO unit_claims(property_id,project_id,reservation_id,acquired_at) VALUES ('unit','p','r2','2026-01-01T00:00:00.000Z')`)
	schemaReject(t,db,`UPDATE unit_claims SET reservation_id=NULL`)
	schemaReject(t,db,`UPDATE unit_claims SET project_id='other'`)
	schemaReject(t,db,`DELETE FROM unit_claims`)
	schemaReject(t,db,`UPDATE properties SET availability='unavailable' WHERE id='unit'`)
	schemaExec(t,db,`UPDATE reservations SET status='expired' WHERE id='r1'`)
	schemaExec(t,db,`DELETE FROM unit_claims WHERE reservation_id='r1'`)
	schemaExec(t,db,`INSERT INTO unit_claims(property_id,project_id,reservation_id,acquired_at) VALUES ('unit','p','r2','2026-01-01T00:00:00.000Z')`)
	schemaReject(t,db,`INSERT INTO reservations(id,project_id,property_id,customer_id,expires_at,created_by) VALUES ('bad','other','unit','buyer','2027-01-01','u')`)
}

func TestSchemaTypedDocumentsAndImmutableAudit(t *testing.T) {
	db := schemaDB(t); seedSchema(t,db)
	schemaExec(t,db,`INSERT INTO upload_batches(id,project_id,status,file_count,total_bytes,created_by) VALUES ('batch','p','committed',1,10,'u')`)
	schemaExec(t,db,`INSERT INTO documents(id,project_id,upload_batch_id,storage_key,original_name,mime_type,size_bytes,sha256,status,classification,created_by) VALUES ('doc','p','batch','random-key','evidence.pdf','application/pdf',10,?,'active','restricted','u')`,strings.Repeat("a",64))
	schemaExec(t,db,`INSERT INTO document_project_links(id,document_id,project_id,parent_id,created_by) VALUES ('link','doc','p','p','u')`)
	schemaReject(t,db,`INSERT INTO document_project_links(id,document_id,project_id,parent_id,created_by) VALUES ('cross','doc','other','other','u')`)
	schemaReject(t,db,`INSERT INTO document_invoice_links(id,document_id,project_id,parent_id,created_by) VALUES ('missing','doc','p','unknown-invoice','u')`)
	schemaReject(t,db,`UPDATE documents SET classification='operational' WHERE id='doc'`)
	schemaReject(t,db,`DELETE FROM document_project_links`)
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM document_links WHERE document_id='doc' AND parent_type='project' AND entity_id='p'`).Scan(&count); err != nil || count!=1 { t.Fatalf("typed registry: %d %v",count,err) }
	schemaExec(t,db,`INSERT INTO audit_events(id,actor_id,occurred_at,action,project_id,record_type,record_id,request_id,changes_json,retention_class) VALUES ('audit','u','2026-01-01T00:00:00.000Z','create','p','project','p','request','{}','business')`)
	schemaReject(t,db,`UPDATE audit_events SET action='hidden'`)
	schemaReject(t,db,`DELETE FROM audit_events`)
}

func TestSchemaExactNumbersLifecycleAndHistory(t *testing.T) {
	db := schemaDB(t); seedSchema(t,db)
	schemaReject(t,db,`UPDATE projects SET status='invented' WHERE id='p'`)
	schemaReject(t,db,`UPDATE projects SET end_date='2020-01-01' WHERE id='p'`)
	schemaReject(t,db,`UPDATE properties SET asking_price_minor=1.25 WHERE id='unit'`)
	schemaReject(t,db,`UPDATE properties SET asking_price_minor=-1 WHERE id='unit'`)
	schemaExec(t,db,`INSERT INTO bookings(id,project_id,property_id,customer_id,sales_executive_id,agreed_price_minor,deposit_required_minor,agreement_date,handover_required_minor,created_by) VALUES ('booking','p','unit','buyer','u',100,10,'2026-01-01',100,'u')`)
	schemaExec(t,db,`INSERT INTO cancellation_terms(id,project_id,booking_id,revision,reason,proposed_by,created_by) VALUES ('terms','p','booking',1,'Initial','u','u')`)
	schemaExec(t,db,`INSERT INTO cancellation_bands(id,terms_id,from_date,to_date,deduction_type,fixed_minor) VALUES ('band','terms','2026-01-01','2027-01-01','fixed',10)`)
	schemaReject(t,db,`INSERT INTO cancellation_bands(id,terms_id,from_date,to_date,deduction_type,percentage_bps) VALUES ('overlap','terms','2026-06-01',NULL,'percentage',100)`)
	schemaReject(t,db,`UPDATE cancellation_bands SET percentage_bps=100 WHERE id='band'`)
	schemaReject(t,db,`UPDATE cancellation_terms SET status='approved',decided_by='u' WHERE id='terms'`)
	schemaExec(t,db,`INSERT INTO stock_transactions(id,project_id,type,reason,actor_id,occurred_at) VALUES ('tx','p','transfer','Move','u','2026-01-01T00:00:00.000Z')`)
	schemaReject(t,db,`UPDATE stock_transactions SET reason='rewrite'`)
	schemaReject(t,db,`DELETE FROM stock_transactions`)
	schemaExec(t,db,`INSERT INTO materials(id,code,name,unit,minimum_quantity,created_by) VALUES ('m','M','Material','kg',0,'u')`)
	schemaReject(t,db,`INSERT INTO stock_entries(id,project_id,transaction_id,site_id,material_id,unit,quantity,source_line_id,effect) VALUES ('negative','p','tx','s-p','m','kg',-1,'line','out')`)
}
