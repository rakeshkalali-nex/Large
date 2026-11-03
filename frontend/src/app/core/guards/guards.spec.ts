import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, provideRouter, Router, RouterStateSnapshot, UrlTree } from '@angular/router';
import { beforeEach, describe, expect, it } from 'vitest';
import { SessionService } from '../session/session.service';
import { SessionStore } from '../state/session-store';
import { authorizedPath, roleLanding } from '../session/destinations';
import { loginGuard, protectedEntryGuard, rootEntryGuard } from './entry.guards';

describe('guards and authorized return paths', () => {
  beforeEach(() => TestBed.configureTestingModule({ providers: [provideRouter([]), { provide: SessionService, useFactory: () => ({ store: TestBed.inject(SessionStore), restore: async () => {}, restoreFailure: () => false, expired: () => false, destination: () => '/projects' }) }] }));
  it.each(['https://evil.test', '//evil.test', '/\\evil', '/projects/%2f', '/projects/../users', '/projects;admin=true', '/projects(aux:users)', '/login', '/unknown', '/projects/p/unknown', '/projects?x=%ZZ'])('rejects malformed/external or unregistered return path %s', value => expect(authorizedPath(value, ['projects.read'])).toBeNull());
  it('requires exact route capabilities including create and plan', () => {
    expect(authorizedPath('/projects/p?tab=summary', ['projects.read'])).toBe('/projects/p?tab=summary');
    expect(authorizedPath('/projects/new', ['projects.read'])).toBeNull();
    expect(authorizedPath('/projects/p/plan', ['projects.read'])).toBeNull();
    expect(authorizedPath('/projects/p/plan', ['plan.read'])).toBe('/projects/p/plan');
    expect(authorizedPath('/users', ['projects.read'])).toBeNull();
  });
  it.each([['administrator','/dashboard'],['project-manager','/projects'],['site-engineer','/site-logs'],['procurement-officer','/purchase-orders'],['finance-officer','/invoices'],['sales-executive','/leads'],['management-viewer','/dashboard']])('uses the exact landing for %s', (role, landing) => expect(roleLanding(role)).toBe(landing));
  it('awaits restoration before root resolution', async () => {
    const result = await TestBed.runInInjectionContext(() => rootEntryGuard({} as ActivatedRouteSnapshot, {} as RouterStateSnapshot));
    expect(TestBed.inject(Router).serializeUrl(result as UrlTree)).toBe('/projects');
  });
  it('redirects anonymous protected entry without allowing content', async () => {
    TestBed.inject(SessionStore).clear();
    const result = await TestBed.runInInjectionContext(() => protectedEntryGuard({} as ActivatedRouteSnapshot, { url: '/projects/p' } as RouterStateSnapshot));
    expect(TestBed.inject(Router).serializeUrl(result as UrlTree)).toContain('/login?returnUrl=');
  });
  it('allows anonymous sign in after restoration', async () => {
    const route = new ActivatedRouteSnapshot(); TestBed.inject(SessionStore).clear();
    expect(await TestBed.runInInjectionContext(() => loginGuard(route, {} as RouterStateSnapshot))).toBe(true);
  });
  it('lets the account shell scrub fragment credentials before restoration requests', async () => {
    const route = new ActivatedRouteSnapshot(); route.fragment = 'token=private';
    const service = TestBed.inject(SessionService); service.restore = () => { throw new Error('Must not make a request before scrubbing'); };
    expect(await TestBed.runInInjectionContext(() => loginGuard(route, {} as RouterStateSnapshot))).toBe(true);
  });
});
