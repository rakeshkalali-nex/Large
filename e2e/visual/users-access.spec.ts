import { test, expect, login } from '../wave16-fixtures';

test('users and access visual baseline', async ({ page, staffServer }) => {
  await login(page, staffServer);
  await expect(page.getByRole('heading', { name: 'Acceptance Administrator', exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByRole('button', { name: 'Create account', exact: true })).toBeVisible();
  await page.evaluate(() => document.fonts.ready);
  await expect(page).toHaveScreenshot('users-access.png', { animations: 'disabled' });
});
