export const LIBRARY_ROUTES = [{ path: 'documents', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
