import { USERS_ROUTES } from './users/route-slot';
import { SETTINGS_ROUTES } from './settings/route-slot';
import { AUDIT_ROUTES } from './audit/route-slot';
export const ADMINISTRATION_ROUTES = [...USERS_ROUTES, ...SETTINGS_ROUTES, ...AUDIT_ROUTES];
