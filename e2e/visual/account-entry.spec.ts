import { test, expect } from '../../tests/integration/auth-entry/fixtures';

test('account entry visual baseline',async ({page})=>{
  await page.goto('/login');
  await expect(page.getByRole('heading',{name:'Welcome back'})).toBeVisible();
  await page.evaluate(()=>document.fonts.ready);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await expect(page).toHaveScreenshot('account-entry.png',{animations:'disabled'});
});
