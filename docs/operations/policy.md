# Local operating policy and stewardship

Domain integration: `settings.RequirePersonalDataReady(ctx, tx)` is the transaction-scoped entry gate for future staff, lead, customer, supplier and contractor personal-record creation services. Acknowledgments call it now. Other domain route slots are not implemented by task 25; this delivery does not claim their future live-entry integration is complete. Bootstrap administrator identity remains a separate setup prerequisite.

Before live personal-record entry, the owner must approve and retain local documentation naming the operating countries, privacy notice version, retention schedule version, purpose and necessity of the default personal fields, any explicit-consent requirements, and approval evidence. Enter those references in Company settings and explicitly confirm the operational policy. No jurisdiction, legal conclusion or company policy is supplied by the application.

Explicitly choose the base currency and minor-unit precision and confirm an IANA timezone (UTC is initially presented). Posting checks these settings in its write transaction. The first financial posting locks the currency. Changing policy references requires renewed confirmation. Recovery instructions must identify the real local recovery process without passwords, tokens or internal storage paths.

Settings changes preserve versioned snapshots and append an audit event atomically. Conflicting versions must be reloaded. Existing checklist revisions are immutable: create a new checklist (empty id) and remove the previous revision from the active selection instead of changing an already-used checklist. Archiving reference values does not remove historical business references. Only reservation duration, stock minimums, checklist choices and operational reference values are overridable. Authorization and approval separation are never settings.

## Personal information

Collect only staff identity/assignments, buyer contact/transactions, supplier/contractor contacts, and necessary evidence. Record notice acknowledgments against existing subjects and the confirmed notice version. Acknowledgments are not a claim that every processing purpose requires consent. Owner-approved consent rules must be followed; refusal is not an authorization to process.

The confirmed company's active reference value `privacy.consent-required` with value `true` requires explicit affirmative consent for acknowledgment recording. Configure its single true/false value only according to the approved local policy. Absence fails closed as required; an explicit false value allows consent to be true, false, or null. A client's consent-required flag must match the server policy. Project overrides cannot supply privacy-policy keys. Only affirmative consent supplies a current lead/customer consent reference.

Administrators create access, correction or anonymization requests, record a reasoned review, and explicitly execute approved requests. Execution rechecks current holds and state in the same transaction as changes and audit evidence. Access responses are limited contact projections, not credentials or unrestricted financial exports. Anonymization replaces unnecessary contacts without deleting IDs, transactions, attribution or evidence. Holds must be explicitly released with a reason, never silently expired on the review date.

## Provisional retention schedule

| Data | Earliest eligibility |
|---|---|
| Business, financial and important mutation/approval evidence | Seven years after parent closure |
| Unconverted lead contact details | Two years after last activity |
| Unnecessary deactivated staff profile details | 90 days after deactivation |
| Expired session/token hashes | 30 days after expiry |
| Routine authentication events | One year |

These are provisional application periods, subject to owner approval and recorded legal holds. Review candidates are not a deletion instruction: execution must recheck eligibility, current activity and direct/inherited holds. Required business/financial evidence remains retained; privacy requests do not override retention. Future-dated reviews must not enable premature destruction. Review dates do not release holds.

## Operator status

Company settings exposes administrator-only sanitized database, attachment and backup status. Unknown or unconfigured backup capability is not a successful backup. The backup implementation and recovery drills belong to the backup delivery task; do not start live operations until those prerequisites and the owner’s restoration drill are complete. Never copy raw driver errors, file paths, secrets or stack traces into operator status. Use diagnostic references for local troubleshooting.
