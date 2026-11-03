import { LOGS_ROUTES } from './logs/route-slot';
import { INSPECTIONS_ROUTES } from './inspections/route-slot';
export const SITEWORK_ROUTES = [...LOGS_ROUTES, ...INSPECTIONS_ROUTES];
