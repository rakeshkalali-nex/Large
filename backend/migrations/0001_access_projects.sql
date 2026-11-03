-- Baseline: retained references use SQLite's default NO ACTION / RESTRICT semantics.
-- UTC timestamps are fixed millisecond RFC3339; services supply UUID identifiers.
CREATE TABLE roles (
 code TEXT PRIMARY KEY CHECK(code IN ('administrator','project-manager','site-engineer','procurement-officer','finance-officer','sales-executive','management-viewer')),
 name TEXT NOT NULL
) STRICT;
INSERT INTO roles VALUES ('administrator','Administrator'),('project-manager','Project Manager'),('site-engineer','Site Engineer'),('procurement-officer','Procurement Officer'),('finance-officer','Finance Officer'),('sales-executive','Sales Executive'),('management-viewer','Management Viewer');
CREATE TABLE users (
 id TEXT PRIMARY KEY, display_name TEXT NOT NULL, email TEXT NOT NULL,
 normalized_email TEXT NOT NULL UNIQUE, role_code TEXT NOT NULL REFERENCES roles(code),
 status TEXT NOT NULL DEFAULT 'setup-required' CHECK(status IN ('setup-required','active','deactivated')),
 password_hash TEXT, access_revision INTEGER NOT NULL DEFAULT 1 CHECK(access_revision > 0),
 setup_required INTEGER NOT NULL DEFAULT 1 CHECK(setup_required IN (0,1)), deactivated_at TEXT,
 created_by TEXT REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE sessions (
 id TEXT PRIMARY KEY, token_digest TEXT NOT NULL UNIQUE, user_id TEXT NOT NULL REFERENCES users(id),
 access_revision INTEGER NOT NULL CHECK(access_revision > 0), credential_revision INTEGER NOT NULL DEFAULT 1 CHECK(credential_revision > 0),
 created_at TEXT NOT NULL, last_interaction_at TEXT NOT NULL, idle_expires_at TEXT NOT NULL,
 absolute_expires_at TEXT NOT NULL, revoked_at TEXT,
 CHECK(created_at < absolute_expires_at), CHECK(idle_expires_at <= absolute_expires_at)
) STRICT;
CREATE INDEX sessions_user ON sessions(user_id,revoked_at);
CREATE INDEX sessions_expiry ON sessions(absolute_expires_at,idle_expires_at);
CREATE TABLE password_tokens (
 id TEXT PRIMARY KEY, token_digest TEXT NOT NULL UNIQUE, user_id TEXT NOT NULL REFERENCES users(id),
 kind TEXT NOT NULL CHECK(kind IN ('setup','recovery')), issuer_id TEXT NOT NULL REFERENCES users(id),
 created_at TEXT NOT NULL, expires_at TEXT NOT NULL, consumed_at TEXT, revoked_at TEXT,
 identity_verified INTEGER NOT NULL DEFAULT 0 CHECK(identity_verified IN (0,1)), reason TEXT NOT NULL,
 CHECK(expires_at > created_at), CHECK(kind <> 'recovery' OR identity_verified = 1)
) STRICT;
CREATE INDEX password_tokens_deadlines ON password_tokens(expires_at,user_id,kind);
CREATE TABLE rate_limit_buckets (
 subject_digest TEXT NOT NULL, operation TEXT NOT NULL, window_start TEXT NOT NULL,
 expires_at TEXT NOT NULL, attempts INTEGER NOT NULL CHECK(attempts >= 0),
 PRIMARY KEY(subject_digest,operation,window_start)
) STRICT;
CREATE INDEX rate_limit_expiry ON rate_limit_buckets(expires_at);
CREATE TABLE projects (
 id TEXT PRIMARY KEY, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL, address TEXT NOT NULL, project_type TEXT NOT NULL,
 start_date TEXT NOT NULL, end_date TEXT NOT NULL, manager_id TEXT NOT NULL REFERENCES users(id),
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','active','on-hold','completed','archived')),
 archived INTEGER NOT NULL DEFAULT 0 CHECK(archived IN (0,1)), closed_at TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 CHECK(start_date <= end_date)
) STRICT;
CREATE INDEX projects_status ON projects(status,updated_at,id);
CREATE TABLE user_projects (
 user_id TEXT NOT NULL REFERENCES users(id), project_id TEXT NOT NULL REFERENCES projects(id),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 PRIMARY KEY(user_id,project_id)
) STRICT;
CREATE INDEX user_projects_project ON user_projects(project_id,user_id);
CREATE TABLE sites (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), code TEXT NOT NULL, name TEXT NOT NULL, address TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')), reservation_hours INTEGER CHECK(reservation_hours > 0),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 UNIQUE(project_id,code), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE project_overrides (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL UNIQUE REFERENCES projects(id), reservation_hours INTEGER CHECK(reservation_hours > 0),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE plan_revisions (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), revision INTEGER NOT NULL CHECK(revision > 0),
 reason TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('draft','active','superseded')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 UNIQUE(project_id,revision), UNIQUE(id,project_id)
) STRICT;
CREATE UNIQUE INDEX one_active_plan ON plan_revisions(project_id) WHERE status='active';
CREATE TABLE milestones (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), name TEXT NOT NULL, responsible_id TEXT NOT NULL REFERENCES users(id),
 start_date TEXT NOT NULL, end_date TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','in-progress','completed','cancelled')),
 completion_bps INTEGER NOT NULL DEFAULT 0 CHECK(completion_bps BETWEEN 0 AND 10000), weight_bps INTEGER NOT NULL CHECK(weight_bps BETWEEN 1 AND 10000),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 CHECK(start_date <= end_date), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE plan_revision_milestones (
 revision_id TEXT NOT NULL, project_id TEXT NOT NULL, milestone_id TEXT NOT NULL,
 name TEXT NOT NULL, responsible_id TEXT NOT NULL REFERENCES users(id), start_date TEXT NOT NULL, end_date TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('planned','in-progress','completed','cancelled')),
 completion_bps INTEGER NOT NULL CHECK(completion_bps BETWEEN 0 AND 10000), weight_bps INTEGER NOT NULL CHECK(weight_bps BETWEEN 1 AND 10000),
 PRIMARY KEY(revision_id,milestone_id), FOREIGN KEY(revision_id,project_id) REFERENCES plan_revisions(id,project_id),
 FOREIGN KEY(milestone_id,project_id) REFERENCES milestones(id,project_id), CHECK(start_date <= end_date)
) STRICT;
CREATE TABLE tasks (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), milestone_id TEXT,
 name TEXT NOT NULL, responsible_id TEXT NOT NULL REFERENCES users(id), start_date TEXT NOT NULL, end_date TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','in-progress','completed','cancelled')),
 completion_bps INTEGER NOT NULL DEFAULT 0 CHECK(completion_bps BETWEEN 0 AND 10000), notes TEXT NOT NULL DEFAULT '',
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 CHECK(start_date <= end_date), FOREIGN KEY(milestone_id,project_id) REFERENCES milestones(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX tasks_due ON tasks(project_id,status,end_date,id);
CREATE TABLE site_logs (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), site_id TEXT NOT NULL, date TEXT NOT NULL,
 work_performed TEXT NOT NULL, weather TEXT NOT NULL, blockers TEXT NOT NULL, notes TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted','approved','returned')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(site_id,project_id) REFERENCES sites(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX site_logs_date ON site_logs(project_id,date,id);
CREATE TABLE site_log_workers (
 site_log_id TEXT NOT NULL REFERENCES site_logs(id), category TEXT NOT NULL, count INTEGER NOT NULL CHECK(count >= 0), PRIMARY KEY(site_log_id,category)
) STRICT;
CREATE TABLE inspection_checklists (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision > 0), status TEXT NOT NULL CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
) STRICT;
CREATE TABLE inspection_checklist_items (
 id TEXT PRIMARY KEY, checklist_id TEXT NOT NULL REFERENCES inspection_checklists(id), label TEXT NOT NULL,
 required INTEGER NOT NULL CHECK(required IN (0,1)), sort_order INTEGER NOT NULL CHECK(sort_order >= 0), UNIQUE(checklist_id,sort_order)
) STRICT;
CREATE TABLE project_checklists (
 project_id TEXT NOT NULL REFERENCES projects(id), checklist_id TEXT NOT NULL REFERENCES inspection_checklists(id), PRIMARY KEY(project_id,checklist_id)
) STRICT;
CREATE TABLE inspections (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), site_id TEXT NOT NULL, checklist_id TEXT NOT NULL REFERENCES inspection_checklists(id),
 milestone_id TEXT, property_id TEXT, status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','submitted','completed')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(site_id,project_id) REFERENCES sites(id,project_id), FOREIGN KEY(milestone_id,project_id) REFERENCES milestones(id,project_id),
 FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE inspection_items (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), inspection_id TEXT NOT NULL, checklist_item_id TEXT REFERENCES inspection_checklist_items(id),
 label TEXT NOT NULL, result TEXT NOT NULL CHECK(result IN ('pass','fail','not-applicable')), finding TEXT NOT NULL, issue_id TEXT,
 FOREIGN KEY(inspection_id,project_id) REFERENCES inspections(id,project_id), FOREIGN KEY(issue_id,project_id) REFERENCES issues(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE issues (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), inspection_item_id TEXT, milestone_id TEXT, property_id TEXT,
 title TEXT NOT NULL, description TEXT NOT NULL, owner_id TEXT NOT NULL REFERENCES users(id),
 priority TEXT NOT NULL CHECK(priority IN ('low','normal','high','critical')), due_date TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'open' CHECK(status IN ('open','in-progress','resolved','reopened')), resolution_notes TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
 updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(inspection_item_id,project_id) REFERENCES inspection_items(id,project_id), FOREIGN KEY(milestone_id,project_id) REFERENCES milestones(id,project_id),
 FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id), CHECK(status <> 'resolved' OR length(trim(resolution_notes)) > 0), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX issues_due ON issues(project_id,status,due_date,id);
CREATE TABLE issue_events (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, issue_id TEXT NOT NULL, actor_id TEXT NOT NULL REFERENCES users(id),
 from_status TEXT CHECK(from_status IN ('open','in-progress','resolved','reopened')), to_status TEXT NOT NULL CHECK(to_status IN ('open','in-progress','resolved','reopened')),
 reason TEXT NOT NULL CHECK(length(trim(reason)) > 0), occurred_at TEXT NOT NULL,
 FOREIGN KEY(issue_id,project_id) REFERENCES issues(id,project_id)
) STRICT;
