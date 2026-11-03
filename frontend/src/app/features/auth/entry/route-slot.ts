import { Routes } from '@angular/router';
export const ENTRY_ROUTES: Routes = [{ path: 'login', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
