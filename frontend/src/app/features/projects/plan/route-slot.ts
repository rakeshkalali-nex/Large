import { Routes } from '@angular/router';
export const PLAN_ROUTES: Routes = [{ path: 'projects/:id/plan', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
