export const AUDIT_ROUTES = [{ path: 'audit', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
