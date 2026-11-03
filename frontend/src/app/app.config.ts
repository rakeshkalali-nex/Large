import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { APP_ROUTES } from './app.routes';
export const appConfig: ApplicationConfig = { providers: [provideZoneChangeDetection(), provideHttpClient(), provideRouter(APP_ROUTES)] };
