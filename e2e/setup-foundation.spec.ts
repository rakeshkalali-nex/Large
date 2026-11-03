import { test, expect } from '@playwright/test';

test('foundation deep link remains safely unavailable after refresh', async ({ page }) => {
  await page.goto('/projects');
  await expect(page.getByRole('heading', { name: 'BuildEstate' })).toBeVisible();
  await expect(page.getByText('This workspace is not available yet. No business operation has been performed.')).toBeVisible();
  await page.reload();
  await expect(page.getByRole('status')).toHaveText('This workspace is not available yet. No business operation has been performed.');
});

test('foundation browser origin reaches real backend readiness', async ({ page }) => {
  await page.goto('/projects');
  await expect(page.getByRole('heading', { name: 'BuildEstate' })).toBeVisible();
  const health = await page.evaluate(async () => {
    const response = await fetch('/api/v1/health/ready');
    return { status: response.status, body: await response.json(), id: response.headers.get('x-request-id') };
  });
  expect(health.status).toBe(200);
  expect(health.body.data.status).toBe('ready');
  expect(health.body.requestId).toBe(health.id);
  expect(health.id).toBeTruthy();
});
