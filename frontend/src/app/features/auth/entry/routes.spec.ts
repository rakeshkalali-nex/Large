import { describe, expect, it } from 'vitest';
import { loginGuard, rootEntryGuard } from '../../../core/guards/entry.guards';
import { AUTH_ROUTES } from '../routes';
import { AccountPage } from './account-page';
import { ENTRY_ROUTES } from './route-slot';

describe('auth entry route contract', () => {
  it('uses the root resolver only for the exact root and preserves the supplied login guard', () => {
    expect(ENTRY_ROUTES.find(route => route.path === '')).toEqual({
      path: '', pathMatch: 'full', canActivate: [rootEntryGuard], component: AccountPage,
    });
    expect(ENTRY_ROUTES.find(route => route.path === 'login')).toEqual({
      path: 'login', canActivate: [loginGuard], component: AccountPage,
    });
    expect(AUTH_ROUTES).toEqual(ENTRY_ROUTES);
  });
  it('leaves access-denied routing to the protected route owner', () => {
    expect(ENTRY_ROUTES.find(route => route.path === 'access-denied')).toBeUndefined();
  });
});
