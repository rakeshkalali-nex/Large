# Server policy integration

The package changes no session state and never refreshes idle expiry.

## API

- `Principal{UserID, SessionDigest}` and `WithPrincipal(context.Context, Principal)`:
  middleware supplies verified user ID and persisted token digest, not browser
  roles, permissions or project claims.
- `Service{DB *sql.DB, Clock contracts.Clock}` implements `contracts.Policy` via
  `Authorize(ctx, capability, projectID) error`.
- `AuthorizeTx(ctx, tx contracts.Transaction, capability, projectID) error` reads
  only the supplied transaction. It rechecks current user status/setup flag,
  role, access revision, credential revision (`sessions.credential_revision ==
  users.version`), revocation, idle/absolute expiry and project assignments.
- `Capabilities(role) []string` returns a detached sorted list; `LandingPath(role)
  string` returns `/login` for unknown roles. Administrator uses explicit grants,
  not a wildcard or an implicit unknown-capability fallback.
- `RouteCapability(method, path string) (capability string, individual bool,
  known bool)` accepts a full `/api/v1/...` path. Only listed domain slots and
  methods are recognized. Password change maps to `auth.password.change` for all
  seven active roles; other auth routes are owned by session handlers. Unknown
  actions/methods fail closed. `individual` requires non-disclosing 404 handling.
- `Scope(ctx, resourceName)` and `ScopeTx(ctx, tx, resourceName)` return
  `QueryScope{Table, Columns, Predicate, Args}`. The table already uses alias `r`.
  Use Columns and Predicate together, bind Args, and apply WHERE before counts,
  sorting, pagination or aggregation. Do not substitute `SELECT *`.
- `RequireResourceTx(ctx, tx, resourceName, id)` uses the same SQL scope as lists;
  forbidden, absent and inaccessible resources receive the same 404.
- `RequireAssignedTaskTx(ctx, tx, taskID)` enforces current project membership and
  persisted engineer task assignment, including changes within the transaction.
- `NotFound()` supplies the stable non-disclosing resource error.

## Boundaries for subsequent feature handlers

Route authorization alone is not record authorization. An empty project ID opens
the collection shell and must never cause an unfiltered query. Scoped query plans
currently cover projects, tasks, site logs, inspections, issues, leads, customers,
bookings, properties and expenses. Unknown plan names fail closed. Extend the
closed registry with tested named projections for each later feature, rather than
creating a caller-configurable SQL identifier or generic full-row escape hatch.

Sales queries restrict assigned records and linked customers; property queries
permit available units and linked assigned bookings/reservations. Engineers' log
and inspection queries restrict ownership. Non-finance expense reads restrict
creator. Issue list visibility is project scoped; assigned issue mutation checks
remain the issue handler's responsibility, not inferred from list visibility.

Report route grants are only gates for role-specific report DTOs: `reports.route.*`
never authorizes full financial fields. Management Viewer remains report-only.
Audit route grants require assigned approval/financial trail predicates in the
audit handler. Document route gates require the intersection of all parent
policies before opening files; no blanket document resource query plan exists.
Privacy acknowledgment handlers must validate authorized parent record ownership.
Workflow handlers must recheck independent reviewer separation, approval routing,
related-record scopes, field writes and lifecycle rules inside the same write
transaction. Administrator does not bypass those checks.

## Validation status

Tests added against disposable databases and real migrations for seven roles,
unknown grants/claims, expiry, current revisions/status, assignment and role
changes inside transactions, scoped counts and projections, no-project behavior,
record ownership and the closed route registry. No commands, formatting commands
or tests were run by this subtask; the parent owns execution and formatting.
