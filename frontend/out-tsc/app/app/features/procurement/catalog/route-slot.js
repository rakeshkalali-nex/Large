export const CATALOG_ROUTES = [{ path: 'materials', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
