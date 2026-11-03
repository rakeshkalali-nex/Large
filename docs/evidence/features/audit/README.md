# Audit history — issue 11 / task 26

Screen `/audit` uses `GET /api/v1/audit-events` and `GET /api/v1/audit-events/{id}`. The implemented feature contract is `docs/api/audit.yaml`; the broad foundation-generated contract remains a design inventory, not proof of implemented responses.

## Delivered boundaries

- Read-only search, actor/action/project/record/UTC-date filters, bounded keyset pagination, UTC event timestamps, safe field changes, request references and reason details.
- Administrator searches company evidence. Finance sees an explicit allowlist of financial/source trails in assigned projects. Project Managers see submission/review decisions for operational/approval sources in assigned projects; exact amount changes are withheld.
- Session authorization, current role and assignments are checked inside the read transaction. Pagination binds principal/session/access revision, role, filters, direction and limit. Lists filter before pagination; inaccessible detail IDs return the same not-found response as missing IDs.
- Before/after changes use a closed field/value projection. Unsafe retained reasons are suppressed. No raw JSON, origin digests, credentials, internal errors, or write controls are exposed.
- Business evidence is retained for at least seven years after completed/archived parent closure. Open parents and non-project evidence without a closure clock remain retained. Routine authentication has a one-year clock; password/setup events are important evidence even when legacy metadata says authentication. Active holds prevent eligibility.
- Event/source/project holds and persisted retained snapshot source identities are followed. Snapshot sources must record resource/id or recordType/recordId identities; report filter strings alone are not evidence of source membership. Export creation remains the reporting task's responsibility; this feature adds no export or purge endpoint.
- Desktop rows adapt to mobile cards. The details modal has labeled before/after fields, deliberate retry, native keyboard behavior and focus restoration. Session/project changes cancel pending reads and clear displayed evidence.

## Integration exception

`backend/internal/administration/audit/route_slot.go` is the only changed file outside the expected task folders. It delegates the already-mounted audit slot to the implemented history handler. The existing frontend audit route slot is inside the expected scope. No specification, migration, dependency manifest or lockfile was changed.

## Validation (2026-11-03 UTC, Windows 11)

- `npm ci`: exit 0. Existing dependency audit reported 23 vulnerabilities (20 moderate, 1 high, 2 critical); no dependency upgrades made.
- `npm run test:backend -- ./internal/audit/history/...`: exit 0, package passed (10.241 seconds). An earlier run began before retention implementation was present and failed with undefined retention symbols; the completed package passed.
- `npm test --workspace frontend -- administration/audit`: exit 0; one file, eight tests passed. Duration 168.43 seconds; test execution 1.38 seconds.
- Backend tests exercise real migrated disposable SQLite stores for scoped reads/details, role/session denial, cursor/filter boundaries, redaction, immutable API/database evidence, closure clocks, routine versus important authentication, and retained export holds. They do not depend on another feature route being absent.
- Frontend tests exercise the routed component through HTTP test requests, filters, pagination/retry, denied feedback, modal details and safe rendering, scope invalidation, and encoded detail IDs. Modal native behavior is stubbed by jsdom, not browser-verified.

Not run: real-backend browser journeys, screenshots at required widths, native dialog keyboard checks, full application build/full suite, capacity campaigns, and later cross-feature property tests. No claim of browser, capacity, pilot or complete report-export acceptance is made.
