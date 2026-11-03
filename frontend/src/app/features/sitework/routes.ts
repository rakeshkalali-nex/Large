import { Routes } from '@angular/router';
import { LOGS_ROUTES } from './logs/route-slot';
import { INSPECTIONS_ROUTES } from './inspections/route-slot';
export const SITEWORK_ROUTES: Routes = [...LOGS_ROUTES, ...INSPECTIONS_ROUTES];
