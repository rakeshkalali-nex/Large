# BuildEstate foundation

Internal construction operations and property sales, using Angular standalone components, Go HTTP and persistent SQLite. This setup task provides infrastructure and stable feature slots, **not completed business workflows**. Unimplemented pages explain that they are unavailable; API slots return safe JSON 503 responses, never mock records or an authentication bypass.

## Prerequisites

Verified local tool versions: **Node 22.23.1, npm 10.9.8, Go 1.26.10** on Windows 11 x64. Angular packages are pinned in `frontend/package.json`; direct Node packages and transitive resolution are pinned by `package-lock.json`. Go dependencies and checksums are in `backend/go.mod` and `backend/go.sum`. The SQLite driver is `modernc.org/sqlite v1.61.1`, documenting SQLite **3.54.1** for Windows amd64, and is pure Go: no C compiler, SQLite CLI, Docker, database service or paid external service is needed. Use a local disk, not a network share, for SQLite and attachments.

## Install and commands

Run from the repository root. The following commands work unchanged in **cmd.exe and PowerShell** (one per line). A fresh checkout needs dependencies before checks; `npm ci` is the reproducible alternative to `npm install`.

```text
go version
node --version
npm --version
npm install
npm run install:backend
npm run migrate
npm run bootstrap
npm run seed:development
npm run dev
npm run build
npm run start
npm test
npm test --workspace frontend -- setup
npm run test:backend -- ./internal/platform/...
npm run test:backend -- ./internal/httpapi/...
npm run test:contracts
npm run test:integration
npm run test:capacity
npm run backup
npm run restore:check
```

| Command | Behavior |
| --- | --- |
| install:backend | Downloads pinned Go modules in the backend module. |
| migrate | Applies embedded, ordered SQL migrations transactionally; rejects changed checksums, missing applied migrations and out-of-order additions. |
| bootstrap | **Unavailable, nonzero exit** until the account bootstrap task supplies private credential input. No account is created at startup. |
| seed:development | **Unavailable, nonzero exit** until the development-fixtures task implements valid related records. Never automatic. |
| dev | Builds Go, migrates non-destructively, starts Go and Angular on separate loopback ports, writes an ignored API proxy and waits for readiness. Prints the browser address; Ctrl+C stops the children. |
| build | Strict Angular compilation, production Angular build, Go server/operator builds into `var/bin`; copies assets when STATIC_DIR is configured. |
| start | Runs the built Go executable, migrates and checks storage, serves both API and compiled pages; default address `http://127.0.0.1:8080`. Recognized deep links support refresh. |
| test | Runs existing frontend unit tests once, Go tests, contract drift checks, then self-starting integration tests. Failures propagate; future feature suites join ordinary discovery. |
| test:backend | Forwards package arguments to `go test` with the backend working directory. |
| test:contracts | Validates OpenAPI references/route coverage and compares generated DTOs and the composed contract to their authoring source. |
| test:integration | Builds the app, locates/installs pinned Playwright Chromium, then starts its own app using an ephemeral loopback port and disposable database/files. Never uses a running preview or operator data. |
| test:capacity | Explicit expensive campaign; nonzero unavailable result until its owner supplies the campaign. |
| backup / restore:check | Nonzero unavailable results until the backup task implements consistent snapshots and disposable restore validation. |

The bootstrap/seed/backup commands above are documented extension slots, not successful operator procedures yet. After the account tasks land, first bootstrap must privately accept an operator's email/password and recovery instructions, create only the first Administrator, and reject a second bootstrap. Staff setup and recovery must use Administrator-issued expiring single-use links delivered through a verified private channel. **There are no fixed credentials, public registration, or demo login controls.** Authentication, reservation-expiry jobs, business rules, backup scheduling and feature pages belong to their owning tasks; do not use this foundation for live operations.

## Configuration and Windows examples

Optional `.env` is loaded from the app root; process environment wins. Defaults are listed in `.env.example`. Never commit `.env`, secrets or mutable data.

cmd.exe:
```bat
copy .env.example .env
set PORT=4200
npm run dev
```

PowerShell:
```powershell
Copy-Item .env.example .env
$env:PORT = '4200'
npm run dev
Remove-Item Env:PORT
```

Without PORT, development allocates an available frontend port; the backend always has a separate available port. Both bind only `127.0.0.1`. The `/api/v1` development proxy gives the browser one origin. Production-build local execution uses `npm run build` then `npm run start`; changing PORT does not require rebuilding. Build and runtime use the same STATIC_DIR. `APP_MODE` is development, production or test; COOKIE_SECURE is false only for local HTTP and must be true behind HTTPS. Paths resolve relative to the repository root, not the backend directory. Static and private directories must not overlap.

## Persistence and operational boundaries

SQLite uses a serialized writer connection, four query-only reader connections, foreign keys on every connection, WAL, synchronous FULL, and a 5000 ms busy timeout. Stateful writers use immediate transactions; repositories must use the injected transaction rather than independent commits. Queries receive request cancellation. Ordinary request context budgets are 30 seconds, reports 35 seconds and uploads 120 seconds; HTTP headers have a 5-second timeout, idle connections 60 seconds and graceful shutdown 15 seconds.

Startup applies migrations but never seeds or erases data. Keep database, WAL/SHM files and private attachments together across restarts. Never copy only a live SQLite database as a backup. Before operational use, choose separate protected backup storage, implement and validate consistent database/attachment snapshots and monthly disposable restore drills, and record disk-space thresholds and the operating privacy/retention policy. The example `var/backups` directory does not prove protected storage, hourly recovery points or a four-hour restore time. Incomplete backups must not be used for recovery; backup functionality is not delivered in this task.

`/api/v1/health/live` is process liveness. `/api/v1/health/ready` checks database access and writable attachment storage. Both return JSON envelopes; readiness failures are sanitized. Structured request logs contain request IDs, methods, status and duration, not bodies, tokens, query strings or paths. Unknown API routes return JSON, never Angular HTML. Missing assets remain 404. Optional service interfaces are fail-closed.

## Contracts and tests

Author the contract in `docs/api/contract.mjs`; run `npm run generate:contracts` after approved changes and check in `docs/openapi.yaml` and `frontend/src/app/core/contracts/generated/api.ts`. JSON syntax in the YAML file is valid YAML 1.2. `npm run test:contracts` is read-only and rejects drift. Contract coverage describes designed endpoints, not completed implementations. Stable route slots and dependency interfaces are documented in `docs/architecture.md`.

Go setup tests use actual migrations and temporary SQLite stores. Frontend tests use Vitest/TestBed in run mode. Integration support lives under `tests/integration`; saved operator journeys in `e2e` are excluded from unit discovery. No automated test requires an existing `.env`, database or server. Browser compatibility, full business acceptance, load measurements and recovery objectives require separate feature/release evidence, not an inference from a passing build.

### Setup verification (2026-11-03)

The five task acceptance commands passed: frontend setup (3 tests), platform Go packages, HTTP API Go package, contract reproducibility (177 paths / 229 operations / 544 schemas), and production build. Direct Playwright execution with `tests/integration/playwright.config.ts` also passed both foundation integration checks. This does not claim that every operator command or future business workflow was exercised.

Dependency audit is **not clean**: the installed graph reports 23 findings (20 moderate, one high, two critical), including Vite and Vitest/Tinypool development tooling. The reported Vitest remediation requires a major-version upgrade and compatibility work with the Angular test adapter. Review and remediate these before treating this setup as production-ready; keep development servers loopback-only and run tests only from trusted sources. `lucide-angular` also reports deprecation in favor of `@lucide/angular`; migration is not silently applied to future feature imports.
