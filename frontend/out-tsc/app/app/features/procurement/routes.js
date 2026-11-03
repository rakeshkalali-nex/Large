import { CATALOG_ROUTES } from './catalog/route-slot';
import { ORDERS_ROUTES } from './orders/route-slot';
import { RECEIPTS_ROUTES } from './receipts/route-slot';
import { BALANCES_ROUTES } from './balances/route-slot';
import { MOVEMENTS_ROUTES } from './movements/route-slot';
export const PROCUREMENT_ROUTES = [...CATALOG_ROUTES, ...ORDERS_ROUTES, ...RECEIPTS_ROUTES, ...BALANCES_ROUTES, ...MOVEMENTS_ROUTES];
