import { Routes } from '@angular/router';
import { DIRECTORY_ROUTES } from './directory/route-slot';
import { WORKORDERS_ROUTES } from './workorders/route-slot';
export const CONTRACTORS_ROUTES: Routes = [...DIRECTORY_ROUTES, ...WORKORDERS_ROUTES];
