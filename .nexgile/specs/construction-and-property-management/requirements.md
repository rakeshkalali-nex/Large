# Requirements: BuildEstate — Construction And Property Management

## Release 1 scope and traceability

BuildEstate is one company's internal construction operations and real estate sales application. Release 1 includes the entire baseline below, using Angular with strict TypeScript, a Go HTTP backend, SQLite persistent records, and IBM Plex Sans. Source identifiers refer to `.nexgile/greenfield/answers.md`; the detailed source is `docs/input/Greenfield-Angular-Go-SQLite-Requirements.md`. The brief's decisions and assumptions take precedence where they refine the detailed source.

Criterion identifiers are requirement number plus acceptance-item number, for example 1.1. Shared requirements apply to every relevant page and operation, not just the pages that name them again. All routes below are browser routes; API paths use `/api/v1`. Account recovery and setup are states of Sign in, not additional baseline pages.

### Required page register

| Screen | Page | Route | Feature requirement |
| --- | --- | --- | --- |
| 01 | Sign in | `/login` | 1–3 |
| 02 | Management dashboard | `/dashboard` | 7 |
| 03 | Projects | `/projects` | 8 |
| 04 | Create project | `/projects/new` | 8 |
| 05 | Project overview | `/projects/:id` | 8 |
| 06 | Project plan | `/projects/:id/plan` | 9 |
| 07 | Daily site logs | `/site-logs` | 10 |
| 08 | Site log editor | `/site-logs/new`, `/site-logs/:id` | 10 |
| 09 | Inspections | `/inspections` | 11 |
| 10 | Site issues | `/issues` | 11 |
| 11 | Materials and suppliers | `/materials` | 12 |
| 12 | Site inventory | `/inventory` | 12 |
| 13 | Purchase orders | `/purchase-orders` | 13 |
| 14 | Create purchase order | `/purchase-orders/new` | 13 |
| 15 | Purchase order detail | `/purchase-orders/:id` | 13 |
| 16 | Goods receipts | `/goods-receipts` | 14 |
| 17 | Stock movements | `/stock-movements` | 14 |
| 18 | Contractors | `/contractors` | 15 |
| 19 | Contractor detail | `/contractors/:id` | 15 |
| 20 | Work orders | `/work-orders` | 15 |
| 21 | Work order detail | `/work-orders/:id` | 15 |
| 22 | Project budgets | `/budgets` | 16 |
| 23 | Expenses | `/expenses` | 16 |
| 24 | Invoices and bills | `/invoices` | 17 |
| 25 | Payments and receipts | `/payments` | 18 |
| 26 | Reports | `/reports` | 7 |
| 27 | Property inventory | `/properties` | 19 |
| 28 | Property detail | `/properties/:id` | 19 |
| 29 | Leads and pipeline | `/leads` | 20 |
| 30 | Lead detail | `/leads/:id` | 20 |
| 31 | Bookings | `/bookings` | 21 |
| 32 | Booking detail | `/bookings/:id` | 21–22 |
| 33 | Documents | `/documents` | 23 |
| 34 | Users and access | `/users` | 4 |
| 35 | Company settings | `/settings` | 24 |
| 36 | Audit history | `/audit` | 25 |

### Role and project access baseline

Each staff account has one assigned role. Project-owned data requires authorized project access, including when reached through search, aggregate figures, exports, attachments, or related records. Administrator access to all modules does not bypass independent approvals. Unlisted grants are denied.

| Role | Landing page | Authorized work within scope |
| --- | --- | --- |
| Administrator | `/dashboard` | All modules; company configuration, users, roles and assignments; independent approval of booking cancellations, refund amounts, cancellation terms and stock adjustments |
| Project Manager | `/projects` | Assigned projects/sites, planning, site-log review, inspections, issues, work orders, budget visibility and operational approvals; approve Procurement Officer purchase submissions |
| Site Engineer | `/site-logs` | Assigned site logs, task progress, inspections, issues, material issues/returns and permitted documents; financial totals are withheld |
| Procurement Officer | `/purchase-orders` | Materials, suppliers, purchase orders, receipts, transfers and permitted stock operations; customer finances are withheld |
| Finance Officer | `/invoices` | Budgets, expenses, bills, invoices, tax entry, payments, receipts, allocations, refunds and financial reports; approve Project Manager purchase submissions, work orders and expenses |
| Sales Executive | `/leads` | Assigned property inventory, leads, visits, customers and bookings; only prices, schedules and collection balances needed for assigned sales work |
| Management Viewer | `/dashboard` | Authorized dashboards and reports, with no operational or financial writes |

Finance Officer and Administrator may independently confirm sale milestones within authorized projects. No creator approves their own submission, including an Administrator; a request stays pending when no eligible independent approver is available.

## Requirement 1: Branded Sign in and authorized entry

**User Story:** As a staff member, I want one accessible Sign in page, so that I can enter my authorized work area.

_Source: W1, R1, R2, R3, R4, R5, R6, R7, N2, N3, N4, F1, F4, F5_

### Acceptance Criteria

1. WHEN a visitor opens `/` THE SYSTEM SHALL restore the session before displaying protected content and route an unauthenticated visitor to `/login` or an authenticated staff member to their authorized role landing page.
2. WHEN Sign in is displayed on desktop THE SYSTEM SHALL show a navy architectural brand panel with a licensed building image or line illustration and BuildEstate wordmark beside a warm-light panel containing a 400–440px-wide form, a “Welcome back” heading and a teal “Sign in” action.
3. WHEN Sign in is displayed at tablet or mobile widths THE SYSTEM SHALL use a compact brand header and single-column form with reachable sign-in controls when the software keyboard is open and no page-wide horizontal scrolling.
4. THE SYSTEM SHALL provide visible email/password labels, email input semantics, password-manager autocomplete, permitted paste, an initially masked password and a keyboard-accessible show/hide control that changes visibility without changing the entered password.
5. WHEN staff submit credentials by button or Enter THE SYSTEM SHALL authenticate through the Go API against persisted accounts while announcing a loading state and preventing duplicate submissions.
6. IF credentials are invalid or the account is inactive THEN THE SYSTEM SHALL show the same non-enumerating credential error, retain the entered email and keep protected content inaccessible.
7. IF a connectivity or server failure prevents sign-in THEN THE SYSTEM SHALL retain the email, announce a safe service-failure message distinct from invalid credentials and require deliberate resubmission.
8. WHEN sign-in succeeds or an authenticated staff member opens `/login` THE SYSTEM SHALL open the authorized internal return route if valid, otherwise the role landing page in the access baseline, rejecting external and malformed return URLs.
9. WHEN staff select account recovery or need setup, reset completion or session-expiry guidance THE SYSTEM SHALL show the corresponding actionable state within the branded account shell with accessible focus and error announcements.

## Requirement 2: Secure sessions and server-determined access

**User Story:** As an Administrator, I want the server to enforce access, so that staff see only permitted records and actions.

_Source: W1, BR1, BR3, EV2, EV3, E1, E25, N2, N5, N10_

### Acceptance Criteria

1. WHEN credentials are verified THE SYSTEM SHALL determine account status, the single assigned role and effective project permissions on the server and return only the minimal identity and permissions needed by the interface.
2. WHEN a protected page, API, action, document, search, report or export is requested THE SYSTEM SHALL enforce the role and project access baseline on the server independently of any browser-supplied role or permission value.
3. IF access lacks authentication or authorization THEN THE SYSTEM SHALL return respectively HTTP 401 or HTTP 403, using a consistent non-disclosing HTTP 404 for inaccessible individual resources and an accessible access-denied page with a safe destination for denied navigation.
4. WHEN a session is created THE SYSTEM SHALL rotate its opaque identifier, persist only its token hash and lifecycle metadata, and deliver its token in an HttpOnly cookie with explicit SameSite behavior, Secure protection for HTTPS operation and CSRF protection for state-changing requests.
5. WHEN a protected page is refreshed THE SYSTEM SHALL restore identity through `GET /api/v1/auth/me` before showing protected data or authorized navigation.
6. IF a session reaches 30 minutes of inactivity or 12 hours since creation THEN THE SYSTEM SHALL require sign-in again with an expiry message and no automatic replay of financial or stock mutations.
7. WHEN staff log out THE SYSTEM SHALL revoke the server session, clear authenticated interface state and deny further use of that session.
8. WHEN a role, project assignment, password or active status changes THE SYSTEM SHALL invalidate affected sessions or re-evaluate permissions before their next protected operation so obsolete access is denied.
9. WHEN login or reset attempts exceed the documented account-identifier and origin limits THE SYSTEM SHALL throttle attempts with safe responses that do not reveal whether an account exists.

## Requirement 3: Password setup, change and assisted recovery

**User Story:** As a staff member, I want working password setup and recovery, so that I can regain access without exposing my password.

_Source: W2, BR13, E26, N10, F5_

### Acceptance Criteria

1. WHEN the first Administrator account is initialized THE SYSTEM SHALL use a documented local bootstrap process accepting private operator-supplied credentials and storing an Argon2id password hash.
2. WHEN an Administrator creates a staff account THE SYSTEM SHALL require initial password setup using temporary credentials or a single-use setup link expiring after 24 hours.
3. WHILE an account requires initial password setup THE SYSTEM SHALL allow only setup completion and logout.
4. WHEN a password is created or changed THE SYSTEM SHALL require at least 12 characters and store only its Argon2id hash.
5. WHEN staff request recovery guidance THE SYSTEM SHALL explain how to contact an Administrator for identity verification and delivery of a reset link through the documented trusted channel.
6. WHEN an Administrator issues recovery through `POST /api/v1/users/{id}/password-reset` THE SYSTEM SHALL create a single-use token expiring after 30 minutes, persist only its hash and lifecycle metadata, and permit its delivery through the trusted channel without exposing an existing password.
7. IF a setup or reset token is expired, revoked or consumed THEN THE SYSTEM SHALL reject it without changing the password and direct staff to obtain a replacement after identity verification.
8. WHEN staff complete setup, authenticated password change or recovery THE SYSTEM SHALL consume applicable tokens, invalidate prior sessions and record a secret-free audit event before access can resume with the new password.

## Requirement 4: Users and access

**User Story:** As an Administrator, I want to manage staff and assignments, so that access follows current responsibilities.

_Source: W2, R1, BR1, BR2, BR3, A7, A8, F2_

### Acceptance Criteria

1. WHEN an Administrator opens Users and access THE SYSTEM SHALL show a searchable staff list with account status, one role, project assignments and working create, edit, recovery and deactivation actions.
2. WHEN staff access is saved THE SYSTEM SHALL validate the account identity, role and project references and preserve setup-required, active and deactivated lifecycle history.
3. IF a change would remove or deactivate the final active Administrator THEN THE SYSTEM SHALL reject the change without altering that account or its access.
4. WHEN an authorized deactivation is confirmed THE SYSTEM SHALL prevent the account's next protected operation and retain business records and audit identity attributed to that account.
5. IF a role needs assigned projects but has none THEN THE SYSTEM SHALL display a no-projects-assigned state without exposing another project's records.
6. IF a non-Administrator attempts to assign roles, change project access or issue another user's recovery token THEN THE SYSTEM SHALL deny the operation on the API as well as withhold its interface control.

## Requirement 5: Shared visual system and accessible page behavior

**User Story:** As a staff member, I want consistent, finished pages across devices, so that I can perform office and site work with the same controls.

_Source: S2, N3, N4, N12, N13, F1, F4, F5, F13_

### Acceptance Criteria

1. THE SYSTEM SHALL provide every page in the required page register as a distinct routable business view with real API-backed content and working visible actions.
2. THE SYSTEM SHALL use a light architectural visual system with navy `#14283F` navigation, teal `#0F766E` primary actions, warm `#F5F6F3` backgrounds, white `#FFFFFF` surfaces, slate `#243447` text and restrained amber `#B45309` accents, with accessible contrast adjustments documented.
3. THE SYSTEM SHALL load licensed IBM Plex Sans assets for used weights 400–700, apply the font to headings, text, controls, tables, dialogs and overlays, and use swap loading with fallback fonts only during loading or failure.
4. THE SYSTEM SHALL provide 8px-based spacing, 8–12px radii, a readable 16px body baseline, aligned financial columns with tabular numerals, coherent accessible icons and consistent hover, focus, pressed, disabled, loading and error states.
5. WHEN staff navigate the application THE SYSTEM SHALL show permission-filtered Overview, Construction, Procurement, Contractors, Finance, Real Estate and Administration navigation, a collapsible desktop sidebar, project context, user menu, current page title, active navigation and detail-page breadcrumbs.
6. WHEN a page loads or an action completes THE SYSTEM SHALL display the applicable loading, empty, success, field-validation, server-error, permission-denied or not-found state, preserving recoverable input, blocking duplicate submissions and confirming destructive actions.
7. WHEN workflows are exercised at 1440px, 1024px, 768px and 390px widths THE SYSTEM SHALL keep controls reachable without page-wide overflow or clipping, adapt tables and navigation, and support mobile site forms and photo attachments.
8. THE SYSTEM SHALL meet WCAG 2.2 AA for required workflows with semantic landmarks, visible labels and focus, keyboard operation, status cues beyond color, announced errors, dialog/route focus management, zoom support and reduced-motion behavior.
9. THE SYSTEM SHALL support current and previous major desktop Chrome, Edge, Firefox and Safari plus current Android Chrome and iOS Safari, with English left-to-right text, UTC timestamp storage, company-timezone display, distinct date-only values and configured currency formatting.
10. IF connectivity is lost THEN THE SYSTEM SHALL show the failure, preserve safe unsaved values within the current page and require deliberate retry before any authoritative mutation.

## Requirement 6: Transaction integrity and independent approvals

**User Story:** As a Project Manager, I want concurrent work and reviews to preserve history, so that repeated submissions do not corrupt records.

_Source: BR12, EV4, A7, A8, E24, S7_

### Acceptance Criteria

1. WHEN a business record is created or changed THE SYSTEM SHALL retain its ID, creator, creation/update timestamps and version, including a project reference for project-owned records.
2. IF an important edit uses a stale version or requests an invalid lifecycle transition THEN THE SYSTEM SHALL return a safe HTTP 409 Conflict without overwriting committed work and preserve entered values for correction.
3. WHEN approvals, receipts, stock movements, reservations, bookings or financial postings are submitted THE SYSTEM SHALL use transactional state checks and persistent duplicate protection so retries cannot repeat their business effect.
4. WHEN a review decision commits THE SYSTEM SHALL atomically retain the resulting state, reviewer, time, reason when required and audit event, with rollback of all effects on failure.
5. WHEN a site log, purchase order, work order, expense or cancellation is submitted THE SYSTEM SHALL place it in the permission-filtered review queue and keep it pending until an eligible independent reviewer acts.
6. IF a creator attempts to approve their own submission THEN THE SYSTEM SHALL reject the approval even when the creator is an Administrator.
7. WHEN a referenced record is retired THE SYSTEM SHALL preserve its relationships and history through archiving, cancellation or reversal according to its lifecycle instead of hard deletion.

## Requirement 7: Management dashboard and Reports

**User Story:** As a Management Viewer, I want reconciled summaries and drill-downs, so that I can compare authorized projects without changing operations.

_Source: G1, G2, RP1, RP2, BR9, EV5, R7, F11_

### Acceptance Criteria

1. WHEN Management dashboard opens THE SYSTEM SHALL show authorized project status, physical progress, cost summaries, overdue work, low stock, available units, sales and collection metrics with source-linked drill-downs and data no more than 60 seconds old.
2. WHEN a metric is displayed THE SYSTEM SHALL document its formula and source records and distinguish physical progress, booked/sold units, approved costs, open commitments, invoiced amounts and collected cash.
3. WHEN Reports opens THE SYSTEM SHALL offer permission-filtered operational, procurement, budget, inventory, sales and collection reports with project/date filters, generation time, active filter values, refresh and CSV export.
4. WHEN a report is generated THE SYSTEM SHALL query committed source records and reconcile totals with detail rows, including unpaid, partially paid, cancelled and reversed records without double-counting linked expense and bill sources.
5. WHEN role-specific reports are selected THE SYSTEM SHALL provide Project Managers progress/overdue/inspection views, Procurement Officers low-stock/movement/outstanding-order views, Finance Officers commitments/cost/variance/payable/receivable views, Sales Executives assigned availability/follow-up/booking/customer-balance views and Management Viewers authorized summaries.
6. WHEN a CSV export is produced THE SYSTEM SHALL apply the source report's permissions, filters and retention holds and escape user-controlled spreadsheet formula content.
7. WHEN invoice, task or lead follow-up dates pass THE SYSTEM SHALL identify outstanding items as overdue in lists and summaries without changing financial amounts.
8. WHEN Project Managers and Finance Officers reconcile project costs THE SYSTEM SHALL support matching supplier bills to orders/receipts and contractor bills to approved work, identifying missing or duplicate evidence, agreeing totals and saving the reconciliation report with supporting documents and audit history.

## Requirement 8: Projects, Create project and Project overview

**User Story:** As a Project Manager, I want to manage projects and sites, so that delivery records share one project context.

_Source: W3, E2, E28, A8, G1_

### Acceptance Criteria

1. WHEN Projects opens THE SYSTEM SHALL show a searchable, filterable authorized portfolio with project code, name, manager, dates, status, progress and permitted create/archive actions.
2. WHEN Create project is submitted THE SYSTEM SHALL validate and persist a unique project code, name, address, project type, valid date range, responsible manager, status and initial site or block configuration.
3. WHEN a Project Manager maintains sites or blocks THE SYSTEM SHALL retain project-unique codes, names, addresses and active/archived status for use by site work, inventory and property records.
4. WHEN Project overview opens THE SYSTEM SHALL show calculated progress, dates, sites, approved recent activity and permitted financial summaries with links to plan, site work, costs, materials, property units and documents.
5. WHEN project status changes THE SYSTEM SHALL preserve the draft, active, on-hold, completed and archived lifecycle history and retain referenced records when archived.
6. WHEN a Site Engineer is assigned to a project THE SYSTEM SHALL make only the assigned project's permitted work available after the server applies the assignment.

## Requirement 9: Project plan and weighted progress

**User Story:** As a Project Manager, I want milestones and tasks with visible schedules, so that project progress is explainable.

_Source: W3, E3, BR9, A8_

### Acceptance Criteria

1. WHEN Project plan opens THE SYSTEM SHALL display editable milestone/task rows and a readable timeline with responsible person, planned dates, due status, lifecycle status, completion percentage and milestone weight.
2. WHEN a plan is saved THE SYSTEM SHALL validate assigned people, date ranges, completion percentages between 0 and 100 and positive milestone weights totaling 100 for the active progress plan.
3. WHEN project progress is calculated THE SYSTEM SHALL show the sum of each active milestone's completion percentage multiplied by its weight divided by 100, with an explicit unconfigured state when no valid active progress plan exists.
4. WHEN a milestone or task changes THE SYSTEM SHALL retain planned, in-progress, completed and cancelled history and require an explicitly rebalanced active plan before a weight-bearing milestone is cancelled.
5. WHEN an authorized Site Engineer updates task progress THE SYSTEM SHALL persist the permitted update and display the resulting plan without granting financial or plan-administration privileges.

## Requirement 10: Daily site logs and Site log editor

**User Story:** As a Site Engineer, I want to submit daily site records for review, so that approved work appears in project history.

_Source: W3, E4, EV4, A8, N3_

### Acceptance Criteria

1. WHEN Daily site logs opens THE SYSTEM SHALL show project/date filters, work summaries, statuses and the review queue permitted for the signed-in role.
2. WHEN Site log editor opens THE SYSTEM SHALL provide project, site, date, work performed, nonnegative worker counts, entered weather, blockers, notes and optional photograph controls in a mobile-capable daily diary layout.
3. WHEN a Site Engineer saves or submits a log THE SYSTEM SHALL persist its content as draft or submitted with project/site validation and visible submission history.
4. WHEN an independent authorized Project Manager approves a submitted log THE SYSTEM SHALL mark it approved and display its activity in Project overview without treating approval alone as a change to weighted milestone percentages.
5. WHEN a reviewer returns an incomplete log THE SYSTEM SHALL require a reason, preserve content and review history, and permit correction and resubmission from the returned state.

## Requirement 11: Inspections and Site issues

**User Story:** As a Site Engineer, I want failed checks linked to corrective work, so that inspection findings reach a documented resolution.

_Source: W8, E5, E6, A8_

### Acceptance Criteria

1. WHEN Inspections opens THE SYSTEM SHALL provide project-filtered inspections and checklist editing with pass, fail and not-applicable results, findings and optional milestone/property-unit links.
2. WHEN a failed checklist item is recorded THE SYSTEM SHALL support creation of a linked corrective issue with owner, priority, due date and project.
3. IF a failed item lacks a linked corrective issue THEN THE SYSTEM SHALL block closure of that item and inspection completion.
4. WHEN Site issues opens THE SYSTEM SHALL show a prioritized, filterable issue list with assignments, deadlines, status, inspection links and actions permitted to the current user.
5. WHEN an authorized owner resolves an issue THE SYSTEM SHALL require resolution notes and evidence while retaining open, in-progress and resolved transitions.
6. WHEN an authorized reviewer reopens an issue THE SYSTEM SHALL require a reason and preserve the earlier resolution and inspection history in the reopened state.
7. WHEN an inspection moves through draft, submitted and completed states THE SYSTEM SHALL validate its checklist and corrective links and retain actor/time history.

## Requirement 12: Materials and suppliers and Site inventory

**User Story:** As a Procurement Officer, I want a material catalog and derived site balances, so that purchasing follows actual stock needs.

_Source: W4, E7, E8, E11, BR4, A8_

### Acceptance Criteria

1. WHEN Materials and suppliers opens THE SYSTEM SHALL provide clear searchable material and supplier subviews with working create/edit/archive controls and links to procurement history.
2. WHEN a material is saved THE SYSTEM SHALL validate its unique code, name, unit of measure and nonnegative minimum-stock level.
3. WHEN a supplier is saved THE SYSTEM SHALL persist supplier details, contacts and active/archived status without removing its referenced purchase history.
4. WHEN Site inventory opens THE SYSTEM SHALL show material balances by authorized site, explicit units, minimum-stock levels, low-stock filtering and links to the contributing ledger entries.
5. WHEN a stock balance is displayed THE SYSTEM SHALL derive it from posted receipts, issues, returns, transfers and adjustments rather than an independently editable balance.

## Requirement 13: Purchase orders, creation and detail

**User Story:** As a Procurement Officer, I want approved purchase orders with remaining quantities, so that deliveries follow authorized purchasing.

_Source: W4, E9, BR4, A7, A8, EV4_

### Acceptance Criteria

1. WHEN Purchase orders opens THE SYSTEM SHALL show supplier/project filters, approval status, delivery status, outstanding quantities and links to create and detail views.
2. WHEN Create purchase order is saved THE SYSTEM SHALL validate supplier, project/site, material lines, positive quantities, compatible units, exact unit prices and delivery dates and show calculated line totals and order total.
3. WHEN a purchase order is submitted THE SYSTEM SHALL route a Procurement Officer submission to an independent authorized Project Manager and a Project Manager submission to an independent authorized Finance Officer, preserving Administrator authority subject to independent approval.
4. WHEN Purchase order detail opens THE SYSTEM SHALL show line items, approval history, amendments, linked receipts and ordered, received and remaining quantities.
5. WHEN an authorized amendment changes an approved order THE SYSTEM SHALL retain prior values and approval history and reject quantities below already received amounts.
6. WHEN delivery or cancellation changes an order THE SYSTEM SHALL preserve draft, submitted, approved, part-received, received and cancelled history and prevent cancellation from erasing posted receipts or stock effects.

## Requirement 14: Goods receipts and Stock movements

**User Story:** As a Procurement Officer, I want receipt and stock transactions to reconcile, so that material movements cannot create impossible balances.

_Source: W4, E10, E11, BR4, BR12, A7, A8_

### Acceptance Criteria

1. WHEN Goods receipts opens THE SYSTEM SHALL offer approved-order selection, per-line remaining quantities, partial/full delivery entry and receipt history.
2. WHEN a valid receipt is posted THE SYSTEM SHALL atomically retain the receipt, append linked stock entries, update remaining quantities and derive part-received or received order status.
3. IF a receipt exceeds the remaining approved quantity or uses incompatible units THEN THE SYSTEM SHALL reject the entire receipt, preserve entered values and identify the permitted quantity or unit for correction.
4. WHEN Stock movements opens THE SYSTEM SHALL provide permitted issue, return, transfer and adjustment controls beside immutable ledger history identifying material, site, quantity, unit, source transaction, actor and time.
5. WHEN stock is issued or returned THE SYSTEM SHALL link it to its site activity or original movement and append the corresponding ledger entries without modifying historical entries.
6. IF an issue, transfer or adjustment would make stock negative THEN THE SYSTEM SHALL reject the entire operation and display available stock for correction.
7. WHEN a transfer is posted THE SYSTEM SHALL commit the source decrease and destination increase together with compatible units and authorized access to both sites.
8. WHEN a stock adjustment is posted THE SYSTEM SHALL require a reason and independent Administrator approval and preserve both with the appended entry.
9. WHEN a posted receipt is reversed THE SYSTEM SHALL preserve its original history, append compensating stock entries and reject a reversal that violates stock availability.
10. WHEN the same receipt or movement submission is retried THE SYSTEM SHALL return its recorded outcome without adding another receipt or ledger effect.

## Requirement 15: Contractors and Work orders

**User Story:** As a Project Manager, I want contractor commitments linked to approved work and bills, so that contracted costs can be reconciled.

_Source: W5, E12, E13, A7, A8, EV4_

### Acceptance Criteria

1. WHEN Contractors opens THE SYSTEM SHALL display a searchable directory with contacts, trade categories, active status and permitted create/edit/archive actions.
2. WHEN Contractor detail opens THE SYSTEM SHALL show its profile, assigned projects, work orders, permitted related bills and linked documents.
3. WHEN Work orders opens THE SYSTEM SHALL show a searchable register with project, contractor, status, dates and working creation controls.
4. WHEN a work order is saved THE SYSTEM SHALL validate project, contractor, scope, milestones, agreed amounts, dates and responsible manager and retain draft/submitted state.
5. WHEN a submitted work order is approved THE SYSTEM SHALL require independent Finance Officer or Administrator authorization and record approved amounts and evidence.
6. WHEN Work order detail opens THE SYSTEM SHALL show scope, agreed amounts, milestones, completion progress, approval history and linked expenses, bills and documents.
7. WHEN approved amounts, completion or lifecycle status change THE SYSTEM SHALL preserve authorized change history through approved, in-progress, completed and cancelled states without erasing linked financial records.

## Requirement 16: Project budgets and Expenses

**User Story:** As a Finance Officer, I want budgets and approved expenses separated from commitments and cash, so that project cost comparisons are accurate.

_Source: W5, E14, E15, BR9, A7, A8, EV4, F11_

### Acceptance Criteria

1. WHEN Project budgets opens THE SYSTEM SHALL show project/period filters and category budget, approved actual cost, open commitment and variance columns with source drill-downs.
2. WHEN Finance saves or revises a project budget THE SYSTEM SHALL validate category amounts and retain draft, approved and revised versions with approval history.
3. WHEN Expenses opens THE SYSTEM SHALL provide project/category filters, expense entry and a permission-filtered review queue showing references, supporting documents, linked sources and status.
4. WHEN an expense is submitted THE SYSTEM SHALL validate project, category, amount, source references and supporting evidence before placing it in the review queue.
5. WHEN Finance or an Administrator independently approves or rejects an expense THE SYSTEM SHALL record the decision and rejection reason, with only approved expenses contributing to approved actual costs.
6. WHEN an expense and bill represent the same source cost THE SYSTEM SHALL link their sources and count that cost once while keeping payment cash movements separate.
7. WHEN remaining commitments and budget variance are shown THE SYSTEM SHALL calculate open approved order/work-order amounts not already recognized as actual cost, and budget minus approved actual costs minus open commitments, with both source sets visible.

## Requirement 17: Invoices and bills and exact money

**User Story:** As a Finance Officer, I want controlled posting and exact amounts, so that receivables and payables remain explainable.

_Source: W5, W7, E16, BR5, BR6, BR9, BR11, A9_

### Acceptance Criteria

1. WHEN Invoices and bills opens THE SYSTEM SHALL separate receivables and payables with project/counterparty filters, draft/posted/reversed status, due dates, balances and working line-item, posting and reversal controls.
2. WHEN an invoice or bill draft is saved THE SYSTEM SHALL validate its unique reference, project, buyer/supplier/contractor, source links, line amounts, due date and nonnegative staff-entered tax amount without deriving tax from configured rates.
3. WHEN financial amounts are calculated THE SYSTEM SHALL use one configured company base currency and integer minor-unit totals, quantities to three decimal places and unit prices to four, rounding each line half-up to the currency minor unit before summing rounded lines and entered tax.
4. IF a contractor bill lacks approved work evidence or a possible duplicate supplier reference has not been reviewed THEN THE SYSTEM SHALL keep it draft and identify the required review without adding a posted cost.
5. WHEN Finance or an Administrator posts an invoice or bill THE SYSTEM SHALL validate source evidence, category and current state and atomically record its amount, source-cost linkage and audit history.
6. WHEN a posted invoice or bill requires correction THE SYSTEM SHALL require a reasoned reversal retaining the original record, allocations and audit trail rather than permitting amount edits or deletion.
7. WHEN a receivable or payable balance is displayed THE SYSTEM SHALL derive it from posted amounts, valid allocations and reversals, with overdue status based on its due date and outstanding amount.

## Requirement 18: Payments and receipts and allocation

**User Story:** As a Finance Officer, I want traceable manual collections and payments, so that customer and counterparty balances reconcile.

_Source: W5, W7, E17, BR5, BR6, BR8, A9_

### Acceptance Criteria

1. WHEN Payments and receipts opens THE SYSTEM SHALL show incoming/outgoing records, project/counterparty filters, reference, date, method, amount, allocation status and permitted posting, allocation, reversal and refund actions.
2. WHEN a manual payment or receipt is saved THE SYSTEM SHALL validate amount, project, customer or counterparty, method, date and reconciliation reference and retain draft/posted/reversed history.
3. WHEN a posted payment or receipt is allocated THE SYSTEM SHALL atomically link it to compatible posted invoices in the same project and for the same customer or counterparty, within both available cash and invoice outstanding amounts.
4. IF an allocation is excessive or incompatible THEN THE SYSTEM SHALL reject the entire allocation request without partial posting and display the balance or relationship needing correction.
5. WHEN a posted payment, receipt or allocation is reversed THE SYSTEM SHALL require a reason, preserve the original trail and recompute invoice, cash and customer balances from valid records.
6. WHEN Finance records an independently approved refund THE SYSTEM SHALL create a separate outgoing record linked to its cancellation and original collections, preserving original receipts and updating paid and remaining refund amounts.

## Requirement 19: Property inventory, Property detail and reservations

**User Story:** As a Sales Executive, I want current unit availability and controlled reservations, so that two buyers cannot acquire the same unit.

_Source: W6, E18, E21, BR7, BR9, BR16, EV1, EV6, A8_

### Acceptance Criteria

1. WHEN Property inventory opens THE SYSTEM SHALL show a searchable grid/table with project, property type, price and availability filters, meaningful images/unit facts and permitted creation controls.
2. WHEN a unit is saved THE SYSTEM SHALL validate a project-unique unit code, block/tower, relevant floor, configurable apartment/villa/commercial-unit/plot type, area and area unit, asking price, description, images and construction status separately from sales availability.
3. WHEN Property detail opens THE SYSTEM SHALL show larger image areas, unit facts, construction status, available/reserved/booked/sold/unavailable state, price history, related booking and permitted reservation actions.
4. WHEN asking price changes THE SYSTEM SHALL retain its history without changing existing booking agreements, installments, invoices or collected amounts.
5. WHEN an available unit is reserved THE SYSTEM SHALL atomically link a customer and explicit expiry time, defaulting to 48 hours unless a permitted project/site override exists, and enforce one active reservation/booking relationship through persistent transaction safeguards.
6. WHEN a reservation expires THE SYSTEM SHALL release only its unconverted unit, checking at least every minute and before every acquisition and processing missed expiries after restart before accepting new reservations.
7. IF simultaneous requests compete for the same available unit THEN THE SYSTEM SHALL commit only one acquisition and return a clear HTTP 409 Conflict with refreshed availability to the other request while preserving its lead/contact input.
8. WHEN an authorized staff member manually marks a unit unavailable THE SYSTEM SHALL allow the change only without an active reservation or booking and preserve the reason and availability history.
9. WHEN a reservation converts or an expiry or approved cancellation releases it THE SYSTEM SHALL apply respectively reserved-to-booked or reserved-to-available transitions without releasing a converted booking; release of a sold unit requires approved cancellation.

## Requirement 20: Leads and pipeline and Lead detail

**User Story:** As a Sales Executive, I want enquiries, follow-ups and visits in one history, so that booking conversion preserves buyer context.

_Source: W6, E19, E20, E27, A8, EV5_

### Acceptance Criteria

1. WHEN Leads and pipeline opens THE SYSTEM SHALL offer searchable/filterable list and pipeline views, enquiry entry, salesperson assignments, next follow-ups and new/contacted/qualified/visit/booked/lost stage changes.
2. WHEN a lead is saved THE SYSTEM SHALL validate and retain contact information, source, assigned Sales Executive, budget range, preferred project/unit, stage, notes and next follow-up date.
3. IF a contact resembles an existing record THEN THE SYSTEM SHALL warn of a possible duplicate without merging or rejecting people solely because they share a phone number.
4. WHEN Lead detail opens THE SYSTEM SHALL show contact profile, preferences, chronological activities, follow-ups, site visits and permitted customer/booking conversion actions.
5. WHEN a site visit is scheduled or completed THE SYSTEM SHALL retain lead, project, optional unit, assigned Sales Executive, scheduled time, actual time, outcome and notes with scheduled/completed/cancelled history.
6. WHEN a lead converts THE SYSTEM SHALL create or link the customer and booking without losing the lead relationship, activities or visits and retain active/archived customer records with their reservations, bookings, invoices and receipts.

## Requirement 21: Bookings, installments and cancellations

**User Story:** As a Sales Executive, I want bookings to preserve agreements and cancellation decisions, so that customers' commitments and balances stay traceable.

_Source: W6, W7, E22, BR7, BR8, BR10, BR16, A7, A8, A9_

### Acceptance Criteria

1. WHEN Bookings opens THE SYSTEM SHALL show active and cancelled records with customer/unit/project filters, dates, installments, outstanding amounts and links to Booking detail.
2. WHEN Booking detail opens THE SYSTEM SHALL show customer, unit, salesperson, agreed price, deposit expectations, dates, installment schedule, linked invoices/receipts, cancellation terms, refund status, sale evidence and handover checklist.
3. WHEN a draft booking is activated THE SYSTEM SHALL require negotiated price, explicit installments totaling that price and independently approved recorded cancellation terms, atomically acquiring the available unit or converting its eligible reservation to booked.
4. WHEN installments are billed THE SYSTEM SHALL create linked receivable invoices whose allocations and reversals determine the booking's remaining collection balance without changing the agreed schedule silently.
5. WHEN Finance proposes cancellation terms THE SYSTEM SHALL record effective-date bands with either a fixed deduction or a percentage of agreed price and require an independent Administrator's approval before activation or use.
6. WHEN a cancellation is requested and independently approved by an Administrator THE SYSTEM SHALL retain cancellation-requested/cancelled history, record the reason, release the unit transactionally and preserve original sale, invoice and receipt records.
7. WHEN a cancelled booking's refundable amount is calculated THE SYSTEM SHALL apply its recorded effective-date deduction to net valid collections, subtract refunds already paid, clamp the remaining amount to zero through unrefunded collections and present it separately from refunds paid.
8. IF cancellation terms are missing THEN THE SYSTEM SHALL block refund calculation and require independently approved terms without guessing a deduction or changing collected amounts.
9. WHEN Finance proposes a refund amount THE SYSTEM SHALL require independent Administrator approval and prevent recording payments exceeding the approved remaining refundable amount.

## Requirement 22: Sale confirmation and handover

**User Story:** As a Finance Officer, I want independent sale confirmation and separate handover checks, so that sold status and possession records have evidence.

_Source: W9, BR14, BR15, E22, A13, A14_

### Acceptance Criteria

1. WHEN Sales records a sale milestone THE SYSTEM SHALL retain the active booking's attached signed agreement, milestone date and recorded payment requirements.
2. WHEN Finance verifies milestone payments THE SYSTEM SHALL compare the booking's requirements to posted receipts and retain verifier, timestamp and evidence references.
3. WHEN an authorized Finance Officer or Administrator other than the booking creator confirms the milestone THE SYSTEM SHALL require active booking status, signed agreement, milestone date and Finance verification before atomically changing booked to sold with confirmer/time/evidence history.
4. IF sale evidence, independent confirmation or required payments are missing THEN THE SYSTEM SHALL reject confirmation without changing unit availability, including when later installments exist but the milestone's own requirements have not been met.
5. WHEN the Project Manager completes the handover checklist THE SYSTEM SHALL retain each completion result and supporting evidence separately from sale confirmation.
6. WHEN Sales records the handover date THE SYSTEM SHALL require confirmed sale, completed Project Manager checklist and Finance clearance of recorded handover payment requirements and retain the completion history.

## Requirement 23: Documents and durable evidence

**User Story:** As a staff member, I want authorized attachments linked to work, so that evidence remains available with its business record.

_Source: E23, N1, N6, BR1, A8_

### Acceptance Criteria

1. WHEN Documents opens THE SYSTEM SHALL display permission-filtered files with project/entity links, upload, safe preview, download and permitted archive controls.
2. WHEN a file is submitted THE SYSTEM SHALL accept only content-verified PDF, JPEG, PNG or WebP files of at most 20 MB each, with at most 10 files per submission, rejecting path traversal, executable content and mismatched types.
3. WHEN an upload completes THE SYSTEM SHALL persist metadata and entity links in SQLite and content under randomized names in durable local attachment storage, keeping incomplete uploads quarantined and unavailable.
4. WHEN an attachment is listed, previewed or downloaded THE SYSTEM SHALL enforce its parent record's project and financial restrictions and serve content without executable interpretation or disclosure of storage paths.
5. WHEN a parent record is archived or retained THE SYSTEM SHALL preserve linked financial evidence and apply the parent's retention and legal holds to its documents.

## Requirement 24: Company settings and data stewardship

**User Story:** As an Administrator, I want validated company settings and controlled personal records, so that operational rules and retention are explicit.

_Source: F2, E1, A10, A11, N12, BR6_

### Acceptance Criteria

1. WHEN Company settings opens THE SYSTEM SHALL provide company details, branding, base currency, timezone, operational defaults, property types, inspection checklists and reference values with validated save controls and change history.
2. WHEN initial operational setup is completed THE SYSTEM SHALL require an explicitly chosen base currency and confirmed timezone, initially presenting UTC, before operational posting and lock currency changes after the first financial posting.
3. WHEN project/site defaults are configured THE SYSTEM SHALL permit reservation duration, stock minimum, checklist and operational-reference overrides without allowing overrides of authorization or independent approval safeguards.
4. WHEN personal information is collected THE SYSTEM SHALL limit default fields to staff identity/assignments, buyer contact/transactions, supplier/contractor contacts and necessary evidence and record notice acknowledgment and explicit consent when required by the approved policy.
5. WHEN operational use is initialized THE SYSTEM SHALL require the owner to record operating countries, approved privacy notice and retention schedule in local operating documentation before entering live personal records, without asserting an unestablished jurisdictional compliance status.
6. WHEN an Administrator reviews an access, correction or deletion request THE SYSTEM SHALL retain an audit trail, honor legal holds and anonymize unnecessary contact details without destroying required business, financial or audit evidence.
7. WHEN retention review identifies eligible records THE SYSTEM SHALL apply provisional periods of seven years after closure for business/financial evidence, two years after last activity for unconverted leads, 90 days after deactivation for unnecessary staff profile details and 30 days for expired session/token hashes, subject to recorded holds.

## Requirement 25: Audit history

**User Story:** As an Administrator, I want searchable immutable audit evidence, so that important actions can be investigated without exposing secrets.

_Source: E24, N11, W1, W2, EV4, A7_

### Acceptance Criteria

1. WHEN Audit history opens THE SYSTEM SHALL show read-only searchable events with actor, UTC timestamp, action, project, record ID/link, request ID, reason and safe before/after details.
2. WHEN authentication, recovery, deactivation, permission changes, approvals, stock adjustments, booking changes, financial posting/reversal or important business mutations occur THE SYSTEM SHALL retain their audit evidence without passwords, reset tokens, session secrets or sensitive internal errors.
3. WHEN audit evidence is requested THE SYSTEM SHALL allow Administrator search, Finance access to relevant financial trails and Project Manager access to assigned project approval trails while preventing application users from editing entries.
4. WHEN audit retention is evaluated THE SYSTEM SHALL retain important mutation and approval evidence for seven years after parent closure and routine authentication events for one year, preserving source-record holds on retained exports.

## Requirement 26: API contracts, persistence and local operation

**User Story:** As an Administrator, I want a documented persistent local application, so that records survive restart and another operator can run it.

_Source: S1, S5, S6, S7, S8, S9, N5, F6, F8, E25, E26_

### Acceptance Criteria

1. THE SYSTEM SHALL implement Angular standalone routed components with strict TypeScript, typed HttpClient services and reactive forms, backed by Go HTTP handlers, business logic and SQLite data access with pinned supported dependencies and recorded tool/native-driver requirements.
2. THE SYSTEM SHALL provide a synchronized versioned OpenAPI contract under `/api/v1` before page-to-API integration, defining each endpoint's method, path, permissions, exact request/response fields, examples, lifecycle actions, errors, identifiers, nullability, timestamps and exact money representation.
3. THE SYSTEM SHALL provide the required API groups for authentication/users/roles/assignments; projects/sites/milestones/tasks/summary; site logs and review; inspections/issues; materials/suppliers/orders/amendments/receipts; inventory/movements; contractors/work orders; budgets/expenses/invoices/payments; properties/images; leads/activities/visits/customers/conversion; reservations/bookings/installments/cancellation/sale/handover; documents/settings/audit; dashboard and progress/budget/inventory/sales/collection reports.
4. WHEN API requests are handled THE SYSTEM SHALL validate them server-side, bound pagination and body size, allowlist sorting/filtering, use parameterized SQL, enforce timeouts/cancellation and return status-appropriate JSON errors with stable code, safe message, optional field errors and request ID, including JSON rather than frontend HTML for unknown API routes.
5. WHEN records are persisted THE SYSTEM SHALL use a configurable SQLite location with ordered versioned migrations, foreign keys on every connection, documented busy-timeout/connection/WAL behavior, lifecycle/uniqueness checks and indexes for project/date/status, source transactions, exclusive active unit ownership, document authorization and token/session lookup.
6. WHEN the backend restarts or the application is rebuilt THE SYSTEM SHALL preserve operational records and attachments without recreating or erasing user data, with development seed execution explicit and separate from normal startup.
7. WHEN a developer follows the local instructions THE SYSTEM SHALL support documented installation, migration, seed, development, build, test and built-application execution including Windows PowerShell instructions, an Angular development API proxy and Go-served same-origin production-build pages with deep-link refresh.
8. WHEN the application starts or stops THE SYSTEM SHALL provide sanitized configuration/storage failure messages, request-ID structured logging, graceful shutdown, lightweight liveness and database-aware readiness without exposing secrets, with environment-specific secrets outside source control.
9. WHEN an Administrator opens the local operator status view within Company settings THE SYSTEM SHALL report startup/readiness, database access, attachment-storage and backup failures with sanitized diagnostic references.

## Requirement 27: Capacity and response targets

**User Story:** As a Management Viewer, I want the application sized for planned operations, so that reporting and daily work complete within measured limits.

_Source: N7, N8, R8, R9, E27, E28_

### Capacity acceptance dataset

Year 1 / year 3 cumulative volumes: staff 25/75; simultaneous users 15/40; projects 10/30; project sites or blocks 30/90; tasks 5,000/15,000; site logs 10,000/30,000; inspections 2,000/6,000; issues 5,000/15,000; materials 2,000/6,000; suppliers and contractors combined 200/600; purchase orders 2,000/6,000; receipts 5,000/15,000; stock entries 50,000/150,000; work orders 1,000/3,000; budget lines 1,000/3,000; expenses 10,000/30,000; invoices 10,000/30,000; payments 15,000/45,000; units 500/1,500; leads 1,000/3,000; customers 500/1,500; site visits 2,000/6,000; reservations 1,000/3,000; bookings 500/1,500; documents 20,000/60,000; audit entries 500,000/1,500,000; attachment storage 100 GB/300 GB; active sessions at most 300/900; outstanding recovery tokens at most 100/300. Year-1 buyer/prospective-buyer contact planning is 1,000. These are sizing inputs, not seeded operational facts.

### Acceptance Criteria

1. WHEN exercised on a documented reference server with the year 1 and year 3 capacity acceptance datasets respectively THE SYSTEM SHALL support 25/75 staff, 15/40 simultaneous users, 10/30 cumulative projects, 30/90 project sites or blocks, 5,000/15,000 tasks, 10,000/30,000 site logs, 2,000/6,000 inspections, 5,000/15,000 issues, 2,000/6,000 materials, 200/600 suppliers and contractors combined, 2,000/6,000 purchase orders, 5,000/15,000 receipts, 50,000/150,000 stock entries, 1,000/3,000 work orders, 1,000/3,000 budget lines, 10,000/30,000 expenses, 10,000/30,000 invoices, 15,000/45,000 payments, 500/1,500 units, 1,000/3,000 leads, 500/1,500 customers, 2,000/6,000 site visits, 1,000/3,000 reservations, 500/1,500 bookings, 20,000/60,000 documents, 500,000/1,500,000 audit entries, 100 GB/300 GB of attachments, at most 300/900 active sessions and at most 100/300 outstanding recovery tokens, including 1,000 buyer/prospective-buyer contacts in year 1, without violating record integrity or authorization.
2. WHEN operating at the planned peak load THE SYSTEM SHALL complete at least 95% of sign-ins, filtered list loads and ordinary saves within 2 seconds, dashboards within 3 seconds and reports within 5 seconds, measuring upload transfer separately from server processing.
3. WHEN exporting up to 50,000 authorized report rows at planned peak load THE SYSTEM SHALL complete the export within 30 seconds.
4. WHEN capacity is measured THE SYSTEM SHALL record reference-server specifications, dataset sizes and observed response distributions for morning planning, end-of-day reporting and month-end reconciliation loads.

## Requirement 28: Backup, retention and restoration

**User Story:** As an Administrator, I want consistent backups and tested restoration, so that a failure does not lose more than an hour of work.

_Source: N1, N9, F7, A12_

### Acceptance Criteria

1. WHILE the application is in operational use THE SYSTEM SHALL create consistent database-and-attachment backups hourly to separate protected storage, retaining hourly copies for 48 hours and daily copies for 30 days.
2. WHEN an active SQLite backup is taken THE SYSTEM SHALL preserve a restorable database/attachment set using documented WAL handling without copying an inconsistent database state.
3. WHEN service is restored from the latest valid backup THE SYSTEM SHALL support a maximum 1-hour data-loss window and restoration within 4 hours using documented procedures.
4. WHEN restoration is exercised before the pilot and monthly thereafter THE SYSTEM SHALL provide disposable restoration checks that reconcile records and attachments without overwriting operational data.
5. WHEN routine maintenance is scheduled THE SYSTEM SHALL identify Sunday 02:00–04:00 in company time as the maintenance window and expose backup status and failures to the Administrator.

## Requirement 29: Connected acceptance and pilot evidence

**User Story:** As an Administrator, I want evidence from real journeys, so that the pilot starts with verified access, records and reconciled totals.

_Source: W1, W2, W3, W4, W5, W6, W7, W8, W9, M1, M2, M3, F3, F8, F9, F10, F12_

### Acceptance Criteria

1. WHEN acceptance fixtures are explicitly seeded into an isolated development database THE SYSTEM SHALL provide at least three projects at different stages with related sites, plans, materials, suppliers, contractors, partial deliveries, logs, inspections and issues, approximately twenty units across availability states, leads, bookings and partial collections whose totals reconcile, without overwriting operational data.
2. WHEN each of the seven roles signs in against the real backend THE SYSTEM SHALL produce its required landing page/navigation and permitted work while denying wrong-project data, unauthorized financial data, forged role values, unauthorized exports/downloads, inactive accounts and obsolete sessions.
3. WHEN the access/restoration journey is exercised THE SYSTEM SHALL demonstrate wrong-credential and rate-limit states, unsafe-return rejection, setup/change/recovery, expired/reused-token rejection, expiry/logout/deactivation and live permission changes through browser and direct-API evidence.
4. WHEN Project to site progress and Inspection to resolution are executed THE SYSTEM SHALL retain project/site creation, weighted planning, Site Engineer assignment, returned/resubmitted/approved logs, failed inspections, corrective evidence and reasoned reopening with reconciled progress and history.
5. WHEN Purchase to consumption and Contractor to payment are executed THE SYSTEM SHALL reconcile partial receipts, remaining quantities, issues, contractor work, expense/bill linkage and payment allocation while demonstrating insufficient-stock rejection, duplicate-request protection and single-counted project costs.
6. WHEN Enquiry to booking, Booking to collection and Sale confirmation and handover are executed THE SYSTEM SHALL preserve lead/visit/customer history, enforce concurrent acquisition and expiry rules, reconcile installments/allocations/cancellations/refunds and require the documented independent sale and handover evidence.
7. WHEN UI-created records are refreshed and the backend restarted THE SYSTEM SHALL preserve committed values and relationships, with isolated database checks demonstrating rollback, uniqueness, foreign keys, stale-update protection and concurrent unit ownership.
8. WHEN release acceptance is recorded THE SYSTEM SHALL provide a 36-screen route/role/API/action/evidence matrix, real-backend browser results, frontend type/build results, Go build/test results, API and database results, responsive desktop/mobile screenshots, actual font-loading and keyboard evidence, with executed commands/results and unrun checks distinguished.
9. WHEN the agreed 30-day pilot is evaluated THE SYSTEM SHALL provide evidence that at least one project's construction and property sales completed the required workflows, weekly report preparation time decreased by at least 50% from the pilot-start baseline, and acceptance found zero critical defects, lost records, duplicate bookings or unauthorized access with costs, stock and customer balances reconciled.

## Assumptions

- The root route is an entry resolver: unauthenticated visitors go to Sign in and authenticated staff go to their authorized role landing page.
- Weighted progress uses positive weights totaling 100 across the active plan; cancelling a weighted milestone requires explicit rebalancing, and approving a site log does not itself alter milestone completion.
- Budget variance means budget minus approved actual costs minus remaining approved commitments; cash is displayed separately.
- Inaccessible individual resources consistently return non-disclosing HTTP 404; role-forbidden operations return HTTP 403.
- Initial Site inventory shows quantities and units, not monetary stock valuation; no valuation policy was supplied.
- The local operator status view is a section within Company settings, not an additional baseline page.
