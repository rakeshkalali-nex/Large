import { test as base, expect } from '@playwright/test';
import { spawn, type ChildProcess } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, rm } from 'node:fs/promises';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { randomUUID } from 'node:crypto';

type AccountServer = { origin: string; email: string; password: string };

async function freePort(): Promise<number> {
  const socket = createServer();
  socket.listen(0, '127.0.0.1');
  await once(socket, 'listening');
  const address = socket.address();
  if (!address || typeof address === 'string') throw new Error('Cannot allocate smoke-test port.');
  await new Promise<void>((resolve, reject) => socket.close(error => error ? reject(error) : resolve()));
  return address.port;
}

async function stop(child: ChildProcess): Promise<void> {
  if (!child.pid || child.exitCode !== null || child.signalCode !== null) return;
  const exited = once(child, 'exit');
  if (process.platform === 'win32') {
    const killer = spawn('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
    await once(killer, 'exit');
  } else {
    child.kill('SIGTERM');
    const timer = setTimeout(() => child.kill('SIGKILL'), 5000);
    timer.unref();
    try { await exited; } finally { clearTimeout(timer); }
    return;
  }
  await exited;
}

// Per-test persistence prevents rate limits, sessions or password mutations from
// leaking across parallel cases. Credentials exist only in memory/child stdin.
export const test = base.extend<{ accountServer: AccountServer }>({
  accountServer: [async ({}, use) => {
    const root = resolve('.');
    const temporary = await mkdtemp(join(tmpdir(), 'buildestate-auth-entry-'));
    const port = await freePort();
    const origin = `http://127.0.0.1:${port}`;
    const executable = (name: string) => join(root, 'var/bin', name + (process.platform === 'win32' ? '.exe' : ''));
    const env = {
      ...process.env, APP_ROOT: temporary, APP_MODE: 'test', PORT: String(port),
      DATA_DIR: join(temporary, 'data'), SQLITE_PATH: join(temporary, 'data/test.sqlite'),
      ATTACHMENTS_DIR: join(temporary, 'data/attachments'), BACKUP_DIR: join(temporary, 'backups'),
      STATIC_DIR: join(root, 'frontend/dist/buildestate/browser'), COOKIE_SECURE: 'false',
    };
    const email = `entry-${randomUUID()}@example.test`;
    const password = randomUUID();
    let child: ChildProcess | undefined;
    try {
      const manage = spawn(executable('manage'), ['bootstrap'], { cwd: root, env, stdio: ['pipe', 'ignore', 'ignore'] });
      child = manage;
      const bootstrapped = once(manage, 'exit');
      manage.stdin.on('error', () => {}); // exit/error below reports failure without exposing input
      manage.stdin.end(JSON.stringify({
        displayName: 'Smoke Administrator', email, password, companyName: 'Smoke Company',
        recoveryContactLabel: 'Smoke account administrator',
        recoveryInstructions: 'Request identity verification through a trusted private channel for a replacement link.',
      }));
      const [code] = await bootstrapped;
      if (code !== 0) throw new Error('Smoke account bootstrap failed (build server and manage first).');
      child = spawn(executable('server'), [], { cwd: root, env, stdio: 'ignore' });
      let startupError = false;
      child.on('error', () => { startupError = true; });
      const deadline = Date.now() + 30000;
      let ready = false;
      while (Date.now() < deadline) {
        if (startupError || child.exitCode !== null) throw new Error('Smoke server failed to start.');
        try {
          const response = await fetch(`${origin}/api/v1/health/ready`, { signal: AbortSignal.timeout(1000) });
          if (response.ok) { ready = true; break; }
        } catch { /* wait for the actual Go listener */ }
        await new Promise(resolve => setTimeout(resolve, 100));
      }
      if (!ready) throw new Error('Smoke server readiness timed out.');
      await use({ origin, email, password });
    } finally {
      if (child) await stop(child);
      await rm(temporary, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 });
    }
  }, { timeout: 60000 }],
  baseURL: async ({ accountServer }, use) => use(accountServer.origin),
});

export { expect };
