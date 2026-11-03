import { Routes } from '@angular/router';
export const PROPERTIES_ROUTES: Routes = [{ path: 'properties', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'properties/:id', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
