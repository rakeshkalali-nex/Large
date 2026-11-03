import { Routes } from '@angular/router';
export const RECEIPTS_ROUTES: Routes = [{ path: 'goods-receipts', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
