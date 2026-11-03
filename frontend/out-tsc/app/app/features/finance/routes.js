import { BUDGETS_ROUTES } from './budgets/route-slot';
import { INVOICES_ROUTES } from './invoices/route-slot';
import { PAYMENTS_ROUTES } from './payments/route-slot';
export const FINANCE_ROUTES = [...BUDGETS_ROUTES, ...INVOICES_ROUTES, ...PAYMENTS_ROUTES];
