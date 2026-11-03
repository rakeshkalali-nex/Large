import { Routes } from '@angular/router';
export const USERS_ROUTES: Routes = [{ path: 'users', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
