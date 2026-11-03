CREATE TABLE materials (
 id TEXT PRIMARY KEY, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL, unit TEXT NOT NULL, minimum_quantity INTEGER NOT NULL CHECK(minimum_quantity >= 0),
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0), UNIQUE(id,unit)
) STRICT;
CREATE TABLE material_site_minimums (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), site_id TEXT, material_id TEXT NOT NULL REFERENCES materials(id), minimum_quantity INTEGER NOT NULL CHECK(minimum_quantity >= 0),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(site_id,project_id) REFERENCES sites(id,project_id), UNIQUE(site_id,material_id)
) STRICT;
CREATE UNIQUE INDEX project_material_minimum ON material_site_minimums(project_id,material_id) WHERE site_id IS NULL;
CREATE TABLE suppliers (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, address TEXT NOT NULL, reference TEXT, status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE supplier_contacts (id TEXT PRIMARY KEY, supplier_id TEXT NOT NULL REFERENCES suppliers(id), name TEXT NOT NULL, email TEXT, phone TEXT, normalized_email TEXT, normalized_phone TEXT) STRICT;
CREATE TABLE cost_categories (
 id TEXT PRIMARY KEY, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE purchase_orders (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), site_id TEXT NOT NULL, supplier_id TEXT NOT NULL REFERENCES suppliers(id), number TEXT NOT NULL UNIQUE,
 delivery_date TEXT NOT NULL, notes TEXT NOT NULL, total_minor INTEGER NOT NULL DEFAULT 0 CHECK(total_minor >= 0), revision INTEGER NOT NULL DEFAULT 1 CHECK(revision > 0), approved_revision_id TEXT,
 submitter_role TEXT REFERENCES roles(code), submitted_by TEXT REFERENCES users(id), submitted_at TEXT,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted','returned','approved','part-received','received','cancelled')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(site_id,project_id) REFERENCES sites(id,project_id), FOREIGN KEY(approved_revision_id,id) REFERENCES purchase_order_revisions(id,purchase_order_id), UNIQUE(id,project_id), UNIQUE(id,project_id,site_id)
) STRICT;
CREATE INDEX purchase_orders_supplier ON purchase_orders(supplier_id,project_id,status,delivery_date);
CREATE TABLE purchase_order_lines (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, purchase_order_id TEXT NOT NULL, material_id TEXT NOT NULL, unit TEXT NOT NULL,
 quantity INTEGER NOT NULL CHECK(quantity > 0), unit_price INTEGER NOT NULL CHECK(unit_price >= 0), amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0), category_id TEXT NOT NULL REFERENCES cost_categories(id),
 FOREIGN KEY(purchase_order_id,project_id) REFERENCES purchase_orders(id,project_id), FOREIGN KEY(material_id,unit) REFERENCES materials(id,unit), UNIQUE(id,project_id), UNIQUE(id,purchase_order_id,project_id,unit)
) STRICT;
CREATE TABLE purchase_order_revisions (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, purchase_order_id TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision > 0), delivery_date TEXT NOT NULL, reason TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('submitted','approved','rejected','superseded')), proposed_by TEXT NOT NULL REFERENCES users(id), decided_by TEXT REFERENCES users(id), decided_at TEXT, decision_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(purchase_order_id,project_id) REFERENCES purchase_orders(id,project_id), UNIQUE(purchase_order_id,revision), UNIQUE(id,purchase_order_id), CHECK(decided_by IS NULL OR decided_by <> proposed_by)
) STRICT;
CREATE TABLE purchase_order_revision_lines (
 revision_id TEXT NOT NULL REFERENCES purchase_order_revisions(id), line_id TEXT NOT NULL REFERENCES purchase_order_lines(id), material_id TEXT NOT NULL REFERENCES materials(id), unit TEXT NOT NULL,
 quantity INTEGER NOT NULL CHECK(quantity > 0), unit_price INTEGER NOT NULL CHECK(unit_price >= 0), amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0), category_id TEXT NOT NULL REFERENCES cost_categories(id), PRIMARY KEY(revision_id,line_id), FOREIGN KEY(material_id,unit) REFERENCES materials(id,unit)
) STRICT;
CREATE TABLE goods_receipts (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, purchase_order_id TEXT NOT NULL, site_id TEXT NOT NULL, received_date TEXT NOT NULL, reference TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','posted','reversed')), stock_transaction_id TEXT REFERENCES stock_transactions(id), reversal_id TEXT REFERENCES stock_transactions(id),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(purchase_order_id,project_id,site_id) REFERENCES purchase_orders(id,project_id,site_id), UNIQUE(id,project_id), UNIQUE(id,purchase_order_id,project_id)
) STRICT;
CREATE TABLE goods_receipt_lines (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, goods_receipt_id TEXT NOT NULL, purchase_order_id TEXT NOT NULL, purchase_order_line_id TEXT NOT NULL,
 quantity INTEGER NOT NULL CHECK(quantity > 0), unit TEXT NOT NULL,
 FOREIGN KEY(goods_receipt_id,purchase_order_id,project_id) REFERENCES goods_receipts(id,purchase_order_id,project_id), FOREIGN KEY(purchase_order_line_id,purchase_order_id,project_id,unit) REFERENCES purchase_order_lines(id,purchase_order_id,project_id,unit),
 UNIQUE(goods_receipt_id,purchase_order_line_id), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX receipt_lines_order ON goods_receipt_lines(purchase_order_line_id);
CREATE TABLE stock_transactions (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), type TEXT NOT NULL CHECK(type IN ('receipt','issue','return','transfer','adjustment','reversal')),
 original_transaction_id TEXT, goods_receipt_id TEXT, adjustment_id TEXT, site_log_id TEXT, task_id TEXT, work_order_id TEXT,
 reason TEXT NOT NULL, actor_id TEXT NOT NULL REFERENCES users(id), occurred_at TEXT NOT NULL,
 FOREIGN KEY(original_transaction_id,project_id) REFERENCES stock_transactions(id,project_id), FOREIGN KEY(goods_receipt_id,project_id) REFERENCES goods_receipts(id,project_id),
 FOREIGN KEY(adjustment_id,project_id) REFERENCES stock_adjustments(id,project_id), FOREIGN KEY(site_log_id,project_id) REFERENCES site_logs(id,project_id), FOREIGN KEY(task_id,project_id) REFERENCES tasks(id,project_id), FOREIGN KEY(work_order_id,project_id) REFERENCES work_orders(id,project_id), UNIQUE(id,project_id),
 CHECK(type <> 'issue' OR (site_log_id IS NOT NULL)+(task_id IS NOT NULL)+(work_order_id IS NOT NULL)=1), CHECK(type NOT IN ('return','reversal') OR original_transaction_id IS NOT NULL)
) STRICT;
CREATE UNIQUE INDEX receipt_post_once ON stock_transactions(goods_receipt_id) WHERE type='receipt';
CREATE UNIQUE INDEX adjustment_post_once ON stock_transactions(adjustment_id) WHERE type='adjustment';
CREATE TABLE stock_entries (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, transaction_id TEXT NOT NULL, site_id TEXT NOT NULL, material_id TEXT NOT NULL, unit TEXT NOT NULL,
 quantity INTEGER NOT NULL CHECK(quantity <> 0), source_line_id TEXT NOT NULL, effect TEXT NOT NULL CHECK(effect IN ('in','out','compensation')), receipt_line_id TEXT,
 FOREIGN KEY(transaction_id,project_id) REFERENCES stock_transactions(id,project_id), FOREIGN KEY(site_id,project_id) REFERENCES sites(id,project_id), FOREIGN KEY(material_id,unit) REFERENCES materials(id,unit), FOREIGN KEY(receipt_line_id,project_id) REFERENCES goods_receipt_lines(id,project_id), UNIQUE(transaction_id,source_line_id,effect)
) STRICT;
CREATE INDEX stock_balance ON stock_entries(site_id,material_id,id);
CREATE TABLE stock_adjustments (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), site_id TEXT NOT NULL, material_id TEXT NOT NULL, unit TEXT NOT NULL, quantity_delta INTEGER NOT NULL CHECK(quantity_delta <> 0), reason TEXT NOT NULL CHECK(length(trim(reason)) > 0),
 status TEXT NOT NULL DEFAULT 'submitted' CHECK(status IN ('submitted','approved','rejected')), decided_by TEXT REFERENCES users(id), decided_at TEXT, decision_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(site_id,project_id) REFERENCES sites(id,project_id), FOREIGN KEY(material_id,unit) REFERENCES materials(id,unit), CHECK(decided_by IS NULL OR decided_by <> created_by), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE contractors (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE contractor_contacts (id TEXT PRIMARY KEY, contractor_id TEXT NOT NULL REFERENCES contractors(id), name TEXT NOT NULL, email TEXT, phone TEXT, normalized_email TEXT, normalized_phone TEXT) STRICT;
CREATE TABLE contractor_trades (contractor_id TEXT NOT NULL REFERENCES contractors(id), trade_category TEXT NOT NULL, PRIMARY KEY(contractor_id,trade_category)) STRICT;
CREATE TABLE contractor_projects (contractor_id TEXT NOT NULL REFERENCES contractors(id), project_id TEXT NOT NULL REFERENCES projects(id), PRIMARY KEY(contractor_id,project_id)) STRICT;
CREATE TABLE work_orders (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), contractor_id TEXT NOT NULL, number TEXT NOT NULL UNIQUE, scope TEXT NOT NULL,
 start_date TEXT NOT NULL, end_date TEXT NOT NULL, manager_id TEXT NOT NULL REFERENCES users(id), total_minor INTEGER NOT NULL CHECK(total_minor >= 0), completion_bps INTEGER NOT NULL DEFAULT 0 CHECK(completion_bps BETWEEN 0 AND 10000),
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision > 0), approved_revision_id TEXT, status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted','returned','approved','in-progress','completed','cancelled')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(contractor_id,project_id) REFERENCES contractor_projects(contractor_id,project_id), FOREIGN KEY(approved_revision_id,id) REFERENCES work_order_revisions(id,work_order_id), CHECK(start_date <= end_date), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE work_order_milestones (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, work_order_id TEXT NOT NULL, name TEXT NOT NULL, due_date TEXT NOT NULL, amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0), category_id TEXT NOT NULL REFERENCES cost_categories(id), completion_bps INTEGER NOT NULL DEFAULT 0 CHECK(completion_bps BETWEEN 0 AND 10000),
 FOREIGN KEY(work_order_id,project_id) REFERENCES work_orders(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE work_order_revisions (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, work_order_id TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision > 0), end_date TEXT NOT NULL, reason TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('submitted','approved','rejected','superseded')), proposed_by TEXT NOT NULL REFERENCES users(id), decided_by TEXT REFERENCES users(id), decided_at TEXT, decision_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(work_order_id,project_id) REFERENCES work_orders(id,project_id), UNIQUE(work_order_id,revision), UNIQUE(id,work_order_id), CHECK(decided_by IS NULL OR decided_by <> proposed_by)
) STRICT;
CREATE TABLE work_order_revision_milestones (
 revision_id TEXT NOT NULL REFERENCES work_order_revisions(id), milestone_id TEXT NOT NULL REFERENCES work_order_milestones(id), name TEXT NOT NULL, due_date TEXT NOT NULL, amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0), category_id TEXT NOT NULL REFERENCES cost_categories(id), PRIMARY KEY(revision_id,milestone_id)
) STRICT;
CREATE TABLE budgets (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL UNIQUE REFERENCES projects(id), period_start TEXT NOT NULL, period_end TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','approved','revised')), current_revision_id TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 CHECK(period_start <= period_end), FOREIGN KEY(current_revision_id,id) REFERENCES budget_revisions(id,budget_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE budget_revisions (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, budget_id TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision > 0), reason TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('draft','approved','superseded')), approved_by TEXT REFERENCES users(id), approved_at TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(budget_id,project_id) REFERENCES budgets(id,project_id), UNIQUE(budget_id,revision), UNIQUE(id,budget_id), CHECK(approved_by IS NULL OR approved_by <> created_by)
) STRICT;
CREATE UNIQUE INDEX current_budget_revision ON budget_revisions(project_id) WHERE status='approved';
CREATE TABLE budget_lines (id TEXT PRIMARY KEY, revision_id TEXT NOT NULL REFERENCES budget_revisions(id), category_id TEXT NOT NULL REFERENCES cost_categories(id), amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0), UNIQUE(revision_id,category_id)) STRICT;
CREATE TABLE cost_sources (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), source_key TEXT NOT NULL, category_id TEXT NOT NULL REFERENCES cost_categories(id), amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0), purchase_order_line_id TEXT, work_order_milestone_id TEXT, description TEXT NOT NULL,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(purchase_order_line_id,project_id) REFERENCES purchase_order_lines(id,project_id), FOREIGN KEY(work_order_milestone_id,project_id) REFERENCES work_order_milestones(id,project_id), CHECK(purchase_order_line_id IS NULL OR work_order_milestone_id IS NULL), UNIQUE(project_id,source_key), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE expenses (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), category_id TEXT NOT NULL REFERENCES cost_categories(id), amount_minor INTEGER NOT NULL CHECK(amount_minor > 0), reference TEXT NOT NULL, date TEXT NOT NULL, description TEXT NOT NULL, work_order_id TEXT, source_cost_id TEXT,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted','approved','rejected')), submission INTEGER NOT NULL DEFAULT 1 CHECK(submission > 0),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(work_order_id,project_id) REFERENCES work_orders(id,project_id), FOREIGN KEY(source_cost_id,project_id) REFERENCES cost_sources(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE cost_recognitions (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, source_cost_id TEXT NOT NULL, category_id TEXT NOT NULL REFERENCES cost_categories(id), amount_minor INTEGER NOT NULL CHECK(amount_minor <> 0),
 kind TEXT NOT NULL CHECK(kind IN ('recognition','reversal')), reverses_id TEXT UNIQUE REFERENCES cost_recognitions(id), source_revision INTEGER NOT NULL CHECK(source_revision > 0),
 expense_id TEXT, invoice_line_id TEXT, purchase_order_line_id TEXT, work_order_milestone_id TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), reason TEXT NOT NULL,
 FOREIGN KEY(source_cost_id,project_id) REFERENCES cost_sources(id,project_id), FOREIGN KEY(expense_id,project_id) REFERENCES expenses(id,project_id), FOREIGN KEY(invoice_line_id,project_id) REFERENCES invoice_lines(id,project_id), FOREIGN KEY(purchase_order_line_id,project_id) REFERENCES purchase_order_lines(id,project_id), FOREIGN KEY(work_order_milestone_id,project_id) REFERENCES work_order_milestones(id,project_id),
 CHECK((kind='recognition' AND amount_minor > 0 AND reverses_id IS NULL) OR (kind='reversal' AND amount_minor < 0 AND reverses_id IS NOT NULL)), UNIQUE(source_cost_id,source_revision,kind)
) STRICT;
