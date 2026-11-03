import { Routes } from '@angular/router';
export const LEADS_ROUTES: Routes = [{ path: 'leads', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'leads/:id', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
