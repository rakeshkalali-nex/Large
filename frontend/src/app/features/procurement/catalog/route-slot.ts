import { Routes } from '@angular/router';
export const CATALOG_ROUTES: Routes = [{ path: 'materials', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
