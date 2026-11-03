# Settings validation — 2026-11-03

Environment: Windows 11, repository-root commands in cmd.exe, Node 22.23.1 / npm 10.9.8.

## Executed

### Readiness correction (criterion 26.9)

- `npm run test:backend -- ./internal/settings/...` — exit 0, 25.222 seconds. Regression cases cover healthy storage, closed/missing database, missing/unconfigured attachment storage, sanitized diagnostics, and a provider withholding startup readiness despite healthy storage.
- `npm run test:backend -- ./internal/httpapi/...` — exit 0, 1.679 seconds.
- `npm ci` — exit 0; 565 packages installed. npm reported 23 dependency vulnerabilities (20 moderate, one high, two critical); no dependency upgrades were made.
- `npm test --workspace frontend -- administration/settings` — exit 0; 12 tests passed, 61.62 seconds.

The readiness handler in `backend/internal/httpapi/router.go` required a small integration change outside the settings directory: it now consumes the same operator diagnostics provider as Company settings rather than independently recomputing readiness. The default provider reuses the platform attachment probe and derives readiness from live database and attachment results. Existing authorization and sanitized diagnostic references remain intact.

### Earlier implementation validation

- `npm run test:backend -- ./internal/settings/...` — exit 0; parent run passed in 9.932 seconds; final backend-worker run after the personal-data gate and UUID validation passed in 10.795 seconds, including persistent duplicate-request coverage.
- `npm run test:backend -- ./internal/httpapi/... ./internal/administration/... ./internal/access/policy/... ./internal/platform/posting/...` — exit 0; route composition, access policy and posting tests passed; administration packages compiled and had no test files.
- `npm ci` — exit 0; installed the pinned workspace dependencies. npm reported 23 dependency vulnerabilities (20 moderate, one high, two critical); no automatic dependency upgrades were made as part of this feature.
- `npm test --workspace frontend -- administration/settings` — exit 0; 12 tests passed. Final run: 14.63 seconds, tests 2.51 seconds.
- `npm run typecheck --workspace frontend` — exit 0 after the final frontend changes.
- OpenAPI validation using the installed `@apidevtools/swagger-parser` against `docs/api/settings-stewardship.yaml` — exit 0, `Settings OpenAPI valid`.

The initial frontend command ran before dependency installation finished and could not find Vitest. Subsequent test failures were traced to asynchronous test setup and a DebugNode query that included non-elements; tests now wait for the actual completed load/request and query the component directive. Strict compilation also caught an incompatible checklist-array spread into the form; editable scalar fields are now mapped explicitly. These failures were fixed, not suppressed.

## Not claimed by these checks

No real-backend browser journey, responsive screenshot campaign, manual accessibility audit, live personal-record entry, production backup/restore drill or capacity measurement was run here. The focused Angular tests use test doubles. The actual backup implementation is task 28, and property-test campaigns #32/#15 remain separate tasks.

The transaction-scoped personal-data prerequisite helper is implemented and used by acknowledgment recording. Future staff, lead, customer, supplier and contractor creation services must invoke it before live personal-record entry; this task does not establish complete cross-domain entry enforcement. Operator status and `/api/v1/health/ready` now use the same diagnostics provider. The default provider checks live database access and attachment write/sync/remove readiness; a supplied provider can additionally gate startup work. Backup reports not-configured rather than implying successful protection. Reservation startup-expiry processing and backup implementation are not supplied by this fix.

## Necessary integration outside the expected feature directories

Backend route composition and the closed authorization route/capability registry must mount the new endpoints; the settings placeholder is no longer mounted. The dependency/operator contract and HTTP composition inject sanitized local diagnostic probes. The posting module exposes its existing currency validator for reuse rather than maintaining a second currency registry. A versioned migration adds immutable settings-history storage. These changes are required for a routable, persistent feature, not unrelated domain work.

The spec files were not edited. No commit, push, pull request, system service change or system package installation was performed.
