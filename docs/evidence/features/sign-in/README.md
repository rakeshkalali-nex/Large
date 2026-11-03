# Sign in and authorized entry — issue #8

## Delivered boundary

- `frontend/src/app/features/auth/entry/` owns the branded account page and local styles. Desktop uses the existing local architectural illustration, navy brand panel and warm-light 420px form. At tablet/mobile sizes a compact brand header replaces the illustration; natural scrolling and dynamic viewport height keep controls reachable.
- `frontend/src/app/core/session/` restores the existing server identity store, calls the real cookie-based Go endpoints, obtains fresh CSRF protection before mutations and never automatically retries credential requests. No credentials or authoritative business data are persisted in browser storage.
- `frontend/src/app/core/guards/` delays protected navigation until restoration, checks closed return-route shapes and server capabilities, and uses the exact seven-role landing baseline. Go remains the record/project authorization boundary.
- Setup, reset, password change, recovery, expiry, service failure and completion guidance share the account shell. Fragment tokens are scrubbed before restoration requests, held transiently and never rendered into the DOM. Passwords are cleared after attempts; email is retained.

## Necessary composition changes

Three files outside the feature-owned directories provide required integration:

- `frontend/src/app/app.routes.ts`: remove the unconditional root redirect, mount auth entry resolution, and protect domain routes and access denial.
- `frontend/src/app/app.config.ts`: provide the real logout action and session-expiry navigation interceptor.
- `frontend/src/app/app.ts`: keep the authenticated application shell off account pages and withhold authenticated navigation until identity is restored.

No specification, backend, dependency, or public-registration code changed.

## Acceptance coverage

| Criteria | Evidence |
| --- | --- |
| 1.1 | Session single-flight restoration unit tests, protected/root guard tests, real login and root-refresh browser case |
| 1.2–1.3 | Screen-owned responsive styles; browser cases at 1440, 1024, 768 and 390px, desktop form width, compact header and mobile keyboard-like viewport shrink |
| 1.4 | Auth form unit assertions for labels, masking, toggle, autocomplete and preserved password; real browser Enter submission and keyboard-operated visibility toggle |
| 1.5 | Real Go/SQLite bootstrap fixture, CSRF transport test, pending and duplicate-submit unit/browser cases |
| 1.6–1.7 | Generic credential error and separate connectivity failure, retained email and cleared password unit/browser cases; no automatic replay |
| 1.8 | Exact seven-role landing unit cases, closed route/capability checks, unsafe return and authenticated-login browser cases |
| 1.9 | Lifecycle state unit/browser cases, real password-change and unknown-token reset browser cases |

## Reproduction and limitations

Run from the repository root:

```text
npm test --workspace frontend -- auth
npm test --workspace frontend -- session
npm test --workspace frontend -- guards
npm run test:integration -- auth-entry
```

The existing integration command builds Angular and Go and starts its own processes. The auth fixture additionally uses per-test OS-temp databases and runtime-generated bootstrap credentials supplied through private standard input. Each test owns its database, so parallel rate limits, sessions and password changes do not interfere. Credential-bearing traces, videos and screenshots are disabled.

The smoke browser is Chromium. Viewport shrinking approximates a keyboard; real iOS/Android keyboards and other browser engines have not been exercised here. Successful initial setup and an actually expired token are covered by existing backend ownership rather than newly seeded browser fixtures. Other-role destinations are unit-tested; the real browser account is an Administrator. No downstream business screen is claimed as delivered by this task.
