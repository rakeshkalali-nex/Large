import { Routes } from '@angular/router';
export const WORKORDERS_ROUTES: Routes = [{ path: 'work-orders', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'work-orders/:id', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
