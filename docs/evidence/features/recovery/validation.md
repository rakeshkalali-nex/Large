# Executed validation — 2026-11-03

Environment: Windows 11, dedicated task worktree, repository-root npm wrappers.
`npm run install:backend` completed successfully without dependency changes.

The implementation workers executed all required commands successfully:

| Command | Result |
| --- | --- |
| `npm run test:backend -- ./internal/access/passwords/...` | PASS, 83.480s |
| `npm run test:backend -- ./internal/access/recovery/...` | PASS, 20.908s |
| `npm run test:backend -- ./internal/access/bootstrap/...` | PASS, 14.828s |
| `npm run test:backend -- ./cmd/manage` | PASS, 3.463s |

The lead also invoked the same four commands sequentially from the repository
root after integration. Passwords and recovery returned cached passes; bootstrap
passed in 29.520s and manage passed in 10.399s; the chained command exited 0.
No credential-bearing requests or scripts were written into
the application tree.

Scope extensions are minimal integration changes: management command dispatch,
access route registration, the recovery-issuance capability route, and the
shared no-referrer response policy required by the password-link design.

## Retry validation — saved recovery evidence

The reported foundation visual test is a Playwright test, but does not exercise
password recovery. Added `e2e/password-recovery.spec.ts` rather than changing the
foundation baseline or implementing the password UI owned by Issue #8. This is
the only new scope extension in this retry: a saved test must live under `e2e/`.
Its self-starting configuration and reproduction instructions live in this
feature's evidence directory and reuse the existing integration server harness.

Executed from the repository root on 2026-11-03:

| Command | Result |
| --- | --- |
| `npm run install:backend` | PASS |
| `npm run test:backend -- ./internal/access/passwords/...` | PASS, 22.510s |
| `npm run test:backend -- ./internal/access/recovery/...` | PASS, 6.181s |
| `npm run test:backend -- ./internal/access/bootstrap/...` | PASS, 10.244s |
| `npm run test:backend -- ./cmd/manage` | PASS, 4.462s |
| `npm ci` | PASS; reported 23 dependency vulnerabilities (20 moderate, 1 high, 2 critical); no dependency upgrades made |
| `npm run build` | PASS after installing missing workspace dependencies |
| `node node_modules/@playwright/test/cli.js test --config docs/evidence/features/recovery/playwright.config.ts` | PASS, 2 projects (desktop and phone Chromium), 40.5s |

The initial build failed because the fresh worktree lacked the Angular compiler
dependency; installing the locked workspace dependencies resolved that failure.
The browser test checks real HTTP guidance, CSRF-backed unknown-token rejection,
replacement instructions and continued unauthenticated status. No mocks, seeded
credentials, visual baseline changes or assumptions about missing feature routes.

Not run in this retry: Firefox/WebKit, physical-device checks, the full
application suite, and the optional generated property campaign. Human identity
verification and private delivery remain operational procedures, not automated
test claims. This API journey does not establish password-form UI coverage.

## Cleaned implementation continuation — 2026-11-03

Continued the preserved implementation from commit `56b5006` on top of
`104b3df` (generated Angular compiler output ignore fix). Before this evidence
update, `git diff 56b5006 --stat` was empty. No runtime code changes were needed
for the reported generated-output merge obstruction.

`git ls-files frontend/out-tsc` returned no tracked files, and
`git check-ignore frontend/out-tsc/app/app/core/state/project-context.js`
confirmed the generated path is ignored. No generated output was added or
staged, and no merge, commit, push, or branch change was performed.

Installed backend dependencies with `npm run install:backend`, then executed
the required commands exactly from the repository root:

| Command | Result |
| --- | --- |
| `npm run test:backend -- ./internal/access/passwords/...` | PASS, 21.575s |
| `npm run test:backend -- ./internal/access/recovery/...` | PASS, 11.170s |
| `npm run test:backend -- ./internal/access/bootstrap/...` | PASS, 8.484s |
| `npm run test:backend -- ./cmd/manage` | PASS, 7.448s |

The installation and validation command chain exited 0. The existing command
dispatch, access route registration, capability mapping, and no-referrer policy
changes remain necessary integration wiring. This continuation changes only
this evidence document; specification files remain untouched. Browser tests,
frontend builds, and the full application suite were not rerun in this
continuation; results above supersede only the four backend validation results.
