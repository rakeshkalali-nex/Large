import { defineConfig } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import integration from '../../../../tests/integration/playwright.config';

// Reuse the real-server harness: free port, disposable SQLite, no seeded login.
export default defineConfig({
  ...integration,
  testDir: fileURLToPath(new URL('../../../../e2e/', import.meta.url)),
  testMatch: 'password-recovery.spec.ts',
  projects: [
    { name: 'desktop', use: { viewport: { width: 1280, height: 720 } } },
    { name: 'phone', use: { viewport: { width: 412, height: 839 }, isMobile: true, hasTouch: true } },
  ],
});
