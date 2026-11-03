# CSRF integration

`Manager{Key, Clock, Secure}` exposes `Issue(w, r, binding) (string, error)` and
`Validate(r, binding) error`. Supply at least 32 private random key bytes and an
injected clock. Keep the key stable across restarts and instances; do not mutate
manager dependencies concurrently.

- Binding is the server-resolved session token digest, or empty for anonymous
  bootstrap/login/reset. Rotate by calling Issue after successful login with the
  new session digest. The caller resolves the binding before validation.
- Issue sets a host-only readable `buildestate_csrf` cookie, path `/`, SameSite
  Lax, with a 30-minute signed expiry and no-store response. HTTPS requests or
  Secure mode require the Secure cookie flag.
- Validate requires exactly one cookie, header `X-CSRF-Token`, and Origin. It
  checks cookie/header equality, signature, and binding using constant-time
  comparisons and rejects expired tokens. Call on every browser mutation,
  including anonymous mutations; method-based exemptions are the caller's job.
- Origin must exactly equal `http://` or `https://` plus the request Host. TLS or
  Secure selects HTTPS. Forwarded, X-Forwarded-Host, X-Forwarded-Proto, and
  X-Forwarded-Port are rejected, not trusted. A trusted reverse-proxy deployment
  must explicitly strip these headers and preserve the external Host while
  configuring Secure; this package does not implement proxy trust.
- Map ErrInvalid to the safe CSRF denial response. Configuration/randomness
  failures must fail closed. Issuance does not revoke prior nonces: old signed
  tokens remain usable until expiry only with matching cookie and binding.

Tests are in csrf_test.go. They were added without running them; the parent owns
test execution and HTTP/session wiring.
