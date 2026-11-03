# Tasks: BuildEstate — Construction And Property Management

## Execution contract

Build the approved `requirements.md` and `design.md` without changing their scope, roles, API fields, calculations or visual direction. Each checkbox below is a runnable leaf. All paths are repository-relative; every command runs from the repository root in cmd.exe and exits. The application stack is Angular with strict TypeScript, Go and SQLite.

There are 30 required implementation leaves and 15 optional generated-property leaves. This exceeds the requested 30-leaf total: separate optional tasks for the approved generated-input properties, complete required coverage, and at most 10 criteria per leaf cannot all fit into 30 leaves. Preserve coverage and bounded leaf size rather than silently omitting required screens or making implementation optional. The exception is recorded under Assumptions.

### Rules applying to every feature leaf

- Deliver the whole named workflow: Go handlers/application/repository behavior, its Angular pages, typed HttpClient integration, local page styles and colocated tests. Implement the approved design's exact endpoint requests, response projections, errors, events and `effects`; validate its OpenAPI fragment before connecting UI. Do not add a second interpretation of business calculations in the browser.
- Implement the screen compositions in design's Screen composition and API wiring table: desktop arrangement, phone arrangement, live data, permitted controls and reachable actions. Use shared navy/teal tokens and IBM Plex Sans. Every feature directory owns its own `.scss` files, including any dialogs and detail panels.
- Test loading, empty, populated, success, validation, conflict, server/connectivity failure, access-denied and not-found states as applicable. Preserve safe form input, announce feedback, prevent duplicate submission, require deliberate retry, confirm destructive actions and retain the idempotency key for an uncertain retry. Test keyboard/focus behavior and widths 1440, 1024, 768 and 390 without page-wide overflow.
- Use server authorization, current account revisions, project/record scope and field projections. Test unauthorized roles and related IDs directly against HTTP handlers; hiding controls is not the security boundary. Use the shared independent-review, version, idempotency and audit transaction helpers.
- Backend tests create temporary SQLite databases, apply the real migrations and use temporary private attachment directories. Fixtures may insert valid prerequisites through shared test helpers, but must exercise the leaf's actual HTTP and business paths. Tests start and stop their own servers on available loopback ports when HTTP is needed; no manually running application, `.env`, fixed port or operational data is required.
- Frontend test filenames include their feature path segment used in `_Verify`. Frontend commands run through the frontend workspace's own Vitest/TestBed configuration. Backend commands name the changed Go package tree directly. Each leaf adds the tests its verification commands execute.
- Each feature adds an evidence fragment under its assigned `docs/evidence/features/` directory with screen IDs, routes, roles, endpoints, actions and actual check results; unexecuted checks remain explicitly unrun. Task 30 assembles the final 36-screen matrix and browser evidence. Feature unit tests are not presented as browser acceptance.
- Task 1 owns shared manifests, dependency installation, full baseline migrations, complete API composition, generated DTOs and central route mounting. It reserves stable interfaces and route slots for every leaf. Later leaves work inside their declared paths; additional shared-file changes must be explicitly assigned and ordered before execution, not made opportunistically by parallel agents.

### Parallel integration boundaries

Task 1 composes the design's ten frontend feature route exports and backend `RegisterRoutes(mux, dependencies)` modules. Within those modules it prewires separately owned subareas: projects/plan, sitework/logs/inspections, procurement/catalog/orders/receipts, inventory/balances/movements, contractors/directory/workorders, finance/budgets/invoices/payments, sales/properties/leads/bookings/sale, and administration/users/settings/audit. Pages resolve through these stable slots; new work never needs an undeclared edit to the central router. Initial unimplemented slots deny access or return a safe unavailable response and are not completed pages.

The foundation defines interfaces for policy, clock, transactions, audit, evidence validation, cost recognition, booking finance and read projections. Contracts are declared independently of concrete packages to avoid import cycles. Dependency injection connects concrete implementations through the foundation's composition root. Cross-area projections use these interfaces and the common schema; final composition must not depend on packages silently registering themselves through side effects.

All baseline table definitions and FK link tables belong to task 1. Schema tests validate the approved relational model, including exclusive unit claims and immutable history. Feature leaves consume those tables rather than creating competing migration numbers. Only task 1 edits root/frontend manifests, lockfiles and Go module files; it installs all specified test, font, icon, validation and runtime dependencies up front. Stable supported versions are selected and pinned then, not guessed here.

The six source phases remain delivery milestones: foundation/access/projects; site work; procurement/contractors; finance; property sales; integrated reporting/administration/operations. Documents, settings primitives and audit infrastructure are built earlier where evidence or posting requires them. Unrelated leaves may run together; phase labels do not impose artificial dependencies.

## Required implementation

- [x] 1. Project setup — Angular, Go, SQLite, contracts and local commands
  - Create the strict Angular standalone workspace, Go service/operator entry points, complete ordered baseline schema, typed API contracts and stable route/module composition described above. Implement configurable SQLite connections, migration checksums, foreign keys, WAL/busy-timeout policy, liveness/readiness, safe JSON middleware, static deep links and one passing frontend and backend test; cover schema constraints with real SQLite tests.
  - Write README commands exactly matching design's Build and run and command table, including cmd.exe and PowerShell usage. Implement root runners, full-test orchestration and package-specific test forwarding; initially discover only existing suites, failing actual failures rather than ignoring missing promised features. Install `@playwright/test`, configure self-starting integration support, and create `e2e/.gitkeep` excluded from unit discovery. Include `.env` in `.gitignore`, safe `.env.example` defaults, actual tool versions and pure-Go driver notes. Bind every development server to `127.0.0.1`; the runner reads `PORT`, allocates a separate Go port and generates the Angular proxy. Bootstrap/seed/backup command slots must report unavailable until their owning task implements them, never pretend success.
  - _Requirements: 26.1, 26.2, 26.3, 26.4, 26.5, 26.7, 26.8_
  - _Files: package.json, package-lock.json, .gitignore, .env.example, README.md, scripts/**, frontend/package.json, frontend/angular.json, frontend/tsconfig*.json, frontend/vitest.config.ts, frontend/src/main.ts, frontend/src/index.html, frontend/src/app/app.*, frontend/src/app/core/contracts/**, frontend/src/app/features/**/routes.ts, frontend/src/app/features/**/index.ts, frontend/src/app/features/**/route-slot.ts, frontend/src/app/setup/**, frontend/public/favicon.svg, backend/go.mod, backend/go.sum, backend/cmd/**, backend/migrations/**, backend/internal/platform/**, backend/internal/contracts/**, backend/internal/testsupport/**, backend/internal/httpapi/**, backend/internal/*/routes.go, backend/internal/*/*/route_slot.go, docs/openapi.yaml, docs/api/**, docs/architecture.md, tests/integration/support/**, tests/integration/playwright.config.ts, tests/integration/setup/**, e2e/.gitkeep_
  - _Verify: npm test --workspace frontend -- setup_
  - _Verify: npm run test:backend -- ./internal/platform/..._
  - _Verify: npm run test:backend -- ./internal/httpapi/..._
  - _Verify: npm run test:contracts_
  - _Verify: npm run build_
  - _Area: setup_
  - _Issue: #1_

- [x] 2. Shared accessible shell, architectural theme and IBM Plex Sans
  - Build the permission-aware sidebar/header, project context, user menu, breadcrumbs, route focus, safe error handling and shared controls. Implement the design's exact shared-state/event contracts; session and permission stores consume the foundation's typed access interfaces. Use locally licensed IBM Plex Sans 400–700, global token styles, accessible CDK overlays, tabular figures and responsive navigation.
  - Add TestBed checks for all shared interaction states, safe unsaved values and deliberate retry, locale/date/currency formatting, keyboard dialogs, focus, reduced motion and responsive controls. Document real-browser compatibility checks still required by task 30.
  - _Requirements: 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 5.10_
  - _Depends: 1_
  - _Files: frontend/src/app/core/shell/**, frontend/src/app/core/state/**, frontend/src/app/core/http/**, frontend/src/app/shared/**, frontend/src/styles.scss, frontend/src/styles/**, frontend/public/fonts/**, frontend/public/branding/**, docs/evidence/features/shared/**_
  - _Verify: npm test --workspace frontend -- shared_
  - _Verify: npm test --workspace frontend -- core_
  - _Area: ui_
  - _Issue: #2_

- [x] 3. Transaction integrity and independent approval infrastructure
  - Implement version checks, record metadata, lifecycle/archive helpers, persistent idempotency, eligible review-queue predicates, creator separation, atomic audit writes and rollback-safe service boundaries. Add shared exact-money/cost-source and posting-configuration interfaces with their concrete low-level helpers; domain decisions remain in their owning workflow.
  - Use fault-injected SQLite tests to prove no partial audit/review writes, stale edits, repeated effects or lost references. Audit every important mutation and authentication event through the safe writer; reject secret-bearing fields and preserve required actor/request metadata.
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 25.2_
  - _Depends: 1_
  - _Files: backend/internal/platform/transactions/**, backend/internal/platform/money/**, backend/internal/platform/posting/**, backend/internal/audit/writer/**, backend/internal/contracts/transactions.go, backend/internal/testsupport/transactions/**, docs/evidence/features/integrity/**_
  - _Verify: npm run test:backend -- ./internal/platform/transactions/..._
  - _Verify: npm run test:backend -- ./internal/platform/money/..._
  - _Verify: npm run test:backend -- ./internal/platform/posting/..._
  - _Verify: npm run test:backend -- ./internal/audit/writer/..._
  - _Area: api_
  - _Issue: #3_

- [x] 4. Secure sessions and server-determined project permissions
  - Implement login, identity restoration, logout, CSRF, opaque hashed session tokens, current-account checks and deny-by-default capability/project/field policies. Apply the design's 30-minute idle and 12-hour absolute expiry, identifier/origin rate limits, exact cookie policy and access-revision invalidation, including rechecking permission in write transactions.
  - Test all seven roles, forged claims, no protected restoration flashes through the client contract, 401/403/non-disclosing 404, current versus obsolete sessions, scoped query predicates and the policy registry for every declared route. Handler tests use real persisted users and normal credential checks.
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9_
  - _Depends: 3_
  - _Files: backend/internal/access/sessions/**, backend/internal/access/policy/**, backend/internal/access/csrf/**, backend/internal/access/ratelimit/**, backend/internal/access/routes.go, docs/api/auth.yaml, docs/evidence/features/sessions/**_
  - _Verify: npm run test:backend -- ./internal/access/sessions/..._
  - _Verify: npm run test:backend -- ./internal/access/policy/..._
  - _Verify: npm run test:backend -- ./internal/access/csrf/..._
  - _Verify: npm run test:backend -- ./internal/access/ratelimit/..._
  - _Area: api_
  - _Issue: #4_

- [ ] 5. Password setup, change and Administrator-assisted recovery
  - Implement private local first-Administrator bootstrap, Argon2id hashing, 12-character minimum, 24-hour setup and 30-minute recovery tokens, trusted-channel guidance and replacement-token handling. Complete the operator command and recovery endpoints; enforce setup-only access and atomic token consumption, session revocation and audit on password completion.
  - Test bootstrap secrecy, repeated bootstrap rejection, unknown/expired/reused/revoked tokens, concurrent token use, authenticated password change and safe Administrator issuance. Document identity verification and private link delivery without storing an existing password or logging tokens.
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8_
  - _Depends: 4_
  - _Files: backend/internal/access/passwords/**, backend/internal/access/recovery/**, backend/internal/access/bootstrap/**, backend/cmd/manage/bootstrap.go, backend/cmd/manage/bootstrap_test.go, docs/api/passwords.yaml, docs/operations/accounts.md, docs/evidence/features/recovery/**_
  - _Verify: npm run test:backend -- ./internal/access/passwords/..._
  - _Verify: npm run test:backend -- ./internal/access/recovery/..._
  - _Verify: npm run test:backend -- ./internal/access/bootstrap/..._
  - _Verify: npm run test:backend -- ./cmd/manage_
  - _Area: api_
  - _Issue: #5_

- [ ] 6. Sign in and authorized entry
  - Deliver screen 01 and root entry resolution: navy architectural desktop panel, warm-light 420px form, compact mobile brand header, visible labels, masked password with show/hide, Enter submission, password-manager/paste support and real Go authentication. Restore identity before routing and select only the authorized return path or exact role landing.
  - Build recovery guidance, setup/reset/change, expiry and service-failure states inside the account shell, with screen-owned styles and accessible loading/errors. Test safe email retention, cleared passwords, duplicate-submit prevention, malformed/external return paths, authenticated login visits and keyboard-open layout behavior through a self-starting browser smoke suite.
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9_
  - _Depends: 2, 5_
  - _Files: frontend/src/app/features/auth/**, frontend/src/app/core/session/**, frontend/src/app/core/guards/**, tests/integration/auth-entry/**, docs/evidence/features/sign-in/**_
  - _Verify: npm test --workspace frontend -- auth_
  - _Verify: npm test --workspace frontend -- session_
  - _Verify: npm test --workspace frontend -- guards_
  - _Verify: npm run test:integration -- auth-entry_
  - _Area: ui_
  - _Issue: #8_

- [ ] 7. Users and access
  - Deliver screen 34: searchable staff table/cards, account editor, explicit role/project checklist, one-time setup/recovery link dialog and reasoned deactivation confirmation. Implement users/roles/assignment endpoints, validation, lifecycle history, no-project feedback and current-session invalidation.
  - Test non-Administrator denial, preserved attribution, replacement setup on reactivation and serialized final-active-Administrator protection including competing changes. Scope permission updates to the affected users and keep the trusted-channel confirmation explicit.
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_
  - _Depends: 2, 5_
  - _Files: backend/internal/access/users/**, frontend/src/app/features/administration/users/**, docs/api/users.yaml, docs/evidence/features/users/**_
  - _Verify: npm run test:backend -- ./internal/access/users/..._
  - _Verify: npm test --workspace frontend -- administration/users_
  - _Area: api_
  - _Issue: #9_

- [ ] 8. Projects, Create project and Project overview workflow
  - Deliver screens 03–05 as one portfolio-to-project workflow: filtered portfolio rows/mobile cards, grouped identity/dates/initial-sites creation, and a project masthead with progress, sites, approved activity and permitted summary links. Implement project/site endpoints, manager assignment, unique codes, date validation, lifecycle and retained archive history.
  - Use the stable summary projection interfaces for linked domains, showing an explicit unconfigured/empty state until actual records exist. Test Project Manager auto-assignment, Administrator assignment of Site Engineers and withheld financial fields; never invent progress or totals.
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_
  - _Depends: 2, 7, 25_
  - _Files: backend/internal/projects/portfolio/**, frontend/src/app/features/projects/portfolio/**, docs/api/projects.yaml, docs/evidence/features/projects/**_
  - _Verify: npm run test:backend -- ./internal/projects/portfolio/..._
  - _Verify: npm test --workspace frontend -- projects/portfolio_
  - _Area: api_
  - _Issue: #13_

- [ ] 9. Project plan and weighted progress
  - Deliver screen 06 with editable milestone/task grid, weight-total indicator and readable desktop timeline; use milestone accordions and task cards on phones. Implement atomic plan revisions, assignments, dates, task progress and cancellation with explicit rebalancing.
  - Test positive weights totaling exactly 100, bounded completion, exact weighted results, unconfigured plans and Site Engineer task-only updates. Expose the concrete progress projection used by Project overview.
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_
  - _Depends: 8_
  - _Files: backend/internal/projects/plan/**, frontend/src/app/features/projects/plan/**, docs/api/plan.yaml, docs/evidence/features/plan/**_
  - _Verify: npm run test:backend -- ./internal/projects/plan/..._
  - _Verify: npm test --workspace frontend -- projects/plan_
  - _Area: api_
  - _Issue: #16_

- [ ] 10. Daily site logs submission and review workflow
  - Deliver screens 07–08 with chronological log cards/table, date/site filters and eligible review queue leading into a mobile diary editor, worker rows, entered weather, photo strip and review history. Implement draft, submit, independent approval, reasoned return and resubmission.
  - Test authorized content ownership, invalid project/site references, nonnegative workers, preserved return history, real photo attachment links and approved activity projection without modifying weighted percentages.
  - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_
  - _Depends: 9, 24_
  - _Files: backend/internal/sitework/logs/**, frontend/src/app/features/sitework/logs/**, docs/api/site-logs.yaml, docs/evidence/features/site-logs/**_
  - _Verify: npm run test:backend -- ./internal/sitework/logs/..._
  - _Verify: npm test --workspace frontend -- sitework/logs_
  - _Area: api_
  - _Issue: #20_

- [ ] 11. Inspection to resolution workflow
  - Deliver screens 09–10: inspection register with checklist/result panel and linked corrective-issue action; prioritized issue register with assignment, due date and evidence drawer. Phone cards retain large pass/fail/not-applicable controls and reachable transitions.
  - Implement atomic failed-item issue creation, completion blocking, owner resolution with notes/evidence and independent reasoned reopening. Test checklist lifecycle, retained history and same-project optional milestone/property references against real SQLite.
  - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6, 11.7_
  - _Depends: 9, 20, 24_
  - _Files: backend/internal/sitework/inspections/**, frontend/src/app/features/sitework/inspections/**, docs/api/inspections-issues.yaml, docs/evidence/features/inspections/**_
  - _Verify: npm run test:backend -- ./internal/sitework/inspections/..._
  - _Verify: npm test --workspace frontend -- sitework/inspections_
  - _Area: api_
  - _Issue: #21_

- [ ] 12. Materials and suppliers
  - Deliver screen 11 with clear Materials/Suppliers subviews, searchable tables/cards, separate create/edit panels and scoped history links. Implement unique material codes, explicit units, nonnegative minimums, supplier contacts and reference-preserving archives.
  - Test material unit immutability once referenced and catalog/history projections for each permitted role; use local styles for responsive cards and full-screen mobile editing.
  - _Requirements: 12.1, 12.2, 12.3_
  - _Depends: 2, 4, 25_
  - _Files: backend/internal/procurement/catalog/**, frontend/src/app/features/procurement/catalog/**, docs/api/catalog.yaml, docs/evidence/features/catalog/**_
  - _Verify: npm run test:backend -- ./internal/procurement/catalog/..._
  - _Verify: npm test --workspace frontend -- procurement/catalog_
  - _Area: api_
  - _Issue: #10_

- [ ] 13. Purchase order submission, approval and amendment workflow
  - Deliver screens 13–15: order/review/delivery register, exact line-item creation with Save draft/Submit, and order detail with remaining quantities, amendment comparison, receipt links and approval timeline. Use stacked line cards and collapsible history on phones.
  - Implement supplier/site/material validation, exact totals, submitter-role approval routing, independent amendments and retained cancellation/delivery history. Read receipt totals through the common projection contract; test amendment limits using real prerequisite receipt fixtures and transaction guards.
  - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6_
  - _Depends: 8, 12_
  - _Files: backend/internal/procurement/orders/**, frontend/src/app/features/procurement/orders/**, docs/api/purchase-orders.yaml, docs/evidence/features/purchase-orders/**_
  - _Verify: npm run test:backend -- ./internal/procurement/orders/..._
  - _Verify: npm test --workspace frontend -- procurement/orders_
  - _Area: api_
  - _Issue: #17_

- [ ] 14. Goods receipts and Stock movements transaction workflow
  - Deliver screens 16–17 as the purchase-to-stock posting workflow: approved-order selection, per-line remaining receipt quantities and history; immutable movement ledger beside Issue/Return/Transfer/Adjustment action forms and independent adjustment queue. Phones use stepwise receipt entry and focused movement cards/forms.
  - Implement all-or-nothing receipt/ledger/order updates, excess/unit rejection, activity-linked issues, bounded returns, two-site atomic transfers, negative-stock rejection, approved reasoned adjustments, compensating receipt reversals and persistent duplicate outcomes. Test actual competing writes and injected rollback failures.
  - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7, 14.8, 14.9, 14.10_
  - _Depends: 10, 13, 16, 24_
  - _Files: backend/internal/procurement/receipts/**, backend/internal/inventory/movements/**, frontend/src/app/features/procurement/receipts/**, frontend/src/app/features/inventory/movements/**, docs/api/goods-receipts.yaml, docs/api/stock-movements.yaml, docs/evidence/features/stock-posting/**_
  - _Verify: npm run test:backend -- ./internal/procurement/receipts/..._
  - _Verify: npm run test:backend -- ./internal/inventory/movements/..._
  - _Verify: npm test --workspace frontend -- procurement/receipts_
  - _Verify: npm test --workspace frontend -- inventory/movements_
  - _Area: api_
  - _Issue: #23_

- [ ] 15. Site inventory
  - Deliver screen 12 with site/material filters, explicit quantity/unit/minimum columns, low-stock labels and contributing-ledger links; phones show readable balance cards. Implement authorized balance queries and minimum overrides without independently editable balances.
  - Test ledger-derived balances after every movement kind and reversal, including authorized scope and low-stock thresholds. Do not show a monetary valuation column.
  - _Requirements: 12.4, 12.5_
  - _Depends: 14_
  - _Files: backend/internal/inventory/balances/**, frontend/src/app/features/inventory/balances/**, docs/api/inventory.yaml, docs/evidence/features/inventory/**_
  - _Verify: npm run test:backend -- ./internal/inventory/balances/..._
  - _Verify: npm test --workspace frontend -- inventory/balances_
  - _Area: api_
  - _Issue: #26_

- [ ] 16. Contractor commissioning and approved work workflow
  - Deliver screens 18–21 as one contractor-to-approved-work workflow: directory and contact profile, assigned project/work-order links, creation register and scope/milestone detail with evidence and approval timeline. Use contact/work-order cards and stacked milestone editing on phones.
  - Implement contractor lifecycle, work-order validation, independent Finance approval, completion evidence, reviewed amount revisions and cancellation that preserves financial references. Supply commitment/evidence projections for finance; linked bills use the contracted read interface and show actual empty states until present.
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 15.7_
  - _Depends: 8, 24_
  - _Files: backend/internal/contractors/directory/**, backend/internal/contractors/workorders/**, frontend/src/app/features/contractors/directory/**, frontend/src/app/features/contractors/workorders/**, docs/api/contractors.yaml, docs/api/work-orders.yaml, docs/evidence/features/contractors/**_
  - _Verify: npm run test:backend -- ./internal/contractors/..._
  - _Verify: npm test --workspace frontend -- contractors_
  - _Area: api_
  - _Issue: #18_

- [ ] 17. Project budgets and expense recognition workflow
  - Deliver screens 22–23 with budget/actual/commitment/variance matrix and source drill-down, revision comparison, expense entry/evidence and independent review queue. Phone category and expense cards retain totals and reachable review controls.
  - Implement category budgets/revisions, approved expense recognition, source-cost identity and remaining commitment calculations. Supply shared cost projections to invoices/reporting; test single-counted linked expense/bill sources, rejected expenses and cash separation with real records.
  - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5, 16.6, 16.7_
  - _Depends: 13, 16, 24, 25_
  - _Files: backend/internal/finance/budgets/**, backend/internal/finance/costs/**, frontend/src/app/features/finance/budgets/**, docs/api/budgets-expenses.yaml, docs/evidence/features/budgets-expenses/**_
  - _Verify: npm run test:backend -- ./internal/finance/budgets/..._
  - _Verify: npm run test:backend -- ./internal/finance/costs/..._
  - _Verify: npm test --workspace frontend -- finance/budgets_
  - _Area: api_
  - _Issue: #22_

- [ ] 18. Invoices and bills and exact posting
  - Deliver screen 24 with Receivables/Payables tabs, due/status filters, exact line editor, explicit entered tax, evidence/duplicate-review panel, balances and post/reverse actions; phone summaries retain line cards and totals above actions.
  - Implement checked scaled arithmetic and half-up line rounding, evidence matching, unique references, reviewed duplicate warnings, cost-source linkage, immutable posting and reasoned reversal. Test contractor bills remaining draft without approved work and canonical outstanding/overdue calculations. Implement booking-source contracts against valid fixtures without requiring a browser booking flow yet.
  - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7_
  - _Depends: 14, 17_
  - _Files: backend/internal/finance/invoices/**, frontend/src/app/features/finance/invoices/**, docs/api/invoices.yaml, docs/evidence/features/invoices/**_
  - _Verify: npm run test:backend -- ./internal/finance/invoices/..._
  - _Verify: npm test --workspace frontend -- finance/invoices_
  - _Area: api_
  - _Issue: #27_

- [ ] 19. Payments and receipts and allocation
  - Deliver screen 25 with incoming/outgoing register, reference/method/date entry, invoice allocation grid, available-cash summary and distinct reversal/refund panels; mobile uses payment cards and per-invoice rows with a visible remaining total.
  - Implement posting, atomic compatible allocations, whole-batch rejection, reasoned reversal and separate approved refund records. Expose booking finance operations through the common interface; test approved refund caps using valid prerequisite cancellation fixtures, preserve original collections, and prevent reversal of protected sale evidence without its reviewed correction.
  - _Requirements: 18.1, 18.2, 18.3, 18.4, 18.5, 18.6_
  - _Depends: 18_
  - _Files: backend/internal/finance/payments/**, frontend/src/app/features/finance/payments/**, docs/api/payments.yaml, docs/evidence/features/payments/**_
  - _Verify: npm run test:backend -- ./internal/finance/payments/..._
  - _Verify: npm test --workspace frontend -- finance/payments_
  - _Area: api_
  - _Issue: #30_

- [ ] 20. Property inventory, Property detail and reservations
  - Deliver screens 27–28: searchable image grid/table and filters, creation dialog, large gallery/fact detail, distinct construction/sales labels, price history and reserve panel. Phone image cards, gallery buttons and stacked facts keep reservation controls accessible.
  - Implement unique project/unit identity, durable image links, historic asking prices, one-row unit claims, explicit/default expiry, authorized unavailability and independently reviewed reservation cancellation. Run expiry every minute and at startup before readiness plus inside acquisition transactions; test simultaneous acquisitions, restart catch-up and protection of converted bookings using real SQLite.
  - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5, 19.6, 19.7, 19.8, 19.9_
  - _Depends: 8, 24, 25_
  - _Files: backend/internal/sales/properties/**, backend/internal/sales/reservations/**, frontend/src/app/features/sales/properties/**, docs/api/properties-reservations.yaml, docs/evidence/features/properties/**_
  - _Verify: npm run test:backend -- ./internal/sales/properties/..._
  - _Verify: npm run test:backend -- ./internal/sales/reservations/..._
  - _Verify: npm test --workspace frontend -- sales/properties_
  - _Area: api_
  - _Issue: #19_

- [ ] 21. Enquiry, site visit and customer conversion workflow
  - Deliver screens 29–30 with list/pipeline toggle, accessible stage menus, follow-up filters and enquiry entry leading to contact/preferences, chronological activities, visit composer and customer/booking conversion. Phone stage selection and stacked cards replace drag-only interactions.
  - Implement lead/activity/site-visit/customer endpoints, assigned-sales visibility, contact warnings without automatic merging, planned/actual visit history and conversion to a linked draft booking. Preserve lead/contact inputs on unit conflicts and all relationships through customer archive.
  - _Requirements: 20.1, 20.2, 20.3, 20.4, 20.5, 20.6_
  - _Depends: 22, 25_
  - _Files: backend/internal/sales/leads/**, frontend/src/app/features/sales/leads/**, docs/api/leads-customers-visits.yaml, docs/evidence/features/leads/**_
  - _Verify: npm run test:backend -- ./internal/sales/leads/..._
  - _Verify: npm test --workspace frontend -- sales/leads_
  - _Area: api_
  - _Issue: #34_

- [ ] 22. Booking agreement, collection and cancellation workflow
  - Deliver screen 31 and screen 32's agreement/collection/cancellation sections: active/cancelled register, customer/unit agreement header, installment/receipt ledger, approved terms and cancellation/refund timeline; phone cards and accordion histories retain exact balances and role-specific actions. Prewire the independently owned sale/handover panel from task 23 through its foundation interface.
  - Implement draft/activation and claim conversion, exact installment sum, single-use installment invoicing, Finance term proposals with independent Administrator decisions, cancellation release and preserved sale/receipt history, recorded-band refund calculation/proposal/approval and caps. Test missing terms, excessive refunds, concurrent activation and cancellation without a cash effect.
  - _Requirements: 21.1, 21.2, 21.3, 21.4, 21.5, 21.6, 21.7, 21.8, 21.9_
  - _Depends: 19, 20_
  - _Files: backend/internal/sales/bookings/**, frontend/src/app/features/sales/bookings/**, docs/api/bookings.yaml, docs/evidence/features/bookings/**_
  - _Verify: npm run test:backend -- ./internal/sales/bookings/..._
  - _Verify: npm test --workspace frontend -- sales/bookings_
  - _Area: api_
  - _Issue: #33_

- [ ] 23. Sale confirmation and handover
  - Complete screen 32's separately styled sale/handover panel: agreement upload/date, recorded payment requirements, Finance verification evidence, independent Confirm sale and distinct Project Manager checklist/Finance clearance/Sales handover date. On phones use explicit staged actions, never one ambiguous completion button.
  - Implement the design's sale/payment-verification/checklist/handover endpoints and current-evidence checks. Test creator denial, missing or reversed evidence, later installments versus current milestone requirements, atomic booked-to-sold state and independent handover history.
  - _Requirements: 22.1, 22.2, 22.3, 22.4, 22.5, 22.6_
  - _Depends: 22, 24_
  - _Files: backend/internal/sales/sale/**, frontend/src/app/features/sales/sale/**, docs/api/sale-handover.yaml, docs/evidence/features/sale-handover/**_
  - _Verify: npm run test:backend -- ./internal/sales/sale/..._
  - _Verify: npm test --workspace frontend -- sales/sale_
  - _Area: api_
  - _Issue: #35_

- [ ] 24. Documents and durable evidence
  - Deliver screen 33 with scoped file register, entity links, per-file upload queue, safe preview/download/archive; phones use touch upload and full-screen safe preview. Supply the reusable evidence picker and typed parent-policy validation used by other workflows.
  - Implement verified PDF/JPEG/PNG/WebP limits of 20 MB and 10 files, private quarantine, randomized immutable content, durable metadata/link commits and inherited retention. Test content mismatch, traversal, executable PDF actions, interrupted uploads, failed metadata commits, unauthorized parent/financial access and held evidence archive rejection.
  - _Requirements: 23.1, 23.2, 23.3, 23.4, 23.5_
  - _Depends: 2, 4_
  - _Files: backend/internal/documents/**, frontend/src/app/features/documents/**, frontend/src/app/shared/evidence/**, docs/api/documents.yaml, docs/evidence/features/documents/**_
  - _Verify: npm run test:backend -- ./internal/documents/..._
  - _Verify: npm test --workspace frontend -- documents_
  - _Verify: npm test --workspace frontend -- shared/evidence_
  - _Area: api_
  - _Issue: #6_

- [ ] 25. Company settings and data stewardship
  - Deliver screen 35's company/branding, currency/timezone, reference defaults, project/site overrides, operating-policy, privacy/retention and operator-status sections with validation/history; mobile uses named stacked sections and clear posting blockers. Implement settings, acknowledgments, privacy requests, holds and reviewed retention endpoints, plus the administrator-only operator-status projection.
  - Test explicit currency/timezone setup and currency lock, permitted overrides, minimal personal fields, notice/consent references, required local operating-policy record, hold-safe correction/anonymization and every approved retention period. Status reads the foundation's diagnostics and backup interface; task 28 supplies the backup implementation and failure-state tests. Document live-use prerequisites without inventing company legal values.
  - _Requirements: 24.1, 24.2, 24.3, 24.4, 24.5, 24.6, 24.7, 26.9_
  - _Depends: 2, 4_
  - _Files: backend/internal/settings/**, frontend/src/app/features/administration/settings/**, docs/api/settings-stewardship.yaml, docs/operations/policy.md, docs/evidence/features/settings/**_
  - _Verify: npm run test:backend -- ./internal/settings/..._
  - _Verify: npm test --workspace frontend -- administration/settings_
  - _Area: api_
  - _Issue: #7_

- [ ] 26. Audit history
  - Deliver screen 36 with actor/action/project/date filters, immutable event rows, safe before/after drawer and authorized record links; phones show event cards and labeled changes. Implement paginated audit reads with Administrator, Finance trail and Project Manager approval-trail projections.
  - Test immutability, no secret disclosure, parent-closure seven-year evidence retention, one-year authentication retention and source holds on exports. Audit writes remain owned by the shared writer and each transaction, not by this screen.
  - _Requirements: 25.1, 25.3, 25.4_
  - _Depends: 2, 4, 25_
  - _Files: backend/internal/audit/history/**, frontend/src/app/features/administration/audit/**, docs/api/audit.yaml, docs/evidence/features/audit/**_
  - _Verify: npm run test:backend -- ./internal/audit/history/..._
  - _Verify: npm test --workspace frontend -- administration/audit_
  - _Area: api_
  - _Issue: #11_

- [ ] 27. Management dashboard, Reports and project-cost reconciliation
  - Deliver screens 02 and 26 as the summary-to-source workflow: scoped metric cards and overdue/low-stock panels, report selector/filters, generation time, formula disclosure, result rows, CSV/save actions and reconciliation discrepancy/signoff panels. Phones use stacked summaries and filter drawers, with only labeled detail regions scrolling horizontally.
  - Implement every designed report/view, current committed snapshots, no-more-than-60-second dashboard data, permission-safe drill-downs, canonical totals, overdue classifications, escaped CSV and retained exports. Match supplier and contractor evidence, retain independent Project Manager/Finance signoffs and immutable reconciliation snapshots with supporting documents; test unpaid, partial, cancelled and reversed examples.
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 7.8_
  - _Depends: 9, 10, 11, 15, 17, 18, 19, 21, 23, 26_
  - _Files: backend/internal/reporting/**, frontend/src/app/features/reporting/**, docs/api/reporting.yaml, docs/operations/metrics.md, docs/evidence/features/reporting/**_
  - _Verify: npm run test:backend -- ./internal/reporting/..._
  - _Verify: npm test --workspace frontend -- reporting_
  - _Area: api_
  - _Issue: #38_

- [ ] 28. Persistent local operation, backups and disposable restoration
  - Implement hourly consistent SQLite/attachment snapshots, WAL-aware manifests, content pins/checksums, separate protected-storage validation, 48-hour hourly/30-day daily retention, incomplete-backup rejection and sanitized status. Finish backup/restore-check operator commands and documentation for Sunday 02:00–04:00 company-time maintenance.
  - Test restart preservation, active writes during snapshot, failure reporting through Company settings, restored relationship/hash reconciliation and revoked restored sessions in disposable directories. Provide pre-pilot/monthly drill procedures measuring a recovery point no older than one hour and restoration within four hours; record timing honestly and keep operational directories untouched by checks.
  - _Requirements: 26.6, 28.1, 28.2, 28.3, 28.4, 28.5_
  - _Depends: 24, 25_
  - _Files: backend/internal/platform/backup/**, backend/cmd/manage/backup.go, backend/cmd/manage/restore_check.go, backend/cmd/manage/backup_test.go, tests/integration/restore/**, docs/operations/backup-restore.md, docs/evidence/features/backup/**_
  - _Verify: npm run test:backend -- ./internal/platform/backup/..._
  - _Verify: npm run test:backend -- ./cmd/manage_
  - _Verify: npm run test:integration -- restore_
  - _Area: api_
  - _Issue: #12_

- [ ] 29. Capacity and response-target campaign
  - Implement isolated year-1/year-3 valid-history generators for every quantity in requirement 27.1, including attachment bytes, sessions, tokens and buyer contacts. Run morning-planning, end-of-day and month-end load mixes with 15/40 simultaneous users, recording actual reference hardware, throughput, lock waits, failed requests and latency distributions.
  - Enforce p95 ordinary actions/sign-in within two seconds, dashboards within three seconds, reports within five seconds and 50,000-row export within 30 seconds; measure upload transfer separately. The command starts/stops its own app and uses disposable storage. If hardware/storage is insufficient, fail with an explicit unmet prerequisite and report the full campaign unrun, never replace it with small fixtures and claim acceptance.
  - _Requirements: 27.1, 27.2, 27.3, 27.4_
  - _Depends: 27, 28_
  - _Files: tests/capacity/**, docs/evidence/capacity/**, docs/operations/capacity.md_
  - _Verify: npm run test:capacity_
  - _Area: api_
  - _Issue: #41_

- [ ] 30. Connected acceptance, explicit development fixtures and pilot evidence
  - Finish the explicit idempotent development seed command with at least three projects, approximately twenty units and reconciling related histories; keep credentials runtime-generated in test fixtures and human bootstrap separate. Implement all real-backend connected journeys, seven-role login/denial and recovery cases, UI refresh/restart persistence, concurrency/rollback checks and screenshot/font/keyboard evidence. Compose the 36-screen route/role/API/action/state/evidence matrix, including every required action and the distinct screen 32 sale/handover panel.
  - Run the full routine suite, strict frontend and Go builds and real-browser desktop/mobile checks. Record current/previous browser and physical device coverage honestly, with unavailable manual environments unrun. Provide pilot measurement worksheets for agreed start date, one complete project, weekly baseline/follow-up reporting minutes, the 50% target and zero-critical-defect reconciliation gates; do not claim a 30-day business result before it is observed. Resolve integration wiring within the declared composition files, not by rewriting independently owned feature files.
  - _Requirements: 5.1, 29.1, 29.2, 29.3, 29.4, 29.5, 29.6, 29.7, 29.8, 29.9_
  - _Depends: 6, 7, 27, 28, 29_
  - _Files: backend/internal/seed/**, backend/cmd/manage/seed.go, backend/cmd/manage/seed_test.go, tests/integration/acceptance/**, tests/integration/fixtures/**, tests/integration/support/**, frontend/src/app/app.routes.ts, backend/internal/httpapi/composition.go, docs/evidence/screens.md, docs/evidence/acceptance/**, docs/evidence/pilot/**, docs/operations/development-seed.md, README.md_
  - _Verify: npm run test:backend -- ./internal/seed/..._
  - _Verify: npm run test:integration -- acceptance_
  - _Verify: npm test_
  - _Verify: npm run build_
  - _Area: setup_
  - _Issue: #44_

## Optional generated-input property checks

These leaves supplement required example/transaction/browser tests; no required behavior depends on running them. Use bounded generated cases with reproducible seeds, shrinking or minimized replay traces, injected clocks and actual temporary SQLite. Implement Go `TestPropertyP<number>` tests so `go test -v` reports each property's passing name; generation runs inside those tests rather than invoking an unbounded fuzz session. The root runner from task 1 forwards arguments after the package to Go unchanged. Do not edit implementation files in these testgen leaves.

- [ ]* 31. Property tests for server grants and session revocation
  - Generate roles, scopes, forged client fields and session histories; prove current server grants bound every result and expired/revoked sessions cannot mutate records.
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.6, 2.7, 2.8, 4.4, 4.6, 23.4_
  - _Properties: P1, P2_
  - _Depends: 7, 24_
  - _Files: backend/internal/access/propertysessions/**_
  - _Verify: npm run test:backend -- ./internal/access/propertysessions/... -v_
  - _Mode: testgen_
  - _Issue: #14_

- [ ]* 32. Property tests for password tokens and independent approval
  - Generate token consumption races, setup restrictions, final-Administrator changes and review histories; assert single-use completion, retained Administrator access and atomic independent decisions.
  - _Requirements: 3.2, 3.3, 3.4, 3.6, 3.7, 3.8, 4.3, 6.4, 6.5, 6.6_
  - _Properties: P3, P4_
  - _Depends: 7, 10_
  - _Files: backend/internal/access/propertyrecovery/**_
  - _Verify: npm run test:backend -- ./internal/access/propertyrecovery/... -v_
  - _Mode: testgen_
  - _Issue: #24_

- [ ]* 33. Property tests for duplicate effects and weighted progress
  - Generate stale edits, same/different-payload retries and retained record histories; independently calculate weighted progress and prove approval does not invent milestone completion.
  - _Requirements: 6.1, 6.2, 6.3, 6.7, 9.2, 9.3, 9.4, 9.5, 10.4, 14.10_
  - _Properties: P5, P6_
  - _Depends: 9, 10, 14_
  - _Files: backend/internal/projects/propertyprogress/**_
  - _Verify: npm run test:backend -- ./internal/projects/propertyprogress/... -v_
  - _Mode: testgen_
  - _Issue: #28_

- [ ]* 34. Property tests for inspection corrective history
  - Generate inspection results and issue transitions; unlinked failures cannot complete and resolution/reopening retains required evidence and reasons.
  - _Requirements: 11.2, 11.3, 11.5, 11.6, 11.7_
  - _Properties: P7_
  - _Depends: 11_
  - _Files: backend/internal/sitework/propertyinspections/**_
  - _Verify: npm run test:backend -- ./internal/sitework/propertyinspections/... -v_
  - _Mode: testgen_
  - _Issue: #25_

- [ ]* 35. Property tests for receipts and approved amendments
  - Generate quantity/unit/order revision and reversal histories, including concurrent requests; net receipt never exceeds approved quantities and receipt/order/stock effects roll back together.
  - _Requirements: 13.2, 13.3, 13.5, 13.6, 14.2, 14.3, 14.9_
  - _Properties: P8_
  - _Depends: 14_
  - _Files: backend/internal/procurement/propertyreceipts/**_
  - _Verify: npm run test:backend -- ./internal/procurement/propertyreceipts/... -v_
  - _Mode: testgen_
  - _Issue: #29_

- [ ]* 36. Property tests for stock conservation
  - Compare generated issue/return/transfer/adjustment sequences to an independent ledger model; assert nonnegative balances, compatible units, conserved transfers and independent adjustment approval.
  - _Requirements: 12.4, 12.5, 14.5, 14.6, 14.7, 14.8_
  - _Properties: P9_
  - _Depends: 15_
  - _Files: backend/internal/inventory/propertystock/**_
  - _Verify: npm run test:backend -- ./internal/inventory/propertystock/... -v_
  - _Mode: testgen_
  - _Issue: #31_

- [ ]* 37. Property tests for single-counted approved costs
  - Generate expense/bill/commitment graphs and compare actuals, remaining commitments and variance to an independent source-identity oracle, excluding cash as a second cost.
  - _Requirements: 7.2, 7.4, 15.5, 15.7, 16.2, 16.4, 16.5, 16.6, 16.7_
  - _Properties: P10_
  - _Depends: 27_
  - _Files: backend/internal/finance/propertycosts/**_
  - _Verify: npm run test:backend -- ./internal/finance/propertycosts/... -v_
  - _Mode: testgen_
  - _Issue: #42_

- [ ]* 38. Property tests for exact money and posting prerequisites
  - Generate bounded quantities/prices/tax and invalid evidence/overflow cases; compare half-up totals to a rational oracle and assert posting never bypasses confirmed settings or duplicate review.
  - _Requirements: 17.2, 17.3, 17.4, 17.5, 24.2_
  - _Properties: P11_
  - _Depends: 18_
  - _Files: backend/internal/finance/propertymoney/**_
  - _Verify: npm run test:backend -- ./internal/finance/propertymoney/... -v_
  - _Mode: testgen_
  - _Issue: #32_

- [ ]* 39. Property tests for allocation and reversal balances
  - Generate posted invoice/payment/allocation histories with incompatible batches and reversals; independently model valid entries and assert invoice, cash and booking balances reconcile without partial posting.
  - _Requirements: 17.6, 17.7, 18.2, 18.3, 18.4, 18.5, 21.4_
  - _Properties: P12_
  - _Depends: 22_
  - _Files: backend/internal/finance/propertyallocations/**_
  - _Verify: npm run test:backend -- ./internal/finance/propertyallocations/... -v_
  - _Mode: testgen_
  - _Issue: #36_

- [ ]* 40. Property tests for exclusive unit ownership over time
  - Generate interleaved reservation, activation, cancellation, manual state and expiry/restart operations on separate connections; assert one claim, correct losing conflict and no release of converted bookings.
  - _Requirements: 19.5, 19.6, 19.7, 19.8, 19.9, 21.3, 21.6_
  - _Properties: P13_
  - _Depends: 22_
  - _Files: backend/internal/sales/propertyclaims/**_
  - _Verify: npm run test:backend -- ./internal/sales/propertyclaims/... -v_
  - _Mode: testgen_
  - _Issue: #37_

- [ ]* 41. Property tests for sales history and recorded refund terms
  - Generate price/contact/lead conversion histories plus cancellation bands, collections, reversals and refunds; preserve original agreements/relationships and compare refund caps to an independent bounded oracle.
  - _Requirements: 18.6, 19.4, 20.3, 20.5, 20.6, 21.5, 21.7, 21.8, 21.9_
  - _Properties: P14, P15_
  - _Depends: 21_
  - _Files: backend/internal/sales/propertyhistory/**_
  - _Verify: npm run test:backend -- ./internal/sales/propertyhistory/... -v_
  - _Mode: testgen_
  - _Issue: #39_

- [ ]* 42. Property tests for independent sale and handover evidence
  - Generate actors, active/cancelled bookings and current/stale evidence combinations; sold status and handover are reachable only with every required independent prerequisite.
  - _Requirements: 22.1, 22.2, 22.3, 22.4, 22.5, 22.6_
  - _Properties: P16_
  - _Depends: 23_
  - _Files: backend/internal/sales/propertysale/**_
  - _Verify: npm run test:backend -- ./internal/sales/propertysale/... -v_
  - _Mode: testgen_
  - _Issue: #40_

- [ ]* 43. Property tests for evidence and retention inheritance
  - Generate file validation boundaries, interrupted commits, parent classifications and holds; invalid/incomplete content remains inaccessible and retention/anonymization preserves required evidence and audit access.
  - _Requirements: 23.2, 23.3, 23.4, 23.5, 24.6, 24.7, 25.3, 25.4_
  - _Properties: P17_
  - _Depends: 24, 25, 26_
  - _Files: backend/internal/documents/propertyretention/**_
  - _Verify: npm run test:backend -- ./internal/documents/propertyretention/... -v_
  - _Mode: testgen_
  - _Issue: #15_

- [ ]* 44. Property tests for report and audit truthfulness
  - Generate committed source rows, due dates, hostile CSV text and safe audit changes; prove detail-total equality, date-only overdue effects, spreadsheet escaping and secret-free authorized audit projections.
  - _Requirements: 7.4, 7.6, 7.7, 25.1, 25.2, 25.3_
  - _Properties: P18_
  - _Depends: 27_
  - _Files: backend/internal/reporting/propertyreports/**_
  - _Verify: npm run test:backend -- ./internal/reporting/propertyreports/... -v_
  - _Mode: testgen_
  - _Issue: #43_

- [ ]* 45. Property tests for snapshot restoration and API boundaries
  - Generate committed database/file snapshots with concurrent later writes and incomplete manifests, plus malformed API inputs; restored references/hashes match the snapshot and invalid requests return safe contract errors without partial effects.
  - _Requirements: 26.2, 26.3, 26.4, 26.5, 26.6, 28.2, 28.4, 29.7_
  - _Properties: P19, P20_
  - _Depends: 28, 30_
  - _Files: backend/internal/platform/propertyrestore/**, backend/internal/httpapi/propertyboundaries/**_
  - _Verify: npm run test:backend -- ./internal/platform/propertyrestore/... -v_
  - _Verify: npm run test:backend -- ./internal/httpapi/propertyboundaries/... -v_
  - _Mode: testgen_
  - _Issue: #45_

## Non-generated property evidence

Properties P21–P29 are explicitly not property-testable in the approved design. Required examples and executed evidence cover them: P21 → tasks 4 and 6; P22 → tasks 4–7; P23 → tasks 2 and 30 plus every page's colocated checks; P24 → tasks 8–11; P25 → tasks 12–16; P26 → tasks 17–22; P27 → tasks 24, 25 and 27; P28 → tasks 1, 25, 28 and 29; P29 → task 30. These are not optional acceptance gates.

## Assumptions

- The 30-task ceiling is interpreted as 30 required implementation leaves with 15 additional optional property leaves because literal compliance conflicts with complete approved criterion coverage, separate optional property tasks and the maximum of 10 criteria per leaf; this is an explicit planning exception, not reduced scope.
- Closely related list/create/detail pages are built as one end-to-end workflow leaf, with every page independently routed and styled; Booking detail alone is split into agreement/collection and sale/handover sections with separate ownership because its combined criteria exceed 10.
- Task 1 establishes the full baseline schema, dependency set, API fragments and prewired route/interface slots so parallel feature leaves do not compete for shared wiring; those slots are not delivered features until their owning leaves complete.
- Property campaigns use named bounded Go tests over generated inputs with verbose reporting; routine required examples continue to cover the same business rules when optional campaigns are not selected.
- Capacity and pilot evidence remain measured release gates: unavailable reference-machine capacity, physical browsers or elapsed pilot observations are recorded as unrun or pending rather than inferred from unit tests.
