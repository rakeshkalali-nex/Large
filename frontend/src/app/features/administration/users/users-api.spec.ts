import { TestBed } from '@angular/core/testing';
import { MOCK_PLATFORM_LOCATION_CONFIG } from '@angular/common/testing';
import { HttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { appConfig } from '../../../app.config';
import { User, UsersApi } from './users-api';

const user: User = { id: 'staff-1', displayName: 'Staff', email: 'staff@example.test', role: 'site-engineer', status: 'setup-required', version: 3, projectIds: ['p1'] };
describe('UsersApi contract', () => {
  let api: UsersApi, http: HttpTestingController;
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [
      ...appConfig.providers,
      // Angular's test location otherwise has no origin, so XSRF correctly skips it.
      { provide: MOCK_PLATFORM_LOCATION_CONFIG, useValue: { startUrl: window.location.href } },
      provideHttpClientTesting()
    ] });
    document.cookie = 'buildestate_csrf=users-test-nonce; path=/';
    api = TestBed.inject(UsersApi); http = TestBed.inject(HttpTestingController);
  });
  afterEach(() => {
    document.cookie = 'buildestate_csrf=; Max-Age=0; path=/';
    http.verify();
  });
  it('unwraps searchable users, roles and projects item envelopes', async () => {
    const users = api.list('a+b@example.test');
    const request = http.expectOne(r => r.url === '/api/v1/users');
    expect(request.request.params.get('search')).toBe('a+b@example.test'); request.flush({ data: { items: [user] } });
    expect(await users).toEqual([user]);
    const roles = api.roles(); http.expectOne('/api/v1/roles').flush({ data: { items: [{ code: 'administrator', name: 'Administrator' }] } });
    expect(await roles).toHaveLength(1);
    const projects = api.projects(); http.expectOne('/api/v1/users/projects').flush({ data: { items: [{ id: 'p1', name: 'Project' }] } });
    expect(await projects).toHaveLength(1); http.verify();
  });
  it('creates without automatic issuance and edits assignments with a version', async () => {
    const input = { displayName: user.displayName, email: user.email, role: user.role, projectIds: user.projectIds };
    const key = crypto.randomUUID();
    const create = api.save(input, null, key); const request = http.expectOne('/api/v1/users');
    expect(request.request.method).toBe('POST'); expect(request.request.body).toEqual(input);
    expect(request.request.headers.get('X-CSRF-Token')).toBe('users-test-nonce');
    expect(request.request.headers.get('Idempotency-Key')).toBe(key); request.flush({ data: user }); expect(await create).toEqual(user);
    document.cookie = 'buildestate_csrf=rotated-users-nonce; path=/';
    const edit = api.save(input, user, key); const update = http.expectOne('/api/v1/users/staff-1');
    expect(update.request.headers.get('X-CSRF-Token')).toBe('rotated-users-nonce');
    expect(update.request.method).toBe('PUT'); expect(update.request.body).toEqual({ ...input, version: 3 });
    expect(update.request.headers.get('Idempotency-Key')).toBe(key); update.flush({ data: user }); await edit; http.verify();
  });
  it.each(['deactivate', 'reactivate'] as const)('sends reason and version for %s', async action => {
    const key = crypto.randomUUID(), result = api.transition(user, action, 'Approved change', key);
    const request = http.expectOne(`/api/v1/users/staff-1/${action}`);
    expect(request.request.headers.get('X-CSRF-Token')).toBe('users-test-nonce');
    expect(request.request.body).toEqual({ version: 3, reason: 'Approved change' });
    expect(request.request.headers.get('Idempotency-Key')).toBe(key); request.flush({ data: user }); await result; http.verify();
  });
  it.each(['setup', 'recovery'] as const)('uses the existing %s issuance contract', async purpose => {
    const key = crypto.randomUUID(), result = api.issue(user, purpose, 'Verified in person', key);
    const request = http.expectOne('/api/v1/users/staff-1/password-reset');
    expect(request.request.headers.get('X-CSRF-Token')).toBe('users-test-nonce');
    expect(request.request.body).toEqual({ version: 3, purpose, reason: 'Verified in person', identityVerified: true });
    expect(request.request.headers.get('Idempotency-Key')).toBe(key);
    const link = { userId: user.id, resetLink: '/login#token=private', expiresAt: '2026-11-04T12:00:00Z' };
    request.flush({ data: link }); expect(await result).toEqual(link); http.verify();
  });
  it('does not send the CSRF nonce on reads or to another origin', async () => {
    const result = api.list('');
    const read = http.expectOne(r => r.url === '/api/v1/users');
    expect(read.request.headers.has('X-CSRF-Token')).toBe(false);
    read.flush({ data: { items: [] } }); await result;
    TestBed.inject(HttpClient).post('https://external.example.test/collect', {}).subscribe();
    const external = http.expectOne('https://external.example.test/collect');
    expect(external.request.headers.has('X-CSRF-Token')).toBe(false);
    external.flush({});
  });
});
