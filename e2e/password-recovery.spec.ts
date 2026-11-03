import { test, expect } from '@playwright/test';

// API-owned browser journey. Password forms are delivered by Issue #8.
// No credentials, seeded accounts, placeholder UI, or mocked routes are needed.
test('recovery guidance and invalid-link replacement through the real browser origin', async ({ page }) => {
  await page.goto('/api/v1/auth/recovery-guidance');
  const guidance = await page.evaluate(async () => {
    const response = await fetch('/api/v1/auth/recovery-guidance');
    return { status: response.status, body: await response.json(), referrerPolicy: response.headers.get('referrer-policy') };
  });
  expect(guidance.status).toBe(200);
  expect(guidance.referrerPolicy).toBe('no-referrer');
  expect(guidance.body.data.contactLabel).toBeTruthy();
  expect(guidance.body.data.instructions).toContain('identity verification');
  expect(guidance.body.data.instructions).toContain('trusted private channel');
  expect(guidance.body.data.instructions).toContain('replacement');

  const result = await page.evaluate(async () => {
    const csrfResponse = await fetch('/api/v1/auth/csrf');
    const csrf = await csrfResponse.json();
    // Generated unknown token, never an account's actual reset credential.
    const token = btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(32))))
      .replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
    const response = await fetch('/api/v1/auth/reset-password', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrf.data.csrfToken,
        'Idempotency-Key': crypto.randomUUID(),
      },
      body: JSON.stringify({ token, newPassword: crypto.randomUUID() }),
    });
    const body = await response.json();
    const identity = await fetch('/api/v1/auth/me');
    return { csrfStatus: csrfResponse.status, status: response.status, body, identityStatus: identity.status };
  });
  expect(result.csrfStatus).toBe(200);
  expect(result.status).toBe(422);
  expect(result.body.error.code).toBe('INVALID_RESET_TOKEN');
  expect(result.body.error.message).toContain('identity verification');
  expect(result.body.error.message).toContain('replacement link');
  expect(result.identityStatus).toBe(401);
});
