import { DIRECTORY_ROUTES } from './directory/route-slot';
import { WORKORDERS_ROUTES } from './workorders/route-slot';
export const CONTRACTORS_ROUTES = [...DIRECTORY_ROUTES, ...WORKORDERS_ROUTES];
