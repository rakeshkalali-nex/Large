import { TestBed } from '@angular/core/testing';
import { describe, expect, it } from 'vitest';
import type { components } from '../contracts/generated/api';
import { SessionStore } from './session-store';
import { ProjectContext, UnsavedChanges } from './project-context';
import { UiEvent, UiEvents } from './ui-events';
import { ScopedQueryCache } from './scoped-query-cache';

export function identity(capabilities = ['projects.read']): components['schemas']['Identity'] {
  return { user: { id: 'staff-1', displayName: 'Staff member', email: 'staff@example.test', role: 'project-manager' }, accessRevision: 1, capabilities, csrfToken: 'test-only', idleExpiresAt: '2026-11-03T10:00:00Z', absoluteExpiresAt: '2026-11-03T20:00:00Z', landingPath: '/projects', projectIds: ['project-1', 'project-2'], setupRequired: false };
}
describe('core transient state', () => {
  it('invalidates scoped reads and rejects late results after session/context changes', () => {
    const cache = TestBed.inject(ScopedQueryCache); const revision = cache.snapshot();
    cache.put('projects', ['authorized'], revision); expect(cache.get('projects')).toEqual(['authorized']);
    TestBed.inject(SessionStore).clear(); cache.put('projects', ['late'], revision);
    expect(cache.get('projects')).toBeUndefined();
  });
  it('withholds navigation until server identity and clears all identity on session loss', () => {
    const store = TestBed.inject(SessionStore);
    expect(store.status()).toBe('restoring'); expect(store.has('projects.read')).toBe(false);
    const response = identity(); store.accept(response); response.capabilities.push('users.manage');
    expect(store.has('projects.read')).toBe(true); expect(store.has('users.manage')).toBe(false);
    store.clear(); expect(store.user()).toBeNull(); expect(store.csrfToken()).toBeNull(); expect(store.projectIds()).toEqual([]);
  });
  it('does not expose capabilities for setup-required identities', () => {
    const store = TestBed.inject(SessionStore); store.accept({ ...identity(), setupRequired: true });
    expect(store.status()).toBe('setupRequired'); expect(store.capabilities()).toEqual([]);
  });
  it('preserves dirty input on refused context change and invalidates on allowed changes', async () => {
    const context = TestBed.inject(ProjectContext); TestBed.inject(SessionStore).accept(identity());
    const events: UiEvent[] = []; TestBed.inject(UiEvents).events.subscribe(event => events.push(event));
    const draft = { notes: 'Safe unsaved site notes' };
    const unregister = TestBed.inject(UnsavedChanges).register(() => false);
    expect(await context.change('project-1')).toBe(false); expect(draft.notes).toBe('Safe unsaved site notes');
    expect(context.context().projectId).toBeNull(); unregister();
    expect(await context.change('forbidden')).toBe(false);
    expect(await context.change('project-1')).toBe(true);
    expect(context.effectiveProject('project-2')).toBe('project-2');
    expect(events.at(-1)?.name).toBe('project.changed');
    TestBed.inject(SessionStore).clear(); expect(context.context().projectId).toBeNull();
  });
  it('cancels an outstanding context confirmation after session replacement', async () => {
    const context = TestBed.inject(ProjectContext); const session = TestBed.inject(SessionStore); session.accept(identity());
    let approve!: (value: boolean) => void;
    TestBed.inject(UnsavedChanges).register(() => new Promise(resolve => { approve = resolve; }));
    const changing = context.change('project-1'); session.clear(); approve(true);
    expect(await changing).toBe(false); expect(context.context().projectId).toBeNull();
  });
  it('emits the exact effects only after an explicit confirmed commit', () => {
    const bus = TestBed.inject(UiEvents); const events: UiEvent[] = []; bus.events.subscribe(event => events.push(event));
    bus.committed({ records: [{ resource: 'projects', id: 'p', projectId: 'p', version: 2 }], stock: { transactionId: 't', projectId: 'p', siteIds: ['s'], materialIds: ['m'] }, unit: { propertyId: 'u', projectId: 'p', availability: 'available', bookingId: null }, finance: { projectId: 'p', counterpartyId: 'c', bookingId: null, invoiceIds: [], paymentIds: [] } }, 'request');
    expect(events.map(event => event.name)).toEqual(['record.committed', 'stock.committed', 'unit.committed', 'finance.committed']);
    expect(events.every(event => 'requestId' in event.payload && event.payload.requestId === 'request')).toBe(true);
  });
});
