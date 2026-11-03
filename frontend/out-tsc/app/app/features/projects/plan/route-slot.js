export const PLAN_ROUTES = [{ path: 'projects/:id/plan', loadComponent: () => import('../../../setup/unavailable').then(m => m.UnavailablePage) }];
