import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { APP_ROUTES } from './app.routes';
import { safeErrorInterceptor } from './core/http/safe-error';
export const appConfig: ApplicationConfig = { providers: [provideZoneChangeDetection(), provideHttpClient(withInterceptors([safeErrorInterceptor])), provideRouter(APP_ROUTES)] };
