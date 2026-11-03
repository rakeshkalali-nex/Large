export const APP_ROUTES = [
    { path: '', pathMatch: 'full', redirectTo: 'login' },
    { path: '', loadChildren: () => import('./features/auth/routes').then(m => m.AUTH_ROUTES) },
    { path: '', loadChildren: () => import('./features/projects/routes').then(m => m.PROJECTS_ROUTES) },
    { path: '', loadChildren: () => import('./features/sitework/routes').then(m => m.SITEWORK_ROUTES) },
    { path: '', loadChildren: () => import('./features/procurement/routes').then(m => m.PROCUREMENT_ROUTES) },
    { path: '', loadChildren: () => import('./features/contractors/routes').then(m => m.CONTRACTORS_ROUTES) },
    { path: '', loadChildren: () => import('./features/finance/routes').then(m => m.FINANCE_ROUTES) },
    { path: '', loadChildren: () => import('./features/sales/routes').then(m => m.SALES_ROUTES) },
    { path: '', loadChildren: () => import('./features/documents/routes').then(m => m.DOCUMENTS_ROUTES) },
    { path: '', loadChildren: () => import('./features/administration/routes').then(m => m.ADMINISTRATION_ROUTES) },
    { path: '', loadChildren: () => import('./features/reporting/routes').then(m => m.REPORTING_ROUTES) },
    { path: '**', loadComponent: () => import('./setup/unavailable').then(m => m.UnavailablePage) }
];
