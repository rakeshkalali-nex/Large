# Secure sessions and server-determined permissions — issue 4

## Delivered boundary

- Real persisted account lookup and Argon2id verification (64 MiB, three passes,
  one lane); unknown accounts perform dummy verification. Login DTO accepts only
  email/password. Hash helper is reusable by the separately owned credential flows.
- Opaque 32-byte session tokens, SHA-256 storage, login rotation/revocation, atomic
  success/logout/expiry audit, generic failed-login audit, server role/assignment
  restoration and 30-minute idle / 12-hour absolute deadlines.
- Current account status, setup flag, role, assignments, access revision and user
  version are checked on protected operations. User version is the conservative
  credential revision: credential writers must bump it or revoke sessions in the
  same transaction. Assignment/role changes are read live; access management must
  additionally bump access_revision/revoke sessions atomically.
- Host-only HttpOnly SameSite=Lax session cookie and readable signed CSRF cookie;
  Secure with HTTPS. Exact Origin, signed nonce, header/cookie equality, expiry and
  session binding are checked. Login rotates CSRF; reset route has CSRF and token
  digest/origin throttling even though credential reset itself belongs to issue 5.
- Persistent keyed identifier/peer-address limits: five and thirty attempts per
  fifteen minutes. Successful and blocked attempts conservatively consume budget.
  Issuance helper supports five target / thirty issuer attempts per hour. See the
  rate-limit package README for the no-nested-writer integration requirement.

## Integration contracts

`GET /api/v1/auth/me` returns the existing Identity shape, including setupRequired,
empty arrays rather than null, deadlines and CSRF. It never extends idle expiry.
The frontend must await it before protected navigation/data, clear transient state
on 401, display SESSION_EXPIRED guidance and never replay stock/financial writes.
Screens and browser no-flash verification are owned by the frontend tasks, not
claimed as tested by these server tests.

The global gate enforces registered route capabilities, rejects unknown actions,
and constructs principals from persisted sessions, not browser role headers.
Empty project ID is only a coarse route capability check, never permission for an
unfiltered query. Domain handlers must use policy Scope/ScopeTx predicates before
pagination/counting, the supplied least-data columns, trusted persisted project
IDs and RequireResourceTx for non-disclosing detail lookup. They must enforce
independent approval and parent-document policy within their owning workflows.
Unimplemented domain routes still do not return business records.

Use sessions.Service.WithinAuthorizedWrite for mutation callbacks: it rechecks
the session and current policy inside the same serialized write transaction and
rolls back idle refresh along with the business mutation on failure. For successful
explicit read interactions, the UI may send X-User-Interaction: true; the common
gate then refreshes idle expiry without moving absolute expiry. Background polls
must omit this marker. auth/me never touches idle even with that marker.

The server creates a 32-byte signing key under DATA_DIR at first startup using
exclusive creation and private permissions. Windows operators must restrict the
data directory ACL to the service/operator account; POSIX mode bits alone are not
a Windows ACL. Preserve this key across restarts to keep throttle subjects stable.
Malformed existing keys fail startup; no secret is committed or logged.

## Necessary composition edits outside the expected directories

- backend/internal/contracts/dependencies.go: inject signing key and secure-cookie setting.
- backend/internal/httpapi/router.go: apply the session gate to mounted API handlers.
- backend/cmd/server/main.go: initialize the private persistent key and inject policy/configuration.

The existing auth route slots remain as fallbacks; method-specific session routes
take precedence. This preserves the separately owned password/recovery mounts.

## Regression coverage

Session tests use a migrated disposable SQLite database, randomly generated
credentials, real Argon2id checks and an injected clock. They cover token storage,
rotation, restoration, logout, deadlines, current/obsolete revisions, setup-only
identities, HTTP cookies and CSRF, forged claims, and transactional rollback.
Policy tests cover all seven roles, closed route registry, no-project assignments,
scoped SQL projections and ownership. CSRF tests cover signature/origin/binding
failures and TLS. Rate-limit tests cover persistence, independent writer connections,
both budgets and transaction rollback. No test counts unrelated table rows or
asserts that another feature's route is absent.

## Validation

Run from the repository root:

```
npm run test:backend -- ./internal/access/sessions/...
npm run test:backend -- ./internal/access/policy/...
npm run test:backend -- ./internal/access/csrf/...
npm run test:backend -- ./internal/access/ratelimit/...
```

Results are recorded in validation.md after execution. No public registration,
demo login bypass, browser-authoritative storage or paid service was introduced.
