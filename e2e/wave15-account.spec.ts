import { test, expect } from '../tests/integration/auth-entry/fixtures';
import { randomUUID } from 'node:crypto';

test('staff signs in with Enter, restores identity and follows authorized return routes', async ({ page, accountServer }) => {
  test.setTimeout(90000);
  await page.goto('/');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', {name:'Welcome back'})).toBeVisible();
  const email = page.getByLabel('Email address', {exact:true});
  const password = page.getByLabel('Password', {exact:true});
  await expect(email).toHaveAttribute('type','email');
  await expect(email).toHaveAttribute('autocomplete','username');
  await expect(password).toHaveAttribute('autocomplete','current-password');
  await expect(password).toHaveAttribute('type','password');
  await email.fill(process.env.E2E_ADMIN_EMAIL ?? accountServer.email);
  await password.fill(process.env.E2E_ADMIN_PASSWORD ?? accountServer.password);
  await password.press('Tab');
  await expect(page.getByRole('button',{name:'Show password',exact:true})).toBeFocused();
  await page.keyboard.press('Space');
  await expect(password).toHaveAttribute('type','text');
  await expect(password).toHaveValue(process.env.E2E_ADMIN_PASSWORD ?? accountServer.password);
  await page.keyboard.press('Space');
  const response = page.waitForResponse(r=>r.url().endsWith('/auth/login') && r.request().method()==='POST');
  await password.press('Enter');
  expect((await response).status()).toBe(200);
  await expect(page).toHaveURL(/\/dashboard$/, {timeout:15000});
  await page.goto('/');
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.goto('/login?returnUrl=%2Fsettings');
  await expect(page).toHaveURL(/\/settings$/);
  await expect(page.getByRole('heading',{name:'Company settings',exact:true})).toBeVisible();
  await page.goto('/login?returnUrl=https%3A%2F%2Fexample.invalid');
  await expect(page).toHaveURL(/\/dashboard$/);
});

test('invalid credentials and service failure retain email and clear secrets without retry', async ({page,accountServer})=>{
  await page.goto('/login');
  await page.getByLabel('Email address').fill(accountServer.email);
  await page.getByLabel('Password',{exact:true}).fill(randomUUID());
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page.getByRole('alert')).toHaveText('Email or password is incorrect.');
  await expect(page.getByLabel('Email address')).toHaveValue(accountServer.email);
  await expect(page.getByLabel('Password',{exact:true})).toHaveValue('');
  let calls=0;
  await page.route('**/api/v1/auth/login',async route=>{calls++;await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:{code:'SERVICE_UNAVAILABLE',message:'Service unavailable'},requestId:'fault'})});});
  await page.getByLabel('Password',{exact:true}).fill(accountServer.password);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page.getByRole('alert')).toContainText('We could not reach the service');
  await expect(page.getByRole('alert')).toBeFocused();
  await expect(page.getByLabel('Email address')).toHaveValue(accountServer.email);
  await expect(page.getByLabel('Password',{exact:true})).toHaveValue('');
  await expect(page.getByRole('button',{name:'Sign in',exact:true})).toBeEnabled();
  expect(calls).toBe(1);
  await page.unroute('**/api/v1/auth/login');
});

test('staff recovers account and sees safe setup expiry and invalid link guidance', async ({page})=>{
  await page.goto('/login');
  await page.getByRole('button',{name:'Forgot password?',exact:true}).click();
  await expect(page.getByRole('heading',{name:'Recover your account'})).toBeVisible();
  await expect(page.getByText('Smoke account administrator',{exact:true})).toBeVisible();
  await expect(page.getByText(/identity verification through a trusted private channel/)).toBeVisible();
  for (const [state,title] of [['setup','Set up your password'],['expired','Your session has expired'],['reset','This link is no longer available']]) {
    await page.goto('/login?state='+state);
    await expect(page.getByRole('heading',{name:title,exact:true})).toBeVisible();
    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  }
});

test('authenticated staff changes password then signs in with replacement',async ({page,accountServer})=>{
  await page.goto('/login');
  await page.getByLabel('Email address').fill(accountServer.email);
  await page.getByLabel('Password',{exact:true}).fill(accountServer.password);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.goto('/login?state=change');
  await expect(page.getByRole('heading',{name:'Change your password'})).toBeVisible();
  const replacement=randomUUID();
  await page.getByLabel('Current password',{exact:true}).fill(accountServer.password);
  await page.getByLabel('New password',{exact:true}).fill(replacement);
  await page.getByLabel('Confirm new password',{exact:true}).fill(replacement);
  await page.getByRole('button',{name:'Save password',exact:true}).click();
  await expect(page.getByRole('heading',{name:'Password updated'})).toBeVisible();
  await page.getByRole('button',{name:'Back to sign in',exact:true}).click();
  await page.getByLabel('Email address').fill(accountServer.email);
  await page.getByLabel('Password',{exact:true}).fill(replacement);
  await page.getByRole('button',{name:'Sign in',exact:true}).click();
  await expect(page).toHaveURL(/\/dashboard$/);
});
