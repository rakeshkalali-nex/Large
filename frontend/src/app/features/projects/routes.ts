import { Routes } from '@angular/router';
import { PORTFOLIO_ROUTES } from './portfolio/route-slot';
import { PLAN_ROUTES } from './plan/route-slot';
export const PROJECTS_ROUTES: Routes = [...PORTFOLIO_ROUTES, ...PLAN_ROUTES];
