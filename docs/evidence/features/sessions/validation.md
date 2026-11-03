# Validation — 2026-11-03

All required commands ran from the dedicated worktree repository root and exited 0:

| Command | Result |
| --- | --- |
| npm run test:backend -- ./internal/access/sessions/... | PASS, 11.980s |
| npm run test:backend -- ./internal/access/policy/... | PASS, 6.646s |
| npm run test:backend -- ./internal/access/csrf/... | PASS, cached from earlier successful run |
| npm run test:backend -- ./internal/access/ratelimit/... | PASS, 3.661s |

Prerequisite `npm run install:backend` exited 0. Additional composition check
`go test ./internal/httpapi/... ./cmd/server/...` ran from backend and exited 0:
HTTP setup tests passed (1.403s), server package compiled (no test files).

The final run includes fixture-scoped SQL assertions/mutations and preservation of
401/503 when an individual-resource policy recheck fails for reasons other than
authorization. Tests use their own migrated temporary databases and do not use
operator data or fixed login credentials.

Not run: frontend/browser journeys, full application build, load benchmarking,
full repository suite, and OpenAPI fragment validation. Frontend no-flash/expiry
screens and domain workflow integration remain with their owning tasks. The
authentication handler integration tests exercise the real HTTP composition root.

No commits, pushes, pull requests, specification edits or system-service changes.

## Authentication error-contract correction — 2026-11-03

Corrected the missing-session response from `AUTHENTICATION_REQUIRED` to the
approved `AUTH_REQUIRED` code and aligned `docs/api/auth.yaml`. The HTTP regression
test exercises the real composition root for unauthenticated `GET /api/v1/auth/me`
and `GET /api/v1/projects`, asserting HTTP 401, the canonical error code, a nonempty
safe message and request ID, and JSON content type. The post-logout restoration
assertion now checks the error envelope rather than only HTTP status.

After `npm run install:backend` exited 0, all four required commands ran exactly
from the repository root and exited 0:

| Command | Result |
| --- | --- |
| npm run test:backend -- ./internal/access/sessions/... | PASS, 8.778s |
| npm run test:backend -- ./internal/access/policy/... | PASS, 3.718s |
| npm run test:backend -- ./internal/access/csrf/... | PASS, 0.757s |
| npm run test:backend -- ./internal/access/ratelimit/... | PASS, 1.697s |

This correction changes only the session service, its HTTP tests, the auth API
fragment and this evidence file. Earlier implementation and integration wiring
are preserved. No frontend/browser journey, live-server request, application
build or full repository suite was run during this correction.
