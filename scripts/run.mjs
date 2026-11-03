import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { mkdir, writeFile, cp } from 'node:fs/promises';
import { resolve, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer } from 'node:net';
const root = fileURLToPath(new URL('../', import.meta.url));
const children = new Set();
if (existsSync(join(root,'.env'))) process.loadEnvFile(join(root,'.env'));
process.env.APP_ROOT = root;
const executable = name => join(root,'var','bin',name+(process.platform==='win32'?'.exe':''));
export function launch(command,args=[],cwd=root,env={}) {
 const child = spawn(command,args,{cwd,env:{...process.env,...env},stdio:'inherit',shell:false});
 children.add(child);
 child.done = new Promise((res,rej)=>{child.once('error',()=>{children.delete(child);rej(new Error(`${command === 'go' ? 'Go is required on PATH' : 'Child process could not start'}.`));});child.once('exit',(code,signal)=>{children.delete(child);code===0?res():rej(new Error(`Child exited ${code ?? signal}.`));});});
 child.done.catch(()=>{}); return child;
}
async function run(command,args,cwd,env) { return launch(command,args,cwd,env).done; }
const node = (path,args=[],cwd=root,env) => run(process.execPath,[join(root,path),...args],cwd,env);
const go = args => run('go',args,join(root,'backend'));
async function stop() { for(const child of children) { if(process.platform==='win32') await new Promise(r=>{const p=spawn('taskkill',['/pid',String(child.pid),'/T','/F'],{stdio:'ignore'});p.on('error',r);p.on('exit',r);}); else child.kill('SIGTERM'); } }
for(const signal of ['SIGINT','SIGTERM']) process.on(signal,()=>{stop().finally(()=>process.exit(130));});
export async function freePort() { const server=createServer();await new Promise((res,rej)=>{server.once('error',rej);server.listen(0,'127.0.0.1',res);});const port=server.address().port;await new Promise(r=>server.close(r));return port; }
async function ready(port,child) { const end=Date.now()+60000;while(Date.now()<end){if(child.exitCode!==null) throw new Error('Server exited before readiness.');try{const r=await fetch(`http://127.0.0.1:${port}/api/v1/health/ready`,{signal:AbortSignal.timeout(1000)});if(r.ok)return;}catch{}await new Promise(r=>setTimeout(r,200));}throw new Error('Server readiness timed out.'); }
async function buildBackend() { await mkdir(join(root,'var/bin'),{recursive:true});await go(['build','-o',executable('server'),'./cmd/server']);await go(['build','-o',executable('manage'),'./cmd/manage']); }
async function main(command,args) {
 switch(command) {
 case 'install:backend': return go(['mod','download']);
 case 'test:backend': return go(['test',...(args.length?args:['./...'])]);
 case 'migrate': case 'bootstrap': case 'seed:development': case 'backup': case 'restore:check': return go(['run','./cmd/manage',command]);
 case 'build':
  await node('node_modules/@angular/compiler-cli/bundles/src/bin/ngc.js',['-p','tsconfig.app.json'],join(root,'frontend'));
  await node('node_modules/@angular/cli/bin/ng.js',['build'],join(root,'frontend'));
  if(process.env.STATIC_DIR && resolve(root,process.env.STATIC_DIR)!==join(root,'frontend/dist/buildestate/browser')) await cp(join(root,'frontend/dist/buildestate/browser'),resolve(root,process.env.STATIC_DIR),{recursive:true});
  return buildBackend();
 case 'start': return run(executable('server'),[]);
 case 'dev': {
  const port=process.env.PORT?Number(process.env.PORT):await freePort();if(!Number.isInteger(port)||port<1||port>65535)throw new Error('PORT must be between 1 and 65535.');
  let backendPort=await freePort();while(backendPort===port)backendPort=await freePort();
  await buildBackend(); await mkdir(join(root,'var'),{recursive:true});
  const proxy=join(root,'var/proxy.json');await writeFile(proxy,JSON.stringify({'/api/v1':{target:`http://127.0.0.1:${backendPort}`,secure:false}}));
  const backend=launch(executable('server'),[],root,{PORT:String(backendPort)});await ready(backendPort,backend);
  const frontend=launch(process.execPath,[join(root,'node_modules/@angular/cli/bin/ng.js'),'serve','--host','127.0.0.1','--port',String(port),'--proxy-config',proxy],join(root,'frontend'));
  await ready(port,frontend);console.log(`BuildEstate: http://127.0.0.1:${port}`);await Promise.race([backend.done,frontend.done]);return;
 }
 case 'test':
  await node('node_modules/vitest/vitest.mjs',['run'],join(root,'frontend'));
  await go(['test','./...']);await node('scripts/contracts.mjs');return main('test:integration',[]);
 case 'test:integration':
  await main('build',[]);
  await node('node_modules/@playwright/test/cli.js',['install','chromium']);
  return node('node_modules/@playwright/test/cli.js',['test','--config','tests/integration/playwright.config.ts',...args]);
 case 'test:capacity':
  if(!existsSync(join(root,'tests/capacity/run.mjs')))throw new Error('Capacity campaign is unavailable until its owning task is implemented.');
  return node('tests/capacity/run.mjs',args);
 default: throw new Error('Unknown local command.');
 }
}
if(process.argv[1] && resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
 try { await main(process.argv[2],process.argv.slice(3)); } catch(error) { console.error(error.message);process.exitCode=1; } finally { await stop(); }
}
