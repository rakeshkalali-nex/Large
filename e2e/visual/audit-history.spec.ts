import { test, expect, login } from '../wave16-fixtures';

test('audit history visual baseline', async ({ page, staffServer }) => {
  await login(page, staffServer, 'admin', '/audit');
  await page.getByLabel('Action', { exact: true }).fill('site-log.approve');
  await page.getByRole('button', { name: 'Search events', exact: true }).click();
  await expect(page.getByRole('list', { name: 'Audit events' }).getByRole('listitem')).toHaveCount(2);
  if (page.viewportSize()!.width > 600) await page.getByLabel('Record type', { exact: true }).hover();
  await page.evaluate(() => document.fonts.ready);
  await expect(page).toHaveScreenshot('audit-history.png', { animations: 'disabled', mask: [page.locator('.event-row small')] });
});
