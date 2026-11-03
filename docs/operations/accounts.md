# Staff accounts and password recovery

BuildEstate has no public registration, demo login, password lookup, or required
email/SMS service. Administrators provision staff through the administration
workflow. Password setup and recovery use private one-time links, not passwords
chosen or retrieved by an Administrator.

## First Administrator

Use the explicit local `npm run bootstrap` operator command on the application
host after configuring persistent SQLite storage. It accepts private
operator-supplied credentials through protected standard input, not command-line
arguments. Never place a password in shell history, an environment file, source
control, a ticket, or a command transcript. Do not redirect the input or output
to an application log. Run only on a trusted, non-recorded operator terminal.

Bootstrap is permitted only when **no staff account exists**. It is not a reset
or backdoor mechanism. It stores an Argon2id password hash, creates the first
Administrator and public recovery contact settings in one transaction, and
records secret-free audit evidence. Concurrent or repeated initialization must
not create another Administrator. Failure must leave no partially created user.

Choose a unique password of 12–128 Unicode characters. Company recovery contact
labels and instructions are intentionally public on Sign in: include only the
company-approved contact procedure, never a password, reset link, private
identity-verification answer, or internal secret. Company currency and business
policy settings still require their separate authorized configuration.

Supply exactly one UTF-8 JSON object (maximum 16 KiB) containing `displayName`,
`email`, `password`, `companyName`, `recoveryContactLabel`, and
`recoveryInstructions`. All fields are required. Unknown fields and trailing
input are rejected. Interactive terminal input is deliberately rejected to
avoid echoing a password. Use an operator-protected temporary file outside the
application folder or a private credential-provider pipe. Restrict that file
to the operator before writing credentials; remove it promptly after use.
The command initializes/migrates the configured database before creation.

For an already protected input file in the system temporary directory, cmd.exe:

```bat
npm run bootstrap < "%TEMP%\buildestate-bootstrap.json"
```

PowerShell can invoke cmd.exe redirection without putting credentials in history:

```powershell
cmd.exe /d /c 'npm run bootstrap < "%TEMP%\buildestate-bootstrap.json"'
```

These are invocation examples, not credential templates. Never use `echo` with
inline credentials or paste a credential-bearing JSON object into a terminal
command. Recovery guidance is stored on the existing company-settings row
`company`. Bootstrap emits only a generic success or safe failure message.

## Trusted identity verification and delivery

1. Staff select recovery guidance on Sign in, or consult their established
   company Administrator contact. Guidance does not reveal whether an account
   exists and does not automatically send a token.
2. The Administrator verifies the person **in person or through a previously
   established known-number conversation**. A number/address newly supplied by
   the requester, possession of an email address alone, or a forwarded chat is
   not identity verification. Follow company escalation policy when uncertain.
3. After verification, an active Administrator issues a link using
   `POST /api/v1/users/{id}/password-reset`, supplying the current user version,
   `purpose`, `identityVerified: true`, and a safe operational reason. Do not put
   identity answers or secrets in the reason. Setup purpose is available only
   for an account requiring setup.
4. Deliver the returned link privately to that verified person through the
   established trusted channel. Never post it in a group, ticket, audit note,
   analytics event or application log. The response is shown once. If delivery
   is uncertain or the response is lost, reverify and issue a replacement; do
   not attempt to recover a token from storage.
5. The person completes setup/reset with their own new password, then signs in
   again. The Administrator never needs to know that password.

Setup links expire after **24 hours**; recovery links after **30 minutes**.
Tokens are single-use and only their digests and lifecycle metadata persist.
Issuing a replacement revokes outstanding tokens of the same purpose. Unknown,
expired, revoked and consumed tokens all receive the same safe rejection. No
password changes on rejection: contact an Administrator for identity
verification and a replacement link.

## Session and client behavior

Setup-required accounts have no domain access; only setup completion and logout
are permitted mutations. An ordinary authenticated password change requires
the current password. All successful setup/change/recovery operations atomically
write the new Argon2id hash, invalidate applicable tokens and all prior sessions,
advance access revision, and record a secret-free audit event. They do not
automatically create a new session.

Links place their token in the `/login` URL **fragment**, never a query string.
The client must extract it into transient memory and immediately remove it from
the address bar/history. Do not use browser persistent storage, referrers,
analytics, request-body logging, or automatic mutation replay for these flows.
Fetch the CSRF bootstrap and send its signed cookie/header and a fresh UUID
`Idempotency-Key` with same-origin password mutations. Committed issuance replay
returns `409 IDEMPOTENCY_MISMATCH` rather than reconstructing a secret link.
Treat expired sessions and connectivity failures as explicit
states requiring user action rather than silently resubmitting credentials.

## Security incident handling

For a disclosed link, issue a replacement after verification. For an account
compromise, use the authorized account deactivation/recovery procedure and
review secret-free audit history. Do not edit hashes or token rows manually.
If every Administrator is locked out, escalate to the company operator and
approved incident/restore procedure; rerunning bootstrap cannot bypass the
existing staff check.

The API contract is `docs/api/passwords.yaml`. Browser presentation and staff
creation remain owned by their respective Sign in and Users-and-access tasks;
this feature supplies the backend lifecycle and operator bootstrap.
