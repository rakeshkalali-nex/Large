import { Routes } from '@angular/router';
export const DIRECTORY_ROUTES: Routes = [{ path: 'contractors', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'contractors/:id', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
