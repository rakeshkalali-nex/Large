import { Routes } from '@angular/router';
export const INSPECTIONS_ROUTES: Routes = [{ path: 'inspections', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'issues', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
