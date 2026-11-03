export const USERS_ROUTES = [{ path: 'users', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
