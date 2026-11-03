import { spawn } from 'node:child_process';
import { mkdir, mkdtemp, rm } from 'node:fs/promises';
import { resolve, join } from 'node:path';
const root = resolve('.');
await mkdir(join(root, 'var/integration'), { recursive: true });
const temporary = await mkdtemp(join(root, 'var/integration/run-'));
const child = spawn(join(root, 'var/bin/server' + (process.platform === 'win32' ? '.exe' : '')), [], {
  cwd: root, stdio: 'inherit', env: {
    ...process.env, APP_ROOT: temporary, APP_MODE: 'test', PORT: process.env.INTEGRATION_PORT,
    DATA_DIR: join(temporary, 'data'), SQLITE_PATH: join(temporary, 'data/test.sqlite'),
    ATTACHMENTS_DIR: join(temporary, 'data/attachments'), BACKUP_DIR: join(temporary, 'backups'),
    STATIC_DIR: join(root, 'frontend/dist/buildestate/browser'), COOKIE_SECURE: 'false'
  }
});
let stopping = false;
async function stop() {
  if (stopping) return; stopping = true;
  if (child.exitCode === null && child.pid) {
    if (process.platform === 'win32') await new Promise(resolve => {
      const killer = spawn('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
      killer.once('exit', resolve); killer.once('error', resolve);
    }); else child.kill('SIGTERM');
  }
}
process.on('SIGTERM', stop); process.on('SIGINT', stop);
try {
  const code = await new Promise((resolve, reject) => { child.once('error', reject); child.once('exit', resolve); });
  if (!stopping && code !== 0) process.exitCode = 1;
} finally { await rm(temporary, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 }); }
