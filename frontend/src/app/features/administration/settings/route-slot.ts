import { Routes } from '@angular/router';
export const SETTINGS_ROUTES: Routes = [{ path: 'settings', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
