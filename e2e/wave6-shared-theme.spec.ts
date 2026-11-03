import { test, expect } from '@playwright/test';

test('shared theme and safe pre-authentication entry', async ({ page }) => {
  await page.goto('/');
  const heading = page.getByRole('heading', { name: 'BuildEstate' });
  await expect(heading).toBeVisible();
  await expect(page.getByRole('status')).toHaveText('This workspace is not available yet. No business operation has been performed.');
  await expect(heading).toBeFocused();
  await expect(page.getByRole('navigation')).toHaveCount(0);
  await expect(page.locator('body')).toHaveCSS('font-family', '"IBM Plex Sans", system-ui, sans-serif');
  await expect(page.locator('body')).toHaveCSS('font-size', '16px');
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(245, 246, 243)');
  await expect(page.locator('body')).toHaveCSS('color', 'rgb(36, 52, 71)');
  const fonts = await page.evaluate(async () => {
    const result: Record<string, boolean> = {};
    for (const weight of [400, 500, 600, 700]) {
      const loaded = await document.fonts.load(`${weight} 16px "IBM Plex Sans"`, 'BuildEstate');
      result[weight] = loaded.length > 0 && loaded.every(face => face.status === 'loaded' && face.display === 'swap');
    }
    return result;
  });
  expect(fonts).toEqual({ 400: true, 500: true, 600: true, 700: true });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  for (const width of [1440, 1024, 768, 390]) {
    await page.setViewportSize({ width, height: 839 });
    await expect(heading).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  }
  await expect(heading).toHaveCSS('animation-duration', '0s');
  await page.reload();
  await expect(heading).toBeVisible();
  await expect(page.getByRole('navigation')).toHaveCount(0);
});
