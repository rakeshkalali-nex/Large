import { Routes } from '@angular/router';
export const AUDIT_ROUTES: Routes = [{ path: 'audit', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
