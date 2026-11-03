import { test, expect, login } from './wave16-fixtures';

test('administrator filters paginated immutable audit evidence and inspects safe details', async ({ page, staffServer }) => {
  await login(page, staffServer, 'admin', '/audit');
  await expect(page.getByRole('heading', { name: 'Audit history', exact: true })).toBeVisible();
  await expect(page.getByRole('list', { name: 'Audit events' }).getByRole('listitem')).toHaveCount(25);
  await page.getByRole('button', { name: 'Next page', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Next page', exact: true })).toBeDisabled();
  await page.getByLabel('Action', { exact: true }).fill('site-log.approve');
  await page.getByLabel('Project ID', { exact: true }).fill('project-a');
  await page.getByLabel('Actor ID', { exact: true }).fill(staffServer.adminId);
  await page.getByLabel('From (UTC)', { exact: true }).fill('2025-12-31T00:00');
  await page.getByLabel('To (UTC)', { exact: true }).fill('2026-01-02T00:00');
  await page.getByRole('button', { name: 'Search events', exact: true }).click();
  const events = page.getByRole('list', { name: 'Audit events' }).getByRole('listitem');
  await expect(events).toHaveCount(1);
  await expect(events).toContainText('Acceptance North');
  await expect(events).toContainText('2026-01-01 10:00:00 UTC');
  await page.getByRole('button', { name: 'View details for site-log.approve approval-own', exact: true }).click();
  const drawer = page.getByRole('dialog');
  await expect(drawer.getByRole('heading', { name: 'Audit event details' })).toBeVisible();
  await expect(drawer).toContainText('request-approval');
  await expect(drawer).toContainText('Reviewed site evidence');
  await expect(drawer).toContainText('Before');
  await expect(drawer).toContainText('submitted');
  await expect(drawer).toContainText('After');
  await expect(drawer).toContainText('approved');
  await expect(drawer).toContainText('2028-06-15 10:30:00 UTC');
  await expect(drawer.getByRole('link', { name: 'Open record' })).toHaveAttribute('href', '/site-logs/log-own');
  await expect(drawer.getByRole('textbox')).toHaveCount(0);
  await page.getByRole('button', { name: 'Close audit event details' }).click();
  await expect(page.getByRole('button', { name: 'View details for site-log.approve approval-own', exact: true })).toBeFocused();
  await page.getByRole('button', { name: 'Reset filters' }).click();
  await page.getByRole('searchbox', { name: 'Search', exact: true }).fill('no matching audit evidence');
  await page.getByRole('button', { name: 'Search events', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'No audit events found' })).toBeVisible();
});

for (const role of ['finance', 'manager']) {
  test(`${role} reads only assigned authorized audit trails`, async ({ page, staffServer }) => {
    await login(page, staffServer, role, '/audit');
    await expect(page.getByRole('heading', { name: 'Audit history', exact: true })).toBeVisible();
    const events = page.getByRole('list', { name: 'Audit events' });
    await expect(events).not.toContainText('Acceptance South');
    await expect(events).not.toContainText('user.create');
    if (role === 'finance') {
      await expect(events.getByRole('listitem')).toHaveCount(25);
      await expect(events).toContainText('invoice.post');
      await expect(events).not.toContainText('site-log.approve');
      await page.getByRole('button', { name: 'Next page', exact: true }).click();
      await expect(events.getByRole('listitem')).toHaveCount(3);
    } else {
      await expect(events.getByRole('listitem')).toHaveCount(1);
      await expect(events).toContainText('site-log.approve');
      await expect(events).not.toContainText('invoice.post');
    }
    await events.getByRole('button').first().click();
    const drawer = page.getByRole('dialog');
    await expect(drawer).toContainText('Safe before / after changes');
    if (role === 'manager') await expect(drawer).not.toContainText('amount_minor');
    else await expect(drawer).toContainText('amount_minor');
    await page.getByRole('button', { name: 'Close audit event details' }).click();
    await page.getByLabel('Project ID', { exact: true }).fill('project-b');
    await page.getByRole('button', { name: 'Search events', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'No audit events found' })).toBeVisible();
  });
}
