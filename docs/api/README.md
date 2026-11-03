# Versioned API contract

`docs/api/contract.mjs` is the compact explicit authoring source for every designed API table. `docs/openapi.yaml` is its composed OpenAPI 3.1.0 document, serialized as JSON (valid YAML 1.2). Contract version is 1.0.0; all paths are relative to `/api/v1`. Edit the authoring source, not the generated document or TypeScript file.

## Generation and validation

The foundation manifest owner installs **pinned** `openapi-typescript` and `@apidevtools/swagger-parser` root development dependencies and owns the lockfile. This area does not install dependencies or modify manifests. After installation, run from the repository root:

```text
node scripts/contracts.mjs generate
node scripts/contracts.mjs check
node scripts/contracts.mjs validate
```

The root `test:contracts` script should run `node scripts/contracts.mjs check`. Generation validates using Swagger Parser, generates TypeScript with openapi-typescript's AST printer, sorts generated declarations, normalizes newlines and writes no timestamps. Check regenerates **in memory**, compares both checked-in outputs byte-for-byte, and fails for drift, unresolved references, duplicate operation IDs, missing policy/status metadata, missing mutation headers or missing explicitly designed paths. Validate performs validation without comparing/writing artifacts. Neither command installs a package or contacts a remote API. The generated DTO entry point is `frontend/src/app/core/contracts/generated/api.ts` and exposes `paths`, `operations` and `components` for typed HttpClient adapters.

## Interpretation

- `x-implementation-status: unavailable` means a designed business endpoint is **not delivered** by the foundation. Its success schemas document the target, not fabricated data. Initial routes must deny access or return a safe JSON unavailable response. Health has `foundation` status and must be verified by platform tests separately.
- Common JSON envelopes include request IDs; mutation envelopes additionally carry committed cache-invalidation effects. Logout is 204 without content; health is a small direct object; downloads and CSV use their declared media types.
- Resource metadata and server-owned lifecycle/calculation fields never appear in editable inputs. Updates use full editable-field replacement and a version. Nullable request fields may be omitted or null; nullable response fields are required and explicit null. Arrays are never null. Initial project sites and initial asking price are create-only.
- UUID identifiers, UTC RFC3339 timestamps and date-only strings are distinct schemas. Money, quantities, prices and percentages are exact decimal strings. Money is minor units; quantities have at most three decimals, unit prices four and percent two (basis-point storage). Percentages must not be confused with integer `percentageBps` cancellation inputs. Financial persistence is checked signed int64; round line amounts half-up before summation. Currency is configured, never invented in examples.
- Every write has Idempotency-Key and X-CSRF-Token headers. Same-origin signed double-submit CSRF and current cookie-session checks are required. Requests are limited to 1 MiB except bounded multipart uploads. No automatic mutation retries.
- Permission policy is documented per operation and in the policy matrix extension. Capability metadata guides route integration, not client authority. Implementers must enforce current role/project/assignment, related IDs, field projection, independent approval and lifecycle predicates inside transactions. Forbidden actions are 403, inaccessible individual IDs indistinguishable 404. An Administrator cannot self-approve.
- Named least-data projections include work-order site evidence, property construction data, procurement invoice evidence, project selectors, assigned customer sales data and Project Manager financial summaries. The server selects the projection before serialization. Report views use named row types, never arbitrary JSON rows. Report-specific column/sort permissions must be enforced alongside the report/view allowlist.
- Settings/reference/checklist, review/history and report support structures refine fields the design names without expanding editable resource inputs. Lifecycle enums and action routes are separate; no generic delete route exists.
- All operation examples are synthetic structural examples, not credentials or business fixtures. They demonstrate wire shape, not that related IDs satisfy a live transaction's rules.

## Contract handoff

Feature owners implement their reserved methods and consume the generated types. They add handler request/response and authorization tests before changing their availability status; schema validation alone does not verify a handler, authorization predicate, business lifecycle or data persistence. Shared contract composition has one owner. Deliberate breaking changes require a new API compatibility version; additive schema changes still require regeneration and consumer checks.
