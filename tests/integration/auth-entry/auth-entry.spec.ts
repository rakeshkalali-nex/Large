import type { Page } from '@playwright/test';
import { randomBytes, randomUUID } from 'node:crypto';
import { test, expect } from './fixtures';

// Login bodies and cookies must not enter persisted Playwright artifacts.
test.use({ trace: 'off', screenshot: 'off', video: 'off' });

const loginPath = '/api/v1/auth/login';
const isLogin = (url: string) => new URL(url).pathname === loginPath;
const emailInput = (page: Page) => page.getByLabel(/^Email(?: address)?$/i);
const recoveryAction = (page: Page) => page.getByRole('link', { name: /forgot|recover/i })
  .or(page.getByRole('button', { name: 'Forgot password?', exact: true }));

async function credentials(page: Page, email: string, password: string) {
  await emailInput(page).fill(email);
  await page.getByLabel('Password', { exact: true }).fill(password);
}

async function signIn(page: Page, email: string, password: string) {
  await credentials(page, email, password);
  const response = page.waitForResponse(response => isLogin(response.url()) && response.request().method() === 'POST');
  await page.getByLabel('Password', { exact: true }).press('Enter');
  expect((await response).status()).toBe(200);
  await expect(page).toHaveURL(/\/dashboard$/);
}

test('real login, persisted root restore and authenticated login visit', async ({ page, accountServer }) => {
  await page.goto('/');
  await expect(page).toHaveURL(/\/login(?:\?.*)?$/);
  await expect(page.getByRole('heading', { name: 'Welcome back', exact: true })).toBeVisible();
  await expect(page.getByLabel('Password', { exact: true })).toHaveAttribute('type', 'password');
  await expect(page.getByLabel('Password', { exact: true })).toHaveAttribute('autocomplete', 'current-password');
  await signIn(page, accountServer.email, accountServer.password);
  const restored = page.waitForResponse(response => new URL(response.url()).pathname === '/api/v1/auth/me');
  await page.goto('/');
  expect((await restored).status()).toBe(200);
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.goto('/login');
  await expect(page).toHaveURL(/\/dashboard$/);
  // Landing URL only: dashboard content belongs to a later feature.
});

test('invalid credentials retain email, clear password and expose only a safe error', async ({ page, accountServer }) => {
  await page.goto('/login');
  await credentials(page, accountServer.email, 'wrong-' + accountServer.password);
  const response = page.waitForResponse(response => isLogin(response.url()));
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  expect((await response).status()).toBe(401);
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(emailInput(page)).toHaveValue(accountServer.email);
  await expect(page.getByLabel('Password', { exact: true })).toHaveValue('');
  await expect(page.getByRole('alert')).not.toContainText(/SQL|stack trace|password_hash|argon2|account does not exist/i);
  await expect(page).toHaveURL(/\/login(?:\?.*)?$/);
});

test('network failure is explicit, clears password and never automatically resubmits', async ({ page, accountServer }) => {
  await page.goto('/login');
  let attempts = 0;
  // Fault injection aborts transport; no authentication response is fabricated.
  await page.route('**/api/v1/auth/login', async route => { attempts++; await route.abort('connectionfailed'); });
  await credentials(page, accountServer.email, accountServer.password);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(emailInput(page)).toHaveValue(accountServer.email);
  await expect(page.getByLabel('Password', { exact: true })).toHaveValue('');
  await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeEnabled();
  await page.waitForTimeout(1500); // bounded observation window for unintended retry
  expect(attempts).toBe(1);
});

test('duplicate submit stays single-flight until the real Go response arrives', async ({ page, accountServer }) => {
  await page.goto('/login');
  let attempts = 0;
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/api/v1/auth/login', async route => { attempts++; await gate; await route.continue(); });
  await credentials(page, accountServer.email, accountServer.password);
  const submit = page.getByRole('button', { name: 'Sign in', exact: true });
  try {
    await submit.click();
    await expect.poll(() => attempts).toBe(1);
    await expect(page.locator('button[type="submit"]')).toBeDisabled();
    await page.keyboard.press('Enter');
    await page.locator('form').evaluate(form => {
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
      form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    });
    expect(attempts).toBe(1);
  } finally { release(); }
  await expect(page).toHaveURL(/\/dashboard$/);
  expect(attempts).toBe(1);
});

for (const returnUrl of ['https://example.invalid/escape', '//example.invalid/escape', '/\\example.invalid', '/%2f%2fexample.invalid', '/projects/../users', '/not-a-registered-route']) {
  test(`rejects unsafe return URL ${returnUrl}`, async ({ page, accountServer }) => {
    await page.goto(`/login?returnUrl=${encodeURIComponent(returnUrl)}`);
    await signIn(page, accountServer.email, accountServer.password);
    expect(new URL(page.url()).origin).toBe(accountServer.origin);
  });
}

test('authorized return URL is preserved instead of the default landing', async ({ page, accountServer }) => {
  await page.goto('/login?returnUrl=%2Fprojects');
  await credentials(page, accountServer.email, accountServer.password);
  const response = page.waitForResponse(response => isLogin(response.url()));
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  expect((await response).status()).toBe(200);
  await expect(page).toHaveURL(/\/projects$/);
});

test('recovery link presents the persisted private-contact guidance', async ({ page }) => {
  await page.goto('/login');
  await recoveryAction(page).click();
  await expect(page.getByRole('heading', { name: 'Recover your account', exact: true })).toBeVisible();
  await expect(page.getByText('Smoke account administrator', { exact: true })).toBeVisible();
  await expect(page.getByText(/identity verification through a trusted private channel/)).toBeVisible();
});

test('authenticated password change persists and requires sign-in with the new password', async ({ page, accountServer }) => {
  await page.goto('/login');
  await signIn(page, accountServer.email, accountServer.password);
  await page.goto('/login?state=change');
  await expect(page.getByRole('heading', { name: 'Change your password', exact: true })).toBeVisible();
  const newPassword = randomUUID();
  await page.getByLabel('Current password', { exact: true }).fill(accountServer.password);
  await page.getByLabel('New password', { exact: true }).fill(newPassword);
  await page.getByLabel('Confirm new password', { exact: true }).fill(newPassword);
  const changed = page.waitForResponse(response => new URL(response.url()).pathname === '/api/v1/auth/change-password'
    && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Save password', exact: true }).click();
  expect((await changed).status()).toBe(200);
  await expect(page.getByRole('heading', { name: 'Password updated', exact: true })).toBeVisible();
  expect((await page.request.get('/api/v1/auth/me')).status()).toBe(401);
  await page.getByRole('button', { name: 'Back to sign in', exact: true }).click();
  await expect(page.getByLabel('Password', { exact: true })).toHaveValue('');
  await signIn(page, accountServer.email, newPassword);
});

test('unknown reset token is scrubbed from the URL and rejected by the real backend', async ({ page }) => {
  // Well-formed but never issued: exercises the same safe unavailable-link state
  // as expired/used/replaced tokens without altering clocks or production data.
  const token = randomBytes(32).toString('base64url');
  const newPassword = randomUUID();
  const requestURLs: string[] = [];
  page.on('request', request => requestURLs.push(request.url()));
  await page.goto(`/login?state=reset#token=${token}`);
  await expect(page.getByRole('heading', { name: 'Set a new password', exact: true })).toBeVisible();
  await expect.poll(() => new URL(page.url()).hash).toBe('');
  expect(await page.locator('body').innerText()).not.toContain(token);
  await page.getByLabel('New password', { exact: true }).fill(newPassword);
  await page.getByLabel('Confirm new password', { exact: true }).fill(newPassword);
  const reset = page.waitForResponse(response => new URL(response.url()).pathname === '/api/v1/auth/reset-password'
    && response.request().method() === 'POST');
  await page.getByRole('button', { name: 'Save password', exact: true }).click();
  expect((await reset).status()).toBe(422);
  await expect(page.getByRole('heading', { name: 'This link is no longer available', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Contact an Administrator', exact: true })).toBeVisible();
  expect(new URL(page.url()).hash).toBe('');
  expect(requestURLs.some(url => url.includes(token))).toBe(false);
  expect((await page.request.get('/api/v1/auth/me')).status()).toBe(401);
});

test('malformed reset fragment is removed without sending a password mutation', async ({ page }) => {
  let mutations = 0;
  page.on('request', request => {
    if (new URL(request.url()).pathname === '/api/v1/auth/reset-password') mutations++;
  });
  await page.goto('/login?state=reset#token=malformed');
  await expect(page.getByRole('heading', { name: 'This link is no longer available', exact: true })).toBeVisible();
  await expect.poll(() => new URL(page.url()).hash).toBe('');
  await expect(page.getByLabel('New password', { exact: true })).toHaveCount(0);
  expect(mutations).toBe(0);
});

for (const width of [1440, 1024, 768, 390]) {
  test(`branded sign-in reflows without overflow at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/login');
    await expect(page.getByRole('heading', { name: 'Welcome back', exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeInViewport();
    if (width > 1024) {
      await expect(page.getByRole('complementary', { name: 'BuildEstate' })).toBeVisible();
      const form = await page.locator('form').boundingBox();
      expect(form?.width).toBeGreaterThanOrEqual(400);
      expect(form?.width).toBeLessThanOrEqual(440);
    } else {
      await expect(page.locator('.mobile-brand')).toBeVisible();
      await expect(page.getByRole('complementary', { name: 'BuildEstate' })).toBeHidden();
    }
  });
}

test('password visibility is keyboard accessible and preserves the entered value', async ({ page }) => {
  await page.goto('/login');
  const password = page.getByLabel('Password', { exact: true });
  const value = randomUUID();
  await password.fill(value);
  await password.press('Tab');
  await expect(page.getByRole('button', { name: 'Show password', exact: true })).toBeFocused();
  await page.keyboard.press('Space');
  await expect(password).toHaveAttribute('type', 'text');
  await expect(password).toHaveValue(value);
  await page.keyboard.press('Space');
  await expect(password).toHaveAttribute('type', 'password');
  await expect(password).toHaveValue(value);
});

test('mobile sign-in remains reachable after a keyboard-like viewport shrink', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'Welcome back', exact: true })).toBeVisible();
  await page.getByLabel('Password', { exact: true }).focus();
  await page.setViewportSize({ width: 390, height: 390 });
  const submit = page.getByRole('button', { name: 'Sign in', exact: true });
  await submit.scrollIntoViewIfNeeded();
  await expect(submit).toBeInViewport();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  const recovery = recoveryAction(page);
  await recovery.scrollIntoViewIfNeeded();
  await expect(recovery).toBeInViewport();
});

for (const [state, heading] of [
  ['setup', 'Set up your password'],
  ['reset', 'This link is no longer available'],
  ['expired', 'Your session has expired'],
  ['change', 'Your session has expired'],
  ['recovery', 'Recover your account'],
] as const) {
  test(`anonymous lifecycle state ${state} stays inside the account shell`, async ({ page }) => {
    await page.goto(`/login?state=${state}`);
    await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
    await expect(page).toHaveURL(/\/login\?/);
    if (state === 'reset' || state === 'setup') {
      await expect(page.getByLabel('New password', { exact: true })).toHaveCount(0);
    }
  });
}
