import { Routes } from '@angular/router';
export const REPORTS_ROUTES: Routes = [{ path: 'dashboard', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'reports', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
