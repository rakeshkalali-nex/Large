import { test, expect } from '../../tests/integration/auth-entry/fixtures';

test('company settings initial setup visual baseline',async ({page,accountServer})=>{
  await page.goto('/login?returnUrl=%2Fsettings');
  await page.getByLabel('Email address').fill(accountServer.email);
  await page.getByLabel('Password',{exact:true}).fill(accountServer.password);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page.getByLabel('Company name')).toHaveValue('Smoke Company');
  await page.reload();
  await expect(page.getByRole('heading',{name:'Company settings',exact:true})).toBeVisible();
  await expect(page.getByLabel('Company name')).toHaveValue('Smoke Company');
  await page.evaluate(()=>document.fonts.ready);
  await expect(page).toHaveScreenshot('company-settings.png',{animations:'disabled',mask:[page.locator('dd').first()]});
});
