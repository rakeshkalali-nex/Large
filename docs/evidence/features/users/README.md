# Users and access — task 7

## Wave 16 merge repair (2026-11-03 UTC)

- Resolved the conflict with issue #8 in `frontend/src/app/app.config.ts`,
  retaining both the Users CSRF cookie/header configuration and the sign-in
  feature's session-expiry interceptor and shell logout provider.
- Preserved the earlier Users implementation and its necessary backend route
  mounting/policy integration; no specification or dependency manifest changes.
- `npm ci` — exit 0; reported 23 vulnerabilities (20 moderate, one high, two
  critical). Dependency remediation remains outside this focused repair.
- `npm run test:backend -- ./internal/access/users/...` — exit 0 (105.666s).
- `npm test --workspace frontend -- administration/users` — exit 0;
  24 tests passed across two files, including real application-provider CSRF tests.
- `git diff --check HEAD` — exit 0 after resolving the conflict.
- Full build, browser/device journeys and accessibility checks were not rerun
  during this merge repair.

## Wave 15 corrections

- Configured Angular's built-in XSRF support in `frontend/src/app/app.config.ts`
  to read `buildestate_csrf` and send `X-CSRF-Token`. The server's signature,
  session binding and same-origin checks remain unchanged. Reading the current
  cookie for each mutation also respects token rotation across tabs/restoration.
- Users API tests now use the real application providers and assert the header
  on create, edit, deactivation/reactivation and setup/recovery issuance, token
  rotation, and omission on reads and requests to another origin.
- The destructive button now explicitly uses white text on red, with darker
  hover/pressed colors, leaving disabled styling to the shared theme. Calculated
  WCAG text contrast is 6.47:1 normally, 8.31:1 on hover and 10.02:1 pressed.
  These calculations are not a full browser accessibility audit.

The earlier implementation below is retained; no backend security checks were
relaxed and no specification files were edited.

Wave 15 validation from the repository root (2026-11-03 UTC):

- `npm ci` — exit 0.
- `npm run test:backend -- ./internal/access/users/...` — exit 0 (91.986s).
- `npm test --workspace frontend -- administration/users` — exit 0,
  24 tests across two files. An initial invocation preceded dependency install
  completion; the next exposed the test platform's empty origin. Configuring
  the mock platform location with the browser URL allowed the real Angular
  same-origin interceptor to run; no security behavior was mocked away.
- Browser journeys, screenshots, full accessibility audit and full build were
  not rerun during this focused correction.

Implemented the Administrator-only Users screen in its existing route slot and
SQLite-backed staff create/edit/list, role/project selectors and reasoned
deactivation/reactivation operations. Setup and recovery reuse the existing
password-reset endpoint with explicit identity verification and private delivery.
No automatic account activation, public registration or login bypass is provided.

Account identities and business references are retained. Lifecycle/role changes
and project assignment/unassignment are recorded through the shared transactional
audit writer. Access updates revoke affected sessions only; deactivation revokes
outstanding links and clears the password. Reactivation returns setup-required,
so the Administrator must explicitly issue a fresh setup link. Final-active-Admin
checks run inside the same serialized write as the mutation. Version checks and
persistent request digests prevent stale writes and repeated effects.

## Earlier implementation validation (2026-11-03)

Run from the repository root:

- `npm run test:backend -- ./internal/access/users/...` — passed, exit 0.
- `npm test --workspace frontend -- administration/users` — passed, exit 0;
  23 tests across two files.

Backend regression coverage includes identity/reference validation, duplicate
email normalization, nonadministrator denial, target-only revocation, retained
attribution and lifecycle audit, replacement setup, final-Administrator rejection
and competing administrator changes. Each fixture uses an isolated real migrated
SQLite database and assertions scope business rows to the fixture's identities.
Frontend coverage exercises the API contracts and rendered component actions.

Browser/device acceptance and a full application build were not run in this task.
`npm ci` reported 23 dependency vulnerabilities (20 moderate, one high, two
critical); dependency remediation is not part of this feature change.

## Necessary integration outside the expected directories

- `backend/internal/administration/users/route_slot.go` delegates the existing
  reserved slot to the implemented access/users module.
- `backend/internal/access/policy/routes.go` explicitly recognizes the new project
  selector and lifecycle action paths; otherwise the deny-by-default shared gate
  would reject those operations before their Administrator checks.

Specifications, shared manifests and migrations were not changed. No commits,
pushes, branches, pull requests or machine configuration changes were made.
