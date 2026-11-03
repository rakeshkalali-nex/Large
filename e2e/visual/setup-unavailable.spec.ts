import { test, expect } from '@playwright/test';

test('foundation unavailable page visual baseline', async ({ page }) => {
  await page.goto('/projects');
  await expect(page.getByRole('heading', { name: 'BuildEstate' })).toBeVisible();
  await expect(page.getByRole('status')).toHaveText('This workspace is not available yet. No business operation has been performed.');
  await expect(page).toHaveScreenshot('setup-unavailable.png', { animations: 'disabled' });
});
