import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideHttpClient, withInterceptors, withXsrfConfiguration } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { APP_ROUTES } from './app.routes';
import { safeErrorInterceptor } from './core/http/safe-error';
import { SHELL_LOGOUT } from './core/shell/app-shell';
import { SessionService } from './core/session/session.service';
import { expiryInterceptor } from './core/session/expiry.interceptor';
export const appConfig: ApplicationConfig = { providers: [provideZoneChangeDetection(), provideHttpClient(withXsrfConfiguration({ cookieName: 'buildestate_csrf', headerName: 'X-CSRF-Token' }), withInterceptors([safeErrorInterceptor, expiryInterceptor])), provideRouter(APP_ROUTES), { provide: SHELL_LOGOUT, useFactory: (session: SessionService) => () => session.logout(), deps: [SessionService] }] };
