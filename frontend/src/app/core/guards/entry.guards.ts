import { inject } from '@angular/core';
import { CanActivateChildFn, CanActivateFn, Router } from '@angular/router';
import { SessionService } from '../session/session.service';
import { authorizedPath } from '../session/destinations';

export const rootEntryGuard: CanActivateFn = async () => {
  const session = inject(SessionService); const router = inject(Router);
  await session.restore();
  return router.parseUrl(session.restoreFailure() ? '/login?state=service' : session.expired() ? '/login?state=expired' : session.destination(null));
};
export const loginGuard: CanActivateFn = async route => {
  const session = inject(SessionService); const router = inject(Router);
  // Let the account shell scrub a one-time fragment before making any request.
  if (route.fragment) return true;
  await session.restore();
  const state = route.queryParamMap.get('state');
  if (route.fragment || ['recovery', 'reset', 'setup'].includes(state ?? '')) return true;
  if (state === 'change' && session.store.status() === 'authenticated') return true;
  if (session.store.status() === 'setupRequired') return state === 'setup' || router.parseUrl('/login?state=setup');
  if (session.store.status() === 'authenticated') return router.parseUrl(session.destination(route.queryParamMap.get('returnUrl')));
  return true;
};
export const protectedEntryGuard: CanActivateChildFn = async (_route, state) => {
  const session = inject(SessionService); const router = inject(Router);
  await session.restore();
  if (session.store.status() !== 'authenticated') {
    return router.createUrlTree(['/login'], { queryParams: { state: session.restoreFailure() ? 'service' : session.store.status() === 'setupRequired' ? 'setup' : session.expired() ? 'expired' : undefined, returnUrl: state.url } });
  }
  return state.url === '/access-denied' || !!authorizedPath(state.url, session.store.capabilities()) || router.parseUrl('/access-denied');
};
