package policy

import "strings"

// RouteCapability covers the foundation's domain route slots only. Authentication
// has its own gate. Unknown methods, paths and actions are never inferred from a
// module prefix. Individual=true requires non-disclosing resource handling.
// This gate is necessary, not sufficient: handlers must apply record scope,
// parent document policy, least-data DTOs and transaction-time authorization.
func RouteCapability(method, path string) (capability string, individual bool, known bool) {
	if !strings.HasPrefix(path, "/api/v1/") || strings.ContainsAny(path, "?%\\") {
		return "", false, false
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return "", false, false
		}
	}
	if len(parts) == 1 {
		if method == "GET" {
			switch parts[0] {
			case "dashboard":
				return "dashboard.read", false, true
			case "roles":
				return "roles.read", false, true
			case "settings":
				return "settings.read", false, true
			case "stock-movements":
				return "inventory.read", false, true
			}
		}
		if parts[0] == "settings" && method == "PUT" {
			return "settings.manage", false, true
		}
	}
	if len(parts) == 2 {
		if parts[0] == "auth" && parts[1] == "change-password" && method == "POST" {
			return "auth.password.change", false, true
		}
		if parts[0] == "operator" && parts[1] == "status" && method == "GET" {
			return "operator.read", false, true
		}
		if parts[0] == "inventory" {
			if parts[1] == "balances" && method == "GET" {
				return "inventory.read", false, true
			}
			if parts[1] == "minimums" && method == "PUT" {
				return "inventory.minimums", false, true
			}
			return "", false, false
		}
		if parts[0] == "stock" {
			if method == "GET" && parts[1] == "adjustments" {
				return "stock.adjustments.read", false, true
			}
			if method == "POST" {
				cap := map[string]string{"issues": "inventory.issue", "returns": "inventory.return", "transfers": "inventory.transfer", "adjustments": "stock.adjustments.propose"}[parts[1]]
				return cap, false, cap != ""
			}
			return "", false, false
		}
		if parts[0] == "reports" && method == "GET" {
			cap := map[string]string{"progress": "reports.route.progress", "budget": "reports.route.budget", "inventory": "reports.route.inventory", "procurement": "reports.route.procurement", "sales": "reports.route.sales", "collections": "reports.route.collections"}[parts[1]]
			return cap, false, cap != ""
		}
	}
	if len(parts) == 3 && parts[0] == "projects" {
		if method == "GET" {
			cap := map[string]string{"sites": "projects.select", "summary": "projects.read", "plan": "plan.read", "milestones": "plan.read", "tasks": "plan.read", "overrides": "projects.read"}[parts[2]]
			return cap, true, cap != ""
		}
		if method == "POST" && (parts[2] == "sites" || parts[2] == "milestones" || parts[2] == "tasks") {
			if parts[2] == "sites" {
				return "sites.manage", true, true
			}
			return "plan.manage", true, true
		}
		if method == "PUT" && parts[2] == "overrides" {
			return "projects.manage", true, true
		}
		return "", false, false
	}
	if len(parts) == 3 && parts[0] == "bookings" && method == "POST" {
		if parts[2] == "sale-confirmation" {
			return "bookings.confirmSale", true, true
		}
		if parts[2] == "handover" {
			return "handover.record", true, true
		}
	}
	if len(parts) == 3 && parts[0] == "users" && parts[2] == "password-reset" && method == "POST" {
		return "users.manage", false, true
	}
	if len(parts) > 2 {
		return "", false, false
	}
	// Separate read names prevent GET from requiring or granting broad management.
	row, ok := routeResources[parts[0]]
	if !ok {
		return "", false, false
	}
	individual = len(parts) == 2
	switch method {
	case "GET":
		capability = row[0]
	case "POST":
		if !individual {
			capability = row[1]
		}
	case "PUT":
		if individual {
			capability = row[2]
		}
	}
	return capability, individual, capability != ""
}

var routeResources = map[string][3]string{
	"projects":               {"projects.read", "projects.create", "projects.manage"},
	"site-logs":              {"siteLogs.read", "siteLogs.manage", "siteLogs.manage"},
	"inspections":            {"inspections.read", "inspections.manage", "inspections.manage"},
	"issues":                 {"issues.read", "issues.manage", "issues.manage"},
	"materials":              {"materials.read", "materials.manage", "materials.manage"},
	"suppliers":              {"suppliers.read", "suppliers.manage", "suppliers.manage"},
	"purchase-orders":        {"purchaseOrders.read", "purchaseOrders.manage", "purchaseOrders.manage"},
	"goods-receipts":         {"goodsReceipts.read", "goodsReceipts.manage", ""},
	"contractors":            {"contractors.read", "contractors.manage", "contractors.manage"},
	"work-orders":            {"workOrders.read", "workOrders.manage", "workOrders.manage"},
	"budgets":                {"budgets.read", "budgets.manage", "budgets.manage"},
	"expenses":               {"expenses.read", "expenses.submit", "expenses.submit"},
	"invoices":               {"invoices.read", "invoices.manage", "invoices.manage"},
	"payments":               {"payments.read", "payments.manage", "payments.manage"},
	"properties":             {"properties.read", "properties.manage", "properties.manage"},
	"leads":                  {"leads.read", "leads.manage", "leads.manage"},
	"customers":              {"customers.read", "customers.manage", "customers.manage"},
	"site-visits":            {"visits.read", "visits.manage", "visits.manage"},
	"reservations":           {"reservations.read", "reservations.manage", ""},
	"bookings":               {"bookings.read", "bookings.manage", "bookings.manage"},
	"documents":              {"documents.parent", "documents.parent", ""},
	"users":                  {"users.read", "users.manage", "users.manage"},
	"audit-events":           {"audit.scoped", "", ""},
	"policy-acknowledgments": {"privacy.read", "privacy.acknowledge", ""},
}
