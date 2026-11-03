import { Routes } from '@angular/router';
export const BALANCES_ROUTES: Routes = [{ path: 'inventory', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
