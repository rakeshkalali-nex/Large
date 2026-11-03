# Company settings and data stewardship — screen 35

Scope: task 25 / issue #7. Browser route `/settings`; administrator access is restored through the real identity endpoint before the page loads. Server authorization remains authoritative.

## Delivered interface

- Company identity and branding, explicit currency and timezone confirmation, reservation defaults, property types, editable reference values and inspection checklists.
- Owner-approved operating-country, notice and retention-schedule references; no invented legal jurisdiction or compliance claim.
- Project overrides for reservation duration, site/material minimums, checklist selection and operational references, without authorization/approval overrides.
- Notice acknowledgment and consent recording, privacy request creation/review/execution, legal holds, and explicit reviewed retention actions.
- Change history and local operator status with safe diagnostics and an explicit missing-backup state.
- Responsive named sections, labeled controls, pending-action blocking, announced failures, preserved safe input and deliberate retries with stable request keys.

## Verification inventory

`frontend/src/app/features/administration/settings/settings.spec.ts` exercises route composition, explicit setup prerequisites, editable-field request projection, currency lock display, stable retry keys, duplicate-submit blocking, conflict comparison, execution confirmation, access denial and CSRF headers. These tests use Angular TestBed and HTTP test doubles; they are not real-backend browser evidence.

Backend verification and final executed command results are recorded in `validation.md`. Task 28 owns the actual backup scheduler/restore implementation and its failure-state campaign. Issues #32 and #15 own the separately scheduled property-test campaigns.

## Manual acceptance still required

Use isolated data and normally authenticated staff accounts; never provide a login bypass. Confirm saves and stewardship actions against the real server, reload the page and restart the backend, inspect retained database evidence, and verify all seven roles are denied or allowed appropriately. Exercise keyboard navigation, 200% zoom and 390/768/1024/1440px widths. Real-device/browser screenshots and WCAG review are not implied by unit-test success.

Operating-policy approval is an owner/operator responsibility. See `docs/operations/policy.md` before entering live personal data.
