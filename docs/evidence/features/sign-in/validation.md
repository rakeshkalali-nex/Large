# Sign-in validation — 2026-11-03

## Wave 15 continuation

Preserved the earlier sign-in implementation. Corrected the foundation test in
`frontend/src/app/setup/setup.spec.ts`: its top-level-only count expected ten lazy
routes, although protected features now correctly sit under the session guard.
The assertion now checks one public lazy mount, nine guarded lazy mounts, the
access-denied route and the final protected fallback. This small out-of-scope test
change is required by the reported wave failure; production guards were not weakened.

After `npm ci` completed, all required repository-root commands exited 0:

- `npm test --workspace frontend -- auth`: 27 tests passed.
- `npm test --workspace frontend -- session`: 5 tests passed.
- `npm test --workspace frontend -- guards`: 23 tests passed.
- `npm run test:integration -- auth-entry`: 26 browser tests passed.
- Full regression `npm test`: 104 frontend tests (including the three setup tests),
  Go tests, contract validation, production builds and 28 integration tests passed.

An initial auth invocation before installation finished could not find Vitest;
the post-install run above passed. Installation reported 23 dependency advisories;
no dependency versions were changed. Physical mobile keyboards, screen readers,
Firefox/WebKit and screenshot baselines were not checked in this continuation.

## Earlier implementation evidence

Environment: Windows 11, repository-root commands, pinned npm dependencies installed with `npm ci`.

| Required command | Result |
| --- | --- |
| `npm test --workspace frontend -- auth` | Exit 0; 27 tests across two files |
| `npm test --workspace frontend -- session` | Exit 0; 5 tests |
| `npm test --workspace frontend -- guards` | Exit 0; 23 tests including fragment-before-restoration behavior |
| `npm run test:integration -- auth-entry` | Exit 0; 26 browser tests passed, including real password change, reset-token scrubbing, four viewport widths and keyboard password visibility |

All four commands were rerun from the repository root after resuming the preserved implementation. The initial auth invocation could not find Vitest in the fresh checkout; `npm ci` installed the locked dependencies, after which all required commands exited 0. The integration command's Angular strict compilation/production build and Go build completed successfully. No system service was started or reconfigured; test-owned Go children and isolated SQLite stores are started and removed by the test harness.

An independent read-only review approved the scoped implementation with no blocking findings. The continuation retained the working implementation and added four-width layout/form-size checks and a keyboard-operated password visibility browser case.

`npm ci` reported 23 dependency advisories (20 moderate, one high, two critical). Dependencies were not changed by this scoped task and advisories were not remediated here. Real mobile keyboard behavior, Firefox/WebKit, screen-reader behavior and visual screenshot baselines were not checked.
