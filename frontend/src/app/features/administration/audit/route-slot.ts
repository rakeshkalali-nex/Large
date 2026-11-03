import { Routes } from '@angular/router';
export const AUDIT_ROUTES: Routes = [{ path: 'audit', loadComponent: () => import('./audit-page').then(m => m.AuditPage) }];
