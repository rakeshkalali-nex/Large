import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { SessionStore } from '../state/session-store';

/** A failed domain request is never replayed after reauthentication. */
export const expiryInterceptor: HttpInterceptorFn = (request, next) => {
  const store = inject(SessionStore); const router = inject(Router);
  return next(request).pipe(catchError((error: unknown) => {
    if (error instanceof HttpErrorResponse && error.status === 401 && request.url.startsWith('/api/v1/') && !request.url.startsWith('/api/v1/auth/')) {
      store.clear();
      void router.navigate(['/login'], { queryParams: { state: 'expired', returnUrl: router.url } });
    }
    return throwError(() => error);
  }));
};
