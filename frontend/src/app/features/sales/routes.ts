import { Routes } from '@angular/router';
import { PROPERTIES_ROUTES } from './properties/route-slot';
import { LEADS_ROUTES } from './leads/route-slot';
import { BOOKINGS_ROUTES } from './bookings/route-slot';
import { SALE_ROUTES } from './sale/route-slot';
export const SALES_ROUTES: Routes = [...PROPERTIES_ROUTES, ...LEADS_ROUTES, ...BOOKINGS_ROUTES, ...SALE_ROUTES];
