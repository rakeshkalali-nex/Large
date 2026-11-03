package policy

import (
	"context"
	"errors"

	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/transactions"
)

// QueryScope is a closed SQL read plan. Use its projection and predicate together,
// before pagination/counting/aggregation. All identifiers are server constants;
// all values are bound arguments. Never replace Columns with SELECT *.
type QueryScope struct {
	Table, Columns, Predicate string
	Args                      []any
}

type resource struct{ table, capability, columns string }

var resources = map[string]resource{
	"projects":    {"projects", "projects.select", "r.id,r.code,r.name,r.status"},
	"tasks":       {"tasks", "plan.read", "r.id,r.project_id,r.name,r.responsible_id,r.status,r.completion_bps"},
	"siteLogs":    {"site_logs", "siteLogs.read", "r.id,r.project_id,r.site_id,r.date,r.work_performed,r.weather,r.blockers,r.notes,r.status"},
	"inspections": {"inspections", "inspections.read", "r.id,r.project_id,r.site_id,r.checklist_id,r.status"},
	"issues":      {"issues", "issues.read", "r.id,r.project_id,r.title,r.description,r.owner_id,r.priority,r.due_date,r.status"},
	"leads":       {"leads", "leads.read", "r.id,r.project_id,r.contact_name,r.contact_email,r.contact_phone,r.stage,r.sales_executive_id,r.next_follow_up_date"},
	"customers":   {"customers", "customers.read", "r.id,r.project_id,r.name,r.status"},
	"bookings":    {"bookings", "bookings.read", "r.id,r.project_id,r.property_id,r.customer_id,r.sales_executive_id,r.agreed_price_minor,r.deposit_required_minor,r.agreement_date,r.status,r.sale_status"},
	"properties":  {"properties", "properties.read", "r.id,r.project_id,r.site_id,r.unit_code,r.property_type,r.area,r.area_unit,r.asking_price_minor,r.construction_status,r.availability"},
	"expenses":    {"expenses", "expenses.read", "r.id,r.project_id,r.created_by,r.status"},
}

func (s Service) Scope(ctx context.Context, name string) (QueryScope, error) {
	if s.DB == nil {
		return QueryScope{}, unavailable()
	}
	return s.scope(ctx, s.DB, name)
}

func (s Service) ScopeTx(ctx context.Context, tx contracts.Transaction, name string) (QueryScope, error) {
	if tx == nil {
		return QueryScope{}, unavailable()
	}
	return s.scope(ctx, tx, name)
}

func (s Service) scope(ctx context.Context, q queryer, name string) (QueryScope, error) {
	i, err := s.current(ctx, q)
	if err != nil {
		return QueryScope{}, err
	}
	r, ok := resources[name]
	if !ok || !hasCapability(i.role, r.capability) {
		return QueryScope{}, forbidden()
	}
	project := "r.project_id"
	if name == "projects" {
		project = "r.id"
	}
	result := QueryScope{Table: r.table + " r", Columns: r.columns, Predicate: "1=1"}
	if i.role != "administrator" {
		result.Predicate = "EXISTS (SELECT 1 FROM user_projects policy_up WHERE policy_up.project_id=" + project + " AND policy_up.user_id=?)"
		result.Args = append(result.Args, i.userID)
	}
	add := func(predicate string, args ...any) {
		result.Predicate += " AND (" + predicate + ")"
		result.Args = append(result.Args, args...)
	}
	if i.role == "sales-executive" {
		switch name {
		case "leads", "bookings":
			add("r.sales_executive_id=?", i.userID)
		case "customers":
			add(`EXISTS (SELECT 1 FROM leads policy_l WHERE policy_l.id=r.source_lead_id AND policy_l.project_id=r.project_id AND policy_l.sales_executive_id=?) OR EXISTS (SELECT 1 FROM bookings policy_b WHERE policy_b.customer_id=r.id AND policy_b.project_id=r.project_id AND policy_b.sales_executive_id=?)`, i.userID, i.userID)
		case "properties":
			add(`r.availability='available' OR EXISTS (SELECT 1 FROM bookings policy_b WHERE policy_b.property_id=r.id AND policy_b.project_id=r.project_id AND policy_b.sales_executive_id=?) OR EXISTS (SELECT 1 FROM reservations policy_r JOIN customers policy_c ON policy_c.id=policy_r.customer_id AND policy_c.project_id=policy_r.project_id JOIN leads policy_l ON policy_l.id=policy_c.source_lead_id AND policy_l.project_id=policy_c.project_id WHERE policy_r.property_id=r.id AND policy_r.project_id=r.project_id AND policy_l.sales_executive_id=? AND policy_r.status IN ('active','pending-review'))`, i.userID, i.userID)
		}
	}
	if i.role == "site-engineer" && (name == "siteLogs" || name == "inspections") {
		add("r.created_by=?", i.userID)
	}
	if name == "expenses" && i.role != "administrator" && i.role != "finance-officer" {
		add("r.created_by=?", i.userID)
	}
	return result, nil
}

// RequireResourceTx resolves a record with the same scoped predicate as lists.
// Missing and inaccessible IDs both return NOT_FOUND; authentication errors stay
// 401. It is intentionally not a substitute for mutation/lifecycle authorization.
func (s Service) RequireResourceTx(ctx context.Context, tx contracts.Transaction, resourceName, id string) error {
	scope, err := s.ScopeTx(ctx, tx, resourceName)
	if err != nil {
		var safe *transactions.Error
		if errors.As(err, &safe) && safe.Status == 403 {
			return NotFound()
		}
		return err
	}
	var found int
	args := append(scope.Args, id)
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM "+scope.Table+" WHERE ("+scope.Predicate+") AND r.id=?)", args...).Scan(&found)
	if err != nil {
		return unavailable()
	}
	if found != 1 {
		return NotFound()
	}
	return nil
}

// RequireAssignedTaskTx prevents engineers progressing another user's task.
// Managers still need their current project assignment. No request owner claim
// is consumed: the responsible user and project are read from the record.
func (s Service) RequireAssignedTaskTx(ctx context.Context, tx contracts.Transaction, taskID string) error {
	if tx == nil {
		return unavailable()
	}
	i, err := s.current(ctx, tx)
	if err != nil {
		return err
	}
	if !hasCapability(i.role, "tasks.progress") {
		return NotFound()
	}
	var found int
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tasks t WHERE t.id=?
		AND (?='administrator' OR EXISTS(SELECT 1 FROM user_projects up WHERE up.project_id=t.project_id AND up.user_id=?))
		AND (?<>'site-engineer' OR t.responsible_id=?))`, taskID, i.role, i.userID, i.role, i.userID).Scan(&found)
	if err != nil {
		return unavailable()
	}
	if found != 1 {
		return NotFound()
	}
	return nil
}
