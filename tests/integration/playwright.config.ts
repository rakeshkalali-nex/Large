import { defineConfig } from '@playwright/test';
import { createServer } from 'node:net';
// Workers reload this config: inherit the runner's allocation instead of
// allocating another port in every worker process.
if (!process.env.INTEGRATION_PORT) {
  const socket = createServer();
  await new Promise<void>((resolve, reject) => { socket.once('error', reject); socket.listen(0, '127.0.0.1', resolve); });
  const address = socket.address();
  if (!address || typeof address === 'string') throw new Error('Could not allocate integration port.');
  process.env.INTEGRATION_PORT = String(address.port);
  await new Promise<void>(resolve => socket.close(() => resolve()));
}
const port = Number(process.env.INTEGRATION_PORT);
export default defineConfig({
  testDir: '.', testMatch: '**/*.spec.ts', fullyParallel: true,
  use: { baseURL: `http://127.0.0.1:${port}`, trace: 'retain-on-failure' },
  webServer: {
    command: 'node tests/integration/support/server.mjs',
    cwd: process.cwd(), env: { INTEGRATION_PORT: String(port) },
    url: `http://127.0.0.1:${port}/api/v1/health/ready`,
    reuseExistingServer: false, timeout: 60000,
    gracefulShutdown: { signal: 'SIGTERM', timeout: 20000 }
  }
});
