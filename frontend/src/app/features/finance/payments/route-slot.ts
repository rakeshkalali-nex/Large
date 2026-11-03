import { Routes } from '@angular/router';
export const PAYMENTS_ROUTES: Routes = [{ path: 'payments', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
