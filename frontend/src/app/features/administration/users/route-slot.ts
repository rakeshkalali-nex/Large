import { Routes } from '@angular/router';
export const USERS_ROUTES: Routes = [{ path: 'users', loadComponent: () => import('./users-page').then(m => m.UsersPage) }];
