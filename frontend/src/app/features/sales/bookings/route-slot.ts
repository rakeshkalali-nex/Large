import { Routes } from '@angular/router';
export const BOOKINGS_ROUTES: Routes = [{ path: 'bookings', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) },{ path: 'bookings/:id', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
