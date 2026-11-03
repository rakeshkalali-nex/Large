# Shared shell integration — issue 2

## Delivered boundaries

The app root wraps the existing router outlet in `AppShell`; anonymous, restoring and setup-required sessions retain the account-page outlet without authorized navigation. This task does not implement authentication, bootstrap users or business pages. The authentication owner must feed the generated `Identity` response into `SessionStore.accept`, call `clear` on logout, and provide `SHELL_LOGOUT` with a server-revoking action. There is intentionally no mock identity, registration, local-storage session or demo bypass.

The navigation registry filters exact server capability strings, never role names or wildcard administrator shortcuts. The access owner should align its capability registry with `core/shell/navigation.ts`. Capabilities only guide visibility; Go remains the security boundary. Project selectors currently display authorized project IDs from Identity; feature integration may supply authorized display names. Route `projectId` parameters (and project detail `id`) take precedence over the selector. Detail routes should supply a router title; shell supplies a safe generic fallback without showing resource IDs in headings.

`ProjectContext` exposes `{projectId,revision}` and emits `project.changed`. Null means all authorized projects, not unrestricted company access. Dirty forms register an async confirmation with `UnsavedChanges` and unregister on destruction. They can use `ConfirmationService` and attach `unsavedChangesGuard` to editor routes. Scope changes are abandoned on cancellation or concurrent session replacement. `ScopedQueryCache` invalidates on all closed UI events and rejects responses from an obsolete generation.

`UiEvents.committed` accepts the generated Effects plus request ID only after confirmed server success and emits the specified record/stock/unit/finance payloads. Session changes emit status/userId/accessRevision. These are in-memory invalidation hints only.

`MutationState` belongs to one safe-input form, disables parallel submissions, never automatically retries, and retains an idempotency key across an explicitly repeated unchanged payload. Edited payloads receive new keys. Feature API clients must send that key as Idempotency-Key and invoke committed effects only after success. Do not retain passwords or reset tokens here. Error mapping uses safe local text rather than raw server bodies. Domain forms own 422 field mapping and 409 current-version comparisons; an error never clears their recoverable values. Session loss clears identity; feature secret fields must also clear on session.changed.

## Minimal composition exceptions

Outside the expected directories, `frontend/src/app/app.ts` mounts the shell, `app.config.ts` installs the safe non-retrying interceptor, and `frontend/angular.json` includes the global stylesheet. No feature route ownership or specification files changed. Existing dependencies already include CDK, Lucide and Fontsource; manifests and lockfile were not changed.

## Validation and remaining browser acceptance

The required shared/core Vitest suites and Angular production build are run from the repository root. TestBed checks permission filtering, responsive breakpoint behavior at 1440/1024/768/390, CDK Escape dismissal, detail context/breadcrumbs, dirty context cancellation, cache invalidation, safe HTTP failures, concurrency, deliberate retry, formatting and shared interaction semantics. Unit tests do not establish real browser layout or screen-reader compatibility.

Task 30 must exercise authenticated business pages in current/previous Chrome, Edge, Firefox and Safari, Android Chrome and iOS Safari: actual widths and 200–400% zoom; keyboard route-heading focus, dialogs and focus restoration; screen-reader error/status announcements; reduced motion; phone keyboard and photo controls; no page-wide overflow; and local IBM Plex Sans network/computed-style loading for weights 400–700. This task does not claim those checks passed. See theme.md for licensing and contrast calculations.

Dependency installation reported 23 existing audit vulnerabilities (20 moderate, 1 high, 2 critical) and deprecated lucide-angular. No unrequested dependency upgrades or audit fixes were applied.
