import { Component, inject } from '@angular/core';
import { CanActivateFn, Router, RouterLink } from '@angular/router';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { SessionStore } from '../../../core/state/session-store';
import type { components } from '../../../core/contracts/generated/api';

/** Restore server identity before this feature renders any protected content. */
export const settingsAccess: CanActivateFn = async () => {
  const http = inject(HttpClient), session = inject(SessionStore), router = inject(Router);
  try {
    const response = await firstValueFrom(http.get<{data: components['schemas']['Identity']}>('/api/v1/auth/me'));
    session.accept(response.data);
    return response.data.user.role === 'administrator' && !response.data.setupRequired
      ? true : router.createUrlTree(['/settings/access-denied']);
  } catch (error) {
    if (error instanceof HttpErrorResponse && error.status === 401) { session.clear(); return router.createUrlTree(['/login'], {queryParams: {returnUrl: '/settings'}}); }
    return router.createUrlTree(['/settings/access-denied']);
  }
};

@Component({standalone: true, imports: [RouterLink], template: '<h1>Company settings unavailable</h1><p>Administrator access could not be confirmed. No company data has been loaded.</p><a routerLink="/settings">Retry access check</a> · <a routerLink="/login">Return to sign in</a>'})
export class SettingsAccessDenied {}
