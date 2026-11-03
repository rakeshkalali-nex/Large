import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { SessionService } from './session.service';
import type { components } from '../contracts/generated/api';

export const staff: components['schemas']['Identity'] = { user: { id: 'staff', displayName: 'Staff', email: 'staff@example.test', role: 'project-manager' }, accessRevision: 1, capabilities: ['projects.read'], csrfToken: 'nonce', idleExpiresAt: '2026-11-03T18:00:00Z', absoluteExpiresAt: '2026-11-04T01:00:00Z', landingPath: '/projects', projectIds: ['p'], setupRequired: false };
describe('session restoration and credential transport', () => {
  let service: SessionService; let http: HttpTestingController;
  beforeEach(() => { TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])] }); service = TestBed.inject(SessionService); http = TestBed.inject(HttpTestingController); });
  afterEach(() => http.verify());
  it('shares pending restoration and withholds identity until the server responds', async () => {
    const first = service.restore(); const second = service.restore(); expect(first).toBe(second);
    expect(service.store.status()).toBe('restoring');
    http.expectOne('/api/v1/auth/me').flush({ data: staff }); await first;
    expect(service.store.status()).toBe('authenticated'); expect(service.destination('/users')).toBe('/projects');
  });
  it('distinguishes unavailable restoration from anonymous and retries only deliberately', async () => {
    const first = service.restore(); http.expectOne('/api/v1/auth/me').flush({}, { status: 503, statusText: 'Unavailable' }); await first;
    expect(service.restoreFailure()).toBe(true); expect(service.store.user()).toBeNull();
    http.expectNone('/api/v1/auth/me');
    const retry = service.restore(); http.expectOne('/api/v1/auth/me').flush({}, { status: 401, statusText: 'Unauthorized' }); await retry;
    expect(service.restoreFailure()).toBe(false);
  });
  it('sends fresh CSRF and accepts only the server login identity', async () => {
    const login = service.login('staff@example.test', 'private password');
    http.expectOne('/api/v1/auth/csrf').flush({ data: { csrfToken: 'fresh' } }); await Promise.resolve();
    const request = http.expectOne('/api/v1/auth/login'); expect(request.request.headers.get('X-CSRF-Token')).toBe('fresh');
    expect(request.request.body).toEqual({ email: 'staff@example.test', password: 'private password' });
    request.flush({ data: staff }); await login; expect(service.destination('/projects/p')).toBe('/projects/p');
  });
  it('clears identity after successful password completion', async () => {
    service.store.accept(staff); const completion = service.completePassword('one-time', 'new private password');
    http.expectOne('/api/v1/auth/csrf').flush({ data: { csrfToken: 'fresh' } }); await Promise.resolve();
    http.expectOne('/api/v1/auth/reset-password').flush({ data: { passwordChanged: true } }); await completion;
    expect(service.store.status()).toBe('anonymous');
  });
  it('keeps setup-required users out of role destinations', () => {
    service.store.accept({ ...staff, setupRequired: true }); expect(service.destination('/projects')).toBe('/login?state=setup');
  });
});
