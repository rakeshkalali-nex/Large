import { Routes } from '@angular/router';
export const BUDGETS_ROUTES: Routes = [{ path: 'budgets', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'expenses', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
