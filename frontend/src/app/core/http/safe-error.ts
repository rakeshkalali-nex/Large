import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';
import { SessionStore } from '../state/session-store';

export interface SafeFailure { kind: 'connectivity' | 'validation' | 'conflict' | 'permission' | 'not-found' | 'session' | 'server'; message: string; uncertain: boolean; }
export function safeFailure(error: unknown): SafeFailure {
  const status = error instanceof HttpErrorResponse ? error.status : -1;
  switch (status) {
    case 0: return { kind: 'connectivity', message: 'Connection lost. Your safe values remain on this page. Check the result before deliberately retrying.', uncertain: true };
    case 401: return { kind: 'session', message: 'Your session has ended. Sign in again before continuing.', uncertain: false };
    case 403: return { kind: 'permission', message: 'You do not have permission to perform this action.', uncertain: false };
    case 404: return { kind: 'not-found', message: 'This record is not available.', uncertain: false };
    case 409: return { kind: 'conflict', message: 'The record changed or the action conflicts with its current state. Compare the latest record before submitting again.', uncertain: false };
    case 422: return { kind: 'validation', message: 'Check the highlighted fields before submitting again.', uncertain: false };
    default: return { kind: 'server', message: 'The service could not confirm the result. Your safe values remain on this page. Check before retrying.', uncertain: true };
  }
}
/** Never log response bodies, expose raw server messages, or automatically replay requests. */
export const safeErrorInterceptor: HttpInterceptorFn = (request, next) => {
  const session = inject(SessionStore);
  return next(request).pipe(catchError((error: unknown) => {
    if (error instanceof HttpErrorResponse && error.status === 401 && request.url.startsWith('/api/v1/')) session.clear();
    return throwError(() => error);
  }));
};
