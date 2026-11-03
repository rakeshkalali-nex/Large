import { Routes } from '@angular/router';
export const LIBRARY_ROUTES: Routes = [{ path: 'documents', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
