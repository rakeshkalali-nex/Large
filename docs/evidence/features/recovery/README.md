# Password lifecycle evidence — task 5

Scope: requirement 3.1–3.8 backend lifecycle and local bootstrap, not the later
Sign in presentation, staff creation UI, or optional generated property campaign.

## Reproduction

From the repository root, with the pinned Go and Node tools available:

```text
npm run install:backend
npm run test:backend -- ./internal/access/passwords/...
npm run test:backend -- ./internal/access/recovery/...
npm run test:backend -- ./internal/access/bootstrap/...
npm run test:backend -- ./cmd/manage
```

Tests must use disposable SQLite databases and runtime-generated credentials.
They must not depend on other feature routes being absent or count unrelated
business rows in a shared database. No operator credential or reset token belongs
in committed evidence, request fixtures, logs, or screenshots.

## Security contract

- First Administrator: private operator input, no public initialization,
  Argon2id storage, and rejection after any staff account exists.
- Setup/recovery: 24-hour/30-minute digest-only token lifetimes, one-time use,
  same-purpose replacement, uniform invalid-token rejection, and fresh login.
- Atomic completion: hash update, token invalidation, session revocation,
  access revision and secret-free audit commit together or not at all.
- Authorization: setup-only access, current-password proof for authenticated
  changes, Administrator-only verified recovery issuance, same-origin CSRF.
- Recovery guidance exposes only intentionally public company instructions.

## Boundaries

The operator procedure is documented in `docs/operations/accounts.md`; the
area-owned HTTP contract is `docs/api/passwords.yaml`. Human identity verification
and confidential delivery are operator responsibilities, not claims established
by automated API tests. Browser fragment removal, screen accessibility and
physical device coverage need the owning frontend/connected acceptance task.

## Saved browser-origin recovery journey

`e2e/password-recovery.spec.ts` is the task-specific Playwright journey. It uses
the real browser origin to retrieve trusted-channel guidance, obtain CSRF,
reject a generated unknown reset token with replacement instructions, and check
that the unauthenticated browser remains unauthenticated. It does not claim to
test the password forms owned by Issue #8 or the complete credential lifecycle.
The foundation visual test is unrelated evidence, not this task's journey.

Run from the repository root after installing project dependencies:

```text
npm run build
node node_modules/@playwright/test/cli.js test --config docs/evidence/features/recovery/playwright.config.ts
```

The configuration reuses the integration harness to allocate a free port and
start/stop the built application with disposable SQLite storage. It runs desktop
and touch/mobile Chromium profiles without visual baselines or mocked APIs.
An installed Playwright Chromium binary is required; browser installation is an
explicit operator step, not performed by this configuration.
