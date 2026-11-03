import { Routes } from '@angular/router';
export const MOVEMENTS_ROUTES: Routes = [{ path: 'stock-movements', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
