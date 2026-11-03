export const PAYMENTS_ROUTES = [{ path: 'payments', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
