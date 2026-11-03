// Package policy is the deny-default server authorization boundary.
package policy

import "sort"

// Grants are exact names, never prefixes or wildcards. Narrow projection grants
// deliberately do not imply access to a module's full read/mutation endpoints.
var roleGrants = map[string][]string{
	"administrator": {
		"users.manage", "roles.read", "assignments.manage", "settings.manage", "operator.read", "privacy.manage", "retention.manage",
		"projects.create", "projects.read", "projects.manage", "projects.select", "sites.manage", "plan.read", "plan.manage", "tasks.progress",
		"siteLogs.read", "siteLogs.manage", "siteLogs.review", "inspections.read", "inspections.manage", "issues.read", "issues.manage", "issues.act",
		"materials.select", "materials.read", "materials.manage", "suppliers.read", "suppliers.manage", "inventory.read", "inventory.issue", "inventory.return", "inventory.transfer", "inventory.minimums",
		"purchaseOrders.read", "purchaseOrders.manage", "purchaseOrders.submit", "purchaseOrders.approve", "goodsReceipts.manage", "stock.adjustments.read", "stock.adjustments.propose", "stock.adjustments.approve",
		"contractors.read", "contractors.manage", "workOrders.read", "workOrders.manage", "workOrders.approve", "workOrders.evidence",
		"budgets.read", "budgets.manage", "expenses.read", "expenses.submit", "expenses.review", "costs.summary", "costs.evidence",
		"invoices.read", "invoices.manage", "invoices.post", "invoices.reverse", "payments.read", "payments.manage", "payments.post", "payments.reverse", "allocations.manage",
		"properties.read", "properties.manage", "properties.construction", "properties.inspection", "leads.read", "leads.manage", "customers.read", "customers.manage", "visits.manage", "reservations.manage",
		"bookings.read", "bookings.manage", "bookings.collections", "bookings.paymentCheck", "bookings.requestCancellation", "bookings.approveCancellation", "bookings.recordSale", "bookings.confirmSale",
		"cancellationTerms.propose", "cancellationTerms.approve", "refunds.propose", "refunds.approve", "refunds.record", "handover.checklist", "handover.clearance", "handover.record",
		"dashboard.read", "reports.progress", "reports.issues", "reports.inventory", "reports.orders", "reports.financial", "reports.sales", "reports.summary", "reconciliations.manage", "audit.read", "documents.parent",
	},
	"project-manager": {
		"projects.create", "projects.read", "projects.manage", "projects.select", "sites.manage", "plan.read", "plan.manage", "tasks.progress",
		"siteLogs.read", "siteLogs.review", "inspections.read", "issues.read", "issues.manage", "issues.act",
		"materials.read", "materials.select", "suppliers.read", "inventory.read", "purchaseOrders.read", "purchaseOrders.manage", "purchaseOrders.submit", "purchaseOrders.approve", "stock.adjustments.read",
		"contractors.read", "contractors.manage", "workOrders.read", "workOrders.manage", "budgets.read", "expenses.read", "expenses.submit", "costs.summary",
		"properties.construction", "handover.checklist", "dashboard.read", "reports.progress", "reports.issues", "reports.budgetSummary", "reconciliations.manage", "audit.approvals", "documents.parent",
	},
	"site-engineer": {
		"projects.select", "plan.read", "tasks.progress", "siteLogs.read", "siteLogs.manage", "inspections.read", "inspections.manage", "issues.read", "issues.act",
		"materials.select", "inventory.read", "inventory.issue", "inventory.return", "workOrders.evidence", "expenses.read", "expenses.submit", "properties.inspection",
		"dashboard.read", "reports.progress", "reports.issues", "reports.quantities", "documents.parent",
	},
	"procurement-officer": {
		"projects.select", "materials.select", "materials.read", "materials.manage", "suppliers.read", "suppliers.manage", "inventory.read", "inventory.issue", "inventory.return", "inventory.transfer", "inventory.minimums",
		"purchaseOrders.read", "purchaseOrders.manage", "purchaseOrders.submit", "goodsReceipts.manage", "stock.adjustments.propose", "stock.adjustments.read", "contractors.references", "workOrders.references", "expenses.read", "expenses.submit",
		"dashboard.read", "reports.orders", "reports.inventory", "reports.quantities", "documents.parent",
	},
	"finance-officer": {
		"projects.select", "costs.summary", "costs.evidence", "purchaseOrders.read", "purchaseOrders.approve", "stock.adjustments.evidence", "contractors.read", "workOrders.read", "workOrders.approve",
		"budgets.read", "budgets.manage", "expenses.read", "expenses.submit", "expenses.review", "invoices.read", "invoices.manage", "invoices.post", "invoices.reverse", "payments.read", "payments.manage", "payments.post", "payments.reverse", "allocations.manage",
		"bookings.read", "bookings.collections", "bookings.paymentCheck", "bookings.confirmSale", "cancellationTerms.propose", "refunds.propose", "refunds.record", "handover.clearance",
		"dashboard.read", "reports.financial", "reconciliations.manage", "audit.financial", "documents.parent",
	},
	"sales-executive": {
		"projects.select", "properties.read", "properties.manage", "leads.read", "leads.manage", "customers.read", "customers.manage", "visits.manage", "reservations.manage", "bookings.read", "bookings.manage", "bookings.collections", "bookings.requestCancellation", "bookings.recordSale", "handover.record", "expenses.read", "expenses.submit",
		"dashboard.read", "reports.sales", "documents.parent",
	},
	"management-viewer": {"dashboard.read", "reports.summary", "documents.reportEvidence"},
}

// Capabilities returns a detached, sorted list. Unknown roles have no grants.
func Capabilities(role string) []string {
	result := append([]string{}, roleGrants[role]...)
	result = append(result, routeReadGrants[role]...)
	if _, known := roleGrants[role]; known {
		result = append(result, "auth.password.change")
	}
	// Administrator receives the explicitly listed narrow projections as well.
	if role == "administrator" {
		result = append(result, "reports.budgetSummary", "reports.quantities", "audit.approvals", "audit.financial", "contractors.references", "workOrders.references", "stock.adjustments.evidence", "documents.reportEvidence")
	}
	sort.Strings(result)
	return result
}

// These grants open only projected route shells; report views, audit trails and
// acknowledgments must still apply the role-specific grants and parent scope.
var routeReadGrants = map[string][]string{
	"administrator":       {"users.read", "settings.read", "goodsReceipts.read", "visits.read", "reservations.read", "audit.scoped", "privacy.read", "privacy.acknowledge", "reports.route.progress", "reports.route.procurement", "reports.route.budget", "reports.route.inventory", "reports.route.sales", "reports.route.collections"},
	"project-manager":     {"audit.scoped", "privacy.acknowledge", "reports.route.progress", "reports.route.budget"},
	"site-engineer":       {"privacy.acknowledge", "reports.route.progress", "reports.route.inventory"},
	"procurement-officer": {"goodsReceipts.read", "privacy.acknowledge", "reports.route.procurement", "reports.route.inventory"},
	"finance-officer":     {"audit.scoped", "privacy.acknowledge", "reports.route.budget", "reports.route.collections"},
	"sales-executive":     {"visits.read", "reservations.read", "privacy.acknowledge", "reports.route.sales", "reports.route.collections"},
	"management-viewer":   {"reports.route.progress", "reports.route.procurement", "reports.route.budget", "reports.route.inventory", "reports.route.sales", "reports.route.collections"},
}

func hasCapability(role, capability string) bool {
	for _, grant := range Capabilities(role) {
		if grant == capability {
			return true
		}
	}
	return false
}

func LandingPath(role string) string {
	switch role {
	case "administrator", "management-viewer":
		return "/dashboard"
	case "project-manager":
		return "/projects"
	case "site-engineer":
		return "/site-logs"
	case "procurement-officer":
		return "/purchase-orders"
	case "finance-officer":
		return "/invoices"
	case "sales-executive":
		return "/leads"
	default:
		return "/login"
	}
}
