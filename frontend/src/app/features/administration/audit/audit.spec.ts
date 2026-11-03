import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, TestRequest, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AuditApi, AuditEvent } from './audit-api';
import { AuditPage } from './audit-page';
import { AUDIT_ROUTES } from './route-slot';
import { SessionStore } from '../../../core/state/session-store';
import { ProjectContext } from '../../../core/state/project-context';
import { SessionStatus, UiEvents } from '../../../core/state/ui-events';

const event = (id = 'event-1'): AuditEvent => ({
  id, actorId: 'actor-1', actorName: 'Alex', occurredAt: '2026-11-03T12:00:00Z', action: 'settings.updated',
  projectId: null, projectName: null, recordType: 'settings', recordId: 'company', recordLink: '/settings',
  requestId: 'request-1', reason: 'Corrected name', changes: [{ field: 'companyName', before: null, after: '<script>unsafe()</script>' }],
  retention: { class: 'business', eligibleAt: null, holdIds: ['hold-1'] }
});
function flush(request: TestRequest, data: AuditEvent[] = [], nextCursor: string | null = null): void {
  request.flush({ data, page: { nextCursor, limit: 25 }, requestId: 'request-page' });
}
async function setup() {
  const status = signal<SessionStatus>('authenticated');
  await TestBed.configureTestingModule({ imports: [AuditPage], providers: [
    provideHttpClient(), provideHttpClientTesting(), provideRouter([]),
    { provide: SessionStore, useValue: { status } },
    { provide: ProjectContext, useValue: { context: signal({ projectId: null, revision: 0 }) } }
  ] }).compileComponents();
  const fixture = TestBed.createComponent(AuditPage); fixture.detectChanges();
  const page = fixture.componentInstance, http = TestBed.inject(HttpTestingController);
  const dialog = fixture.nativeElement.querySelector('dialog') as HTMLDialogElement;
  // jsdom has no native modal/focus implementation. Browser behavior remains a browser check.
  dialog.showModal = vi.fn(() => { dialog.setAttribute('open', ''); });
  dialog.close = vi.fn(() => { dialog.removeAttribute('open'); });
  return { fixture, page, http, dialog, status, events: TestBed.inject(UiEvents), initial: http.expectOne(r => r.url === '/api/v1/audit-events') };
}
afterEach(() => { TestBed.inject(HttpTestingController).verify({ ignoreCancelled: true }); TestBed.resetTestingModule(); });

describe('Audit history', () => {
  it('routes to the implemented screen and starts a bounded read-only list', async () => {
    const { initial, fixture } = await setup();
    expect(await (AUDIT_ROUTES[0].loadComponent as () => Promise<unknown>)()).toBe(AuditPage);
    expect(initial.request.method).toBe('GET');
    expect(initial.request.params.get('limit')).toBe('25');
    expect(initial.request.params.get('sort')).toBe('occurredAt');
    expect(initial.request.params.get('direction')).toBe('desc');
    expect(fixture.nativeElement.textContent).toContain('Loading audit events');
    flush(initial); fixture.detectChanges(); expect(fixture.nativeElement.textContent).toContain('No audit events found');
  });
  it('sends every filter and explicit UTC dates, preserving applied filters for cursor pages', async () => {
    const { initial, page, http } = await setup(); flush(initial);
    page.form.patchValue({ q: ' approval ', actorId: 'a', action: 'approved', projectId: 'p', recordType: 'order', recordId: 'o', from: '2026-11-01T00:00', to: '2026-11-03T23:59', direction: 'asc', limit: 50 });
    page.apply();
    const request = http.expectOne(r => r.url === '/api/v1/audit-events');
    expect(request.request.params.get('from')).toBe('2026-11-01T00:00:00.000Z');
    expect(request.request.params.get('to')).toBe('2026-11-03T23:59:00.000Z');
    for (const [key, value] of Object.entries({ q: 'approval', actorId: 'a', action: 'approved', projectId: 'p', recordType: 'order', recordId: 'o', direction: 'asc', limit: '50' })) expect(request.request.params.get(key)).toBe(value);
    flush(request, [event()], 'opaque+cursor'); page.form.controls.q.setValue('unapplied'); page.next();
    const next = http.expectOne(r => r.url === '/api/v1/audit-events');
    expect(next.request.params.get('cursor')).toBe('opaque+cursor'); expect(next.request.params.get('q')).toBe('approval');
    next.flush({}, { status: 503, statusText: 'Unavailable' }); page.load();
    const retry = http.expectOne(r => r.url === '/api/v1/audit-events'); expect(retry.request.params.get('cursor')).toBe('opaque+cursor'); flush(retry);
    page.apply(); const reset = http.expectOne(r => r.url === '/api/v1/audit-events'); expect(reset.request.params.has('cursor')).toBe(false); flush(reset);
  });
  it('validates dates and limits before requests', async () => {
    const { initial, page, http } = await setup(); flush(initial);
    page.form.patchValue({ from: '2026-11-03T00:00', to: '2026-11-01T00:00' }); page.apply(); expect(page.validation()).toContain('From no later');
    page.form.patchValue({ from: '', to: '', limit: 101 }); page.apply(); expect(page.validation()).toContain('1 to 100');
    http.expectNone(r => r.url === '/api/v1/audit-events');
  });
  it('shows safe denied feedback rather than rendering server internals', async () => {
    const { initial, fixture, page } = await setup();
    initial.flush({ error: { message: 'SECRET_INTERNAL_ERROR' } }, { status: 403, statusText: 'Forbidden' }); fixture.detectChanges();
    expect(page.denied()).toBe(true); expect(fixture.nativeElement.textContent).toContain('Access denied');
    expect(fixture.nativeElement.textContent).not.toContain('SECRET_INTERNAL_ERROR'); expect(fixture.nativeElement.textContent).not.toContain('Retry audit history');
  });
  it('renders safe detailed changes, retention, record links and restores focus on close', async () => {
    const { initial, fixture, page, http, dialog } = await setup(); flush(initial, [event()]); fixture.detectChanges();
    const opener = fixture.nativeElement.querySelector('.event-row button') as HTMLButtonElement;
    const focus = vi.spyOn(opener, 'focus'); opener.click();
    expect(dialog.showModal).toHaveBeenCalled(); expect(dialog.getAttribute('aria-labelledby')).toBe('detail-title');
    http.expectOne('/api/v1/audit-events/event-1').flush({ data: event(), requestId: 'detail-request' }); fixture.detectChanges();
    expect(dialog.textContent).toContain('<script>unsafe()</script>'); expect(dialog.querySelector('script')).toBeNull();
    expect(dialog.textContent).toContain('Not set'); expect(dialog.textContent).toContain('hold-1'); expect(dialog.textContent).toContain('request-1');
    expect(dialog.querySelector('a')?.getAttribute('href')).toBe('/settings');
    page.closeDetails(); expect(focus).toHaveBeenCalled(); expect(page.selected()).toBeNull();
    for (const link of ['https://outside.test', '//outside.test', '/\\outside.test', 'javascript:alert(1)']) expect(page.safeLink(link)).toBeNull();
  });
  it('handles detail not-found and deliberate retry without mutation calls', async () => {
    const { initial, page, http, fixture } = await setup(); flush(initial, [event()]); fixture.detectChanges();
    page.openDetails(event(), fixture.nativeElement.querySelector('.event-row button'));
    http.expectOne('/api/v1/audit-events/event-1').flush({}, { status: 503, statusText: 'Unavailable' }); expect(page.detailRetry()).toBe(true);
    page.loadDetail(); http.expectOne('/api/v1/audit-events/event-1').flush({}, { status: 404, statusText: 'Not found' });
    expect(page.detailError()).toContain('not found'); expect(page.detailRetry()).toBe(false);
  });
  it('cancels in-flight reads and clears evidence on project and session changes', async () => {
    const { initial, page, http, events, status, fixture } = await setup();
    events.emit({ name: 'project.changed', payload: { projectId: 'p2', revision: 2 } }); expect(initial.cancelled).toBe(true);
    const changed = http.expectOne(r => r.url === '/api/v1/audit-events'); expect(changed.request.params.get('projectId')).toBe('p2');
    flush(changed, [event()], 'next'); fixture.detectChanges();
    page.openDetails(event(), fixture.nativeElement.querySelector('.event-row button')); const detail = http.expectOne('/api/v1/audit-events/event-1');
    status.set('anonymous'); events.emit({ name: 'session.changed', payload: { status: 'anonymous', userId: null, accessRevision: null } });
    expect(detail.cancelled).toBe(true); expect(page.events()).toEqual([]); expect(page.nextCursor()).toBeNull(); expect(page.selected()).toBeNull();
    expect(page.error()).toContain('session has ended'); page.load(); http.expectNone(r => r.url === '/api/v1/audit-events');
  });
  it('encodes opaque event identifiers through the typed client', async () => {
    const { initial, http } = await setup(); flush(initial);
    TestBed.inject(AuditApi).detail('event/with space').subscribe();
    const request = http.expectOne('/api/v1/audit-events/event%2Fwith%20space'); expect(request.request.method).toBe('GET'); request.flush({ data: event(), requestId: 'r' });
  });
});
