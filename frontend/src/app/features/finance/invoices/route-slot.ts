import { Routes } from '@angular/router';
export const INVOICES_ROUTES: Routes = [{ path: 'invoices', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
