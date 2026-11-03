# Persistent authentication rate limits

`Limiter{DB, Key, Clock}` exposes
`Attempt(ctx, operation, identifier, origin) (time.Duration, error)`.
Supply the writable platform SQLite pool, an injected clock, and at least 32
private random key bytes, stable across restarts and processes. Do not mutate
these dependencies concurrently. Key rotation starts fresh subject budgets.

Supported operation constants:

| Constant / value | Identifier | Origin | Window / limits |
| --- | --- | --- | --- |
| Login / `login` | email | trusted peer IP without port | 15 minutes; 5 / 30 |
| Reset / `reset` | token digest, not raw token | trusted peer IP without port | 15 minutes; 5 / 30 |
| Issuance / `issuance` | target ID | authenticated issuing Administrator ID | 1 hour; 5 / 30 |

Identifiers are trimmed and lowercased; pass canonical digests/IDs consistently.
Origins are trimmed and otherwise caller-canonicalized. Never derive trusted
origins from arbitrary forwarded headers. Empty values share an empty-subject
bucket rather than bypassing a limit. Unknown operations return ErrOperation.

Attempt runs before credential verification and **conservatively counts every
attempt, including success and already-throttled requests**, against both
budgets. There is no success refund API. This intentionally tightens the design's
failed-attempt baseline. Counters saturate at limit plus one once blocked.

Each subject has a fixed window beginning with its first attempt, not an aligned
wall-clock window or sliding window. A SQLite transaction takes the writer lock
before reading counters and atomically commits both charges. Subject values are
domain-separated HMAC-SHA256 digests; no email, IP, password, or token is stored.
Expired rows for the current subjects are pruned transactionally. Dormant expired
rows require a separately owned maintenance retention job.

A positive duration requires denial with HTTP 429 and Retry-After rounded **up**
to seconds. Zero with no error permits proceeding; any error must fail closed.
Call outside another transaction on the same single-connection writer pool to
avoid waiting on a connection the caller already holds. This package does not
write HTTP responses or audit events.

Tests use isolated fully migrated temporary SQLite databases, injected clocks,
and independent writer connections for contention. Test execution belongs to
the parent; no tests were run during implementation.
