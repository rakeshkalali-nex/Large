CREATE TABLE upload_batches (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), status TEXT NOT NULL CHECK(status IN ('quarantine','validated','committed','failed')),
 file_count INTEGER NOT NULL CHECK(file_count BETWEEN 1 AND 10), total_bytes INTEGER NOT NULL CHECK(total_bytes BETWEEN 1 AND 200000000),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE documents (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), upload_batch_id TEXT NOT NULL, storage_key TEXT NOT NULL UNIQUE,
 original_name TEXT NOT NULL, mime_type TEXT NOT NULL CHECK(mime_type IN ('application/pdf','image/jpeg','image/png','image/webp')), size_bytes INTEGER NOT NULL CHECK(size_bytes BETWEEN 1 AND 20000000), sha256 TEXT NOT NULL CHECK(length(sha256)=64),
 status TEXT NOT NULL DEFAULT 'quarantine' CHECK(status IN ('quarantine','active','archived')), classification TEXT NOT NULL CHECK(classification IN ('operational','financial','sales','restricted')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(upload_batch_id,project_id) REFERENCES upload_batches(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX documents_list ON documents(project_id,status,created_at,id);
-- Each registry entry identifies an actual FK table, never a free-form parent ID.
-- The following migration creates those tables and their insert/delete-maintained registry rows.
CREATE TABLE document_parent_types (code TEXT PRIMARY KEY, link_table TEXT NOT NULL UNIQUE, classification TEXT NOT NULL CHECK(classification IN ('operational','financial','sales','restricted'))) STRICT;
CREATE TABLE document_links (
 id TEXT PRIMARY KEY, document_id TEXT NOT NULL, project_id TEXT NOT NULL, parent_type TEXT NOT NULL REFERENCES document_parent_types(code),
 classification TEXT NOT NULL CHECK(classification IN ('operational','financial','sales','restricted')),
 FOREIGN KEY(document_id,project_id) REFERENCES documents(id,project_id), UNIQUE(id,document_id,project_id,parent_type)
) STRICT;
CREATE INDEX document_links_document ON document_links(document_id,parent_type);
CREATE TABLE property_images (
 property_id TEXT NOT NULL, project_id TEXT NOT NULL, document_id TEXT NOT NULL, alt_text TEXT NOT NULL, sort_order INTEGER NOT NULL CHECK(sort_order >= 0),
 FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id), FOREIGN KEY(document_id,project_id) REFERENCES documents(id,project_id), PRIMARY KEY(property_id,document_id), UNIQUE(property_id,sort_order)
) STRICT;
CREATE TABLE audit_events (
 id TEXT PRIMARY KEY, actor_id TEXT REFERENCES users(id), occurred_at TEXT NOT NULL, action TEXT NOT NULL, project_id TEXT REFERENCES projects(id), record_type TEXT NOT NULL, record_id TEXT NOT NULL, request_id TEXT NOT NULL, reason TEXT,
 changes_json TEXT NOT NULL CHECK(json_valid(changes_json) AND json_type(changes_json) IN ('array','object')), origin_digest TEXT, retention_class TEXT NOT NULL CHECK(retention_class IN ('authentication','business'))
) STRICT;
CREATE INDEX audit_project_time ON audit_events(project_id,occurred_at,id);
CREATE INDEX audit_actor_time ON audit_events(actor_id,occurred_at,id);
CREATE INDEX audit_record ON audit_events(record_type,record_id,occurred_at);
CREATE TABLE idempotency_records (
 principal_id TEXT NOT NULL, operation TEXT NOT NULL, key TEXT NOT NULL, request_digest TEXT NOT NULL, response_status INTEGER NOT NULL CHECK(response_status BETWEEN 200 AND 499), result_type TEXT NOT NULL, result_id TEXT NOT NULL, project_id TEXT REFERENCES projects(id), created_at TEXT NOT NULL,
 PRIMARY KEY(principal_id,operation,key)
) STRICT;
CREATE TABLE company_settings (
 id TEXT PRIMARY KEY CHECK(id='company'), company_name TEXT NOT NULL, wordmark TEXT NOT NULL, logo_document_id TEXT REFERENCES documents(id),
 currency_code TEXT CHECK(length(currency_code)=3 AND currency_code=upper(currency_code)), currency_minor_digits INTEGER CHECK(currency_minor_digits BETWEEN 0 AND 4), currency_locked INTEGER NOT NULL DEFAULT 0 CHECK(currency_locked IN (0,1)),
 timezone TEXT NOT NULL DEFAULT 'UTC', timezone_confirmed INTEGER NOT NULL DEFAULT 0 CHECK(timezone_confirmed IN (0,1)), recovery_instructions TEXT NOT NULL, recovery_contact_label TEXT NOT NULL, reservation_hours INTEGER NOT NULL DEFAULT 48 CHECK(reservation_hours > 0),
 notice_version TEXT, retention_schedule_version TEXT, operational_policy_confirmed_at TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 CHECK((currency_code IS NULL)=(currency_minor_digits IS NULL)), CHECK(currency_locked=0 OR currency_code IS NOT NULL)
) STRICT;
CREATE TABLE operating_countries (settings_id TEXT NOT NULL REFERENCES company_settings(id), country_code TEXT NOT NULL CHECK(length(country_code)=2), PRIMARY KEY(settings_id,country_code)) STRICT;
CREATE TABLE reference_values (
 id TEXT PRIMARY KEY, project_id TEXT REFERENCES projects(id), key TEXT NOT NULL, value TEXT NOT NULL, label TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0), UNIQUE(project_id,key,value)
) STRICT;
CREATE UNIQUE INDEX company_reference_values ON reference_values(key,value) WHERE project_id IS NULL;
CREATE TABLE property_types (
 id TEXT PRIMARY KEY, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE policy_acknowledgments (
 id TEXT PRIMARY KEY, subject_type TEXT NOT NULL CHECK(subject_type IN ('user','lead','customer','supplier-contact','contractor-contact')), subject_id TEXT NOT NULL, notice_version TEXT NOT NULL, acknowledged_at TEXT NOT NULL,
 consent_required INTEGER NOT NULL CHECK(consent_required IN (0,1)), consent_given INTEGER CHECK(consent_given IN (0,1)), recorded_by_reason TEXT NOT NULL, created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL,
 CHECK(consent_required=0 OR consent_given IS NOT NULL)
) STRICT;
CREATE TABLE privacy_requests (
 id TEXT PRIMARY KEY, subject_type TEXT NOT NULL CHECK(subject_type IN ('user','lead','customer','supplier-contact','contractor-contact')), subject_id TEXT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('access','correction','anonymization')), reason TEXT NOT NULL CHECK(length(trim(reason)) > 0), status TEXT NOT NULL DEFAULT 'submitted' CHECK(status IN ('submitted','approved','rejected','completed')),
 reviewed_by TEXT REFERENCES users(id), reviewed_at TEXT, review_reason TEXT, correction_name TEXT, correction_email TEXT, correction_phone TEXT, completed_at TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE retention_holds (
 id TEXT PRIMARY KEY, project_id TEXT REFERENCES projects(id), record_type TEXT NOT NULL, record_id TEXT NOT NULL, reason TEXT NOT NULL CHECK(length(trim(reason)) > 0), review_at TEXT,
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','released')), released_by TEXT REFERENCES users(id), released_at TEXT, release_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE INDEX retention_hold_record ON retention_holds(record_type,record_id,status);
CREATE TABLE retention_reviews (
 id TEXT PRIMARY KEY, project_id TEXT REFERENCES projects(id), as_of_date TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','reviewed','executed')),
 executed_by TEXT REFERENCES users(id), executed_at TEXT, execution_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE retention_review_candidates (
 id TEXT PRIMARY KEY, review_id TEXT NOT NULL REFERENCES retention_reviews(id), record_type TEXT NOT NULL, record_id TEXT NOT NULL, eligible_at TEXT NOT NULL, action TEXT NOT NULL CHECK(action IN ('retain','anonymize','purge')), executed_at TEXT, UNIQUE(review_id,record_type,record_id)
) STRICT;
CREATE TABLE retention_candidate_holds (candidate_id TEXT NOT NULL REFERENCES retention_review_candidates(id), hold_id TEXT NOT NULL REFERENCES retention_holds(id), PRIMARY KEY(candidate_id,hold_id)) STRICT;
CREATE TABLE reconciliations (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), period_start TEXT NOT NULL, period_end TEXT NOT NULL, generated_at TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','in-review','approved')), approved_cost_minor INTEGER NOT NULL CHECK(approved_cost_minor >= 0), commitment_minor INTEGER NOT NULL CHECK(commitment_minor >= 0), budget_minor INTEGER NOT NULL CHECK(budget_minor >= 0), discrepancy_minor INTEGER NOT NULL,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0), CHECK(period_start <= period_end), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE reconciliation_lines (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, reconciliation_id TEXT NOT NULL, source_cost_id TEXT, invoice_line_id TEXT, receipt_line_id TEXT, purchase_order_line_id TEXT, work_order_milestone_id TEXT,
 discrepancy TEXT NOT NULL CHECK(discrepancy IN ('matched','missing-evidence','duplicate','amount-mismatch','unrecognized')), expected_minor INTEGER NOT NULL CHECK(expected_minor >= 0), actual_minor INTEGER NOT NULL CHECK(actual_minor >= 0), source_version INTEGER NOT NULL CHECK(source_version > 0), description TEXT NOT NULL,
 FOREIGN KEY(reconciliation_id,project_id) REFERENCES reconciliations(id,project_id), FOREIGN KEY(source_cost_id,project_id) REFERENCES cost_sources(id,project_id), FOREIGN KEY(invoice_line_id,project_id) REFERENCES invoice_lines(id,project_id), FOREIGN KEY(receipt_line_id,project_id) REFERENCES goods_receipt_lines(id,project_id), FOREIGN KEY(purchase_order_line_id,project_id) REFERENCES purchase_order_lines(id,project_id), FOREIGN KEY(work_order_milestone_id,project_id) REFERENCES work_order_milestones(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE reconciliation_reviews (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, reconciliation_id TEXT NOT NULL, line_id TEXT, role_code TEXT NOT NULL CHECK(role_code IN ('project-manager','finance-officer')), resolution TEXT NOT NULL, reason TEXT NOT NULL CHECK(length(trim(reason)) > 0), source_cost_id TEXT,
 reviewer_id TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL,
 FOREIGN KEY(reconciliation_id,project_id) REFERENCES reconciliations(id,project_id), FOREIGN KEY(line_id,project_id) REFERENCES reconciliation_lines(id,project_id), FOREIGN KEY(source_cost_id,project_id) REFERENCES cost_sources(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE UNIQUE INDEX reconciliation_role_signoff ON reconciliation_reviews(reconciliation_id,role_code) WHERE line_id IS NULL;
CREATE UNIQUE INDEX reconciliation_independent_signoff ON reconciliation_reviews(reconciliation_id,reviewer_id) WHERE line_id IS NULL;
CREATE TABLE report_snapshots (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), report TEXT NOT NULL CHECK(report IN ('progress','procurement','budget','inventory','sales','collections')), view TEXT NOT NULL, from_date TEXT, to_date TEXT, generated_at TEXT NOT NULL, document_id TEXT NOT NULL,
 filters_json TEXT NOT NULL CHECK(json_valid(filters_json) AND json_type(filters_json)='object'), totals_json TEXT NOT NULL CHECK(json_valid(totals_json) AND json_type(totals_json)='object'), source_versions_json TEXT NOT NULL CHECK(json_valid(source_versions_json) AND json_type(source_versions_json)='array'),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL, FOREIGN KEY(document_id,project_id) REFERENCES documents(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE backup_runs (
 id TEXT PRIMARY KEY, started_at TEXT NOT NULL, completed_at TEXT, status TEXT NOT NULL CHECK(status IN ('running','completed','failed')), diagnostic_id TEXT, manifest_key TEXT UNIQUE, manifest_sha256 TEXT, database_sha256 TEXT,
 database_bytes INTEGER CHECK(database_bytes >= 0), attachment_count INTEGER CHECK(attachment_count >= 0), attachment_bytes INTEGER CHECK(attachment_bytes >= 0), recovery_point_at TEXT, retained_until TEXT, integrity_verified INTEGER NOT NULL DEFAULT 0 CHECK(integrity_verified IN (0,1)),
 CHECK(status <> 'completed' OR (completed_at IS NOT NULL AND manifest_key IS NOT NULL AND manifest_sha256 IS NOT NULL AND database_sha256 IS NOT NULL AND integrity_verified=1))
) STRICT;
CREATE INDEX backup_status_time ON backup_runs(status,started_at);
CREATE TABLE backup_objects (
 backup_run_id TEXT NOT NULL REFERENCES backup_runs(id), document_id TEXT NOT NULL REFERENCES documents(id), storage_key TEXT NOT NULL, sha256 TEXT NOT NULL CHECK(length(sha256)=64), size_bytes INTEGER NOT NULL CHECK(size_bytes > 0), PRIMARY KEY(backup_run_id,document_id)
) STRICT;
CREATE TABLE review_decisions (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), site_log_id TEXT, purchase_order_id TEXT, work_order_id TEXT, expense_id TEXT, reservation_id TEXT, inspection_id TEXT,
 decision TEXT NOT NULL CHECK(decision IN ('submit','approve','return','reject','cancel','complete')), subject_version INTEGER NOT NULL CHECK(subject_version > 0), submitted_by TEXT NOT NULL REFERENCES users(id), reviewer_id TEXT NOT NULL REFERENCES users(id), reason TEXT, occurred_at TEXT NOT NULL,
 FOREIGN KEY(site_log_id,project_id) REFERENCES site_logs(id,project_id), FOREIGN KEY(purchase_order_id,project_id) REFERENCES purchase_orders(id,project_id), FOREIGN KEY(work_order_id,project_id) REFERENCES work_orders(id,project_id), FOREIGN KEY(expense_id,project_id) REFERENCES expenses(id,project_id), FOREIGN KEY(reservation_id,project_id) REFERENCES reservations(id,project_id), FOREIGN KEY(inspection_id,project_id) REFERENCES inspections(id,project_id),
 CHECK((site_log_id IS NOT NULL)+(purchase_order_id IS NOT NULL)+(work_order_id IS NOT NULL)+(expense_id IS NOT NULL)+(reservation_id IS NOT NULL)+(inspection_id IS NOT NULL)=1),
 CHECK(decision NOT IN ('approve','return','reject') OR reviewer_id <> submitted_by), CHECK(decision NOT IN ('return','reject','cancel') OR (reason IS NOT NULL AND length(trim(reason)) > 0))
) STRICT;
