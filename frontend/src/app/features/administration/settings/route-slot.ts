import { Routes } from '@angular/router';
import { settingsAccess } from './settings-access';
export const SETTINGS_ROUTES: Routes = [
  { path: 'settings', title: 'Company settings', canActivate: [settingsAccess], loadComponent: () => import('./settings-page').then(m => m.SettingsPage) },
  { path: 'settings/access-denied', title: 'Settings access', loadComponent: () => import('./settings-access').then(m => m.SettingsAccessDenied) }
];
