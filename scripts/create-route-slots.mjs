// Foundation scaffolder. Deliberately refuses to overwrite feature implementations.
import { mkdir, writeFile } from 'node:fs/promises';
const groups = {
 auth: { entry: ['login'] }, projects: { portfolio: ['projects','projects/new','projects/:id'], plan: ['projects/:id/plan'] },
 sitework: { logs: ['site-logs','site-logs/new','site-logs/:id'], inspections: ['inspections','issues'] },
 procurement: { catalog: ['materials'], orders: ['purchase-orders','purchase-orders/new','purchase-orders/:id'], receipts: ['goods-receipts'], balances: ['inventory'], movements: ['stock-movements'] },
 contractors: { directory: ['contractors','contractors/:id'], workorders: ['work-orders','work-orders/:id'] },
 finance: { budgets: ['budgets','expenses'], invoices: ['invoices'], payments: ['payments'] },
 sales: { properties: ['properties','properties/:id'], leads: ['leads','leads/:id'], bookings: ['bookings','bookings/:id'], sale: [] },
 documents: { library: ['documents'] }, administration: { users: ['users'], settings: ['settings'], audit: ['audit'] }, reporting: { reports: ['dashboard','reports'] }
};
async function create(path, text) { await mkdir(path.slice(0,path.lastIndexOf('/')), {recursive:true}); await writeFile(path,text,{flag:'wx'}); }
for (const [area, slots] of Object.entries(groups)) {
 let imports = "import { Routes } from '@angular/router';\n";
 for (const [slot,paths] of Object.entries(slots)) {
  const name = slot.toUpperCase()+'_ROUTES';
  await create(`frontend/src/app/features/${area}/${slot}/route-slot.ts`, `import { Routes } from '@angular/router';\nexport const ${name}: Routes = ${JSON.stringify(paths.map(path=>({path}))).replace(/\{"path":"([^"]+)"\}/g,`{ path: '$1', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }`)};\n`);
  imports += `import { ${name} } from './${slot}/route-slot';\n`;
 }
 await create(`frontend/src/app/features/${area}/routes.ts`,imports+`export const ${area.toUpperCase()}_ROUTES: Routes = [${Object.keys(slots).map(s=>'...'+s.toUpperCase()+'_ROUTES').join(', ')}];\n`);
}
await create('frontend/src/app/app.routes.ts', `import { Routes } from '@angular/router';\nexport const APP_ROUTES: Routes = [\n { path: '', pathMatch: 'full', redirectTo: 'login' },\n${Object.keys(groups).map(a=>` { path: '', loadChildren: () => import('./features/${a}/routes').then(m => m.${a.toUpperCase()}_ROUTES) },`).join('\n')}\n { path: '**', loadComponent: () => import('./setup/unavailable').then(m => m.UnavailablePage) }\n];\n`);
