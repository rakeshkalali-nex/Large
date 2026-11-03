# Sign-in smoke suite — task 6 / issue #8

Run from the repository root with `npm run test:integration -- auth-entry`.
The existing `tests/integration/playwright.config.ts` discovers `**/*.spec.ts`,
so no discovery/config change is required. The runner builds the Angular app and
Go `server` and `manage` binaries before launching Playwright.

Each test independently bootstraps a real Administrator using the actual Go
operator binary, protected stdin, a random email/password and disposable SQLite.
It starts the actual Go server on a free loopback port and overrides Playwright's
base URL for that test. All runtime persistence is under the OS temporary folder;
credentials are memory-only, never command arguments or repository files. The
fixture terminates its own server and removes its temporary directory on teardown.
Login traces, screenshots and videos are disabled to avoid persisting credentials,
session cookies or request bodies. No install is performed by these fixtures.

## Lead wiring / execution notes

- No wiring is required for discovery. The parent config also starts its ordinary
  empty-database server; this suite deliberately uses independent per-test servers
  to avoid sharing credentials, rate limits, sessions or mutation state.
- The existing `scripts/run.mjs` integration command unconditionally invokes
  `playwright install chromium`. If the no-installs policy must hold at execution,
  the lead must remove/gate that step or run the already-built suite directly with
  `node node_modules/@playwright/test/cli.js test --config tests/integration/playwright.config.ts auth-entry`.
  Browser binaries must already be installed; these tests do not install them.
- Authentication UI contract: `/login`, `Email` (or `Email address`), `Password`,
  `Sign in`, heading `Welcome back`, an accessible recovery action, `state=recovery`,
  and `returnUrl`. Recovery may be a button that changes local account-shell state.
- Assertions check authorized landing URLs, not contents of future feature pages
  and never an expected 404/placeholder. Administrator login/root restore and an
  authorized `/projects` return are exercised. Other role fixtures are intentionally
  not synthesized through raw SQL or future user-management APIs; their browser
  landings remain outside this smoke suite until a supported fixture path exists.
- Lifecycle coverage includes anonymous query-state entry, recovery guidance,
  successful authenticated password change followed by sign-in with a new runtime
  random password, and malformed/unknown reset fragment scrubbing. Unknown tokens
  exercise the unavailable-link state shared with expired/used/replaced tokens;
  no genuinely expired token is seeded. Successful setup/reset consumption remains
  outside this suite.
- The network-error and duplicate-submit cases inject a transport abort/delay only;
  they never fabricate a successful authentication response.
- Viewport shrink is a keyboard-layout approximation, not proof using a real mobile
  software keyboard. No screenshot baselines are created or updated.

Commands and browser tests were not run during authoring, as requested. The lead
must execute the suite after the concurrent UI/session implementation is ready.
