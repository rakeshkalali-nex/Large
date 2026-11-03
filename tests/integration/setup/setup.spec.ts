import { test, expect } from '@playwright/test';

test('setup serves database-aware health and JSON unknown API routes', async ({ request }) => {
  const ready = await request.get('/api/v1/health/ready');
  expect(ready.status()).toBe(200);
  const missing = await request.get('/api/v1/__setup_unknown_route__');
  expect(missing.status()).toBe(404);
  expect(missing.headers()['content-type']).toContain('application/json');
  expect((await missing.json()).error.code).toBe('NOT_FOUND');
});

test('setup serves the Angular application on direct deep-link refresh', async ({ page }) => {
  const response = await page.goto('/projects/setup-record/plan');
  expect(response?.status()).toBe(200);
  await expect(page.locator('app-root')).toBeAttached();
  const refreshed = await page.reload();
  expect(refreshed?.status()).toBe(200);
  await expect(page.locator('app-root')).toBeAttached();
});
