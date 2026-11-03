import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SessionStore } from '../../../core/state/session-store';
import { User, UsersApi } from './users-api';
import { UsersPage } from './users-page';
import { USERS_ROUTES } from './route-slot';

const staff: User = { id: 'u1', displayName: 'Alex Staff', email: 'alex@example.test', role: 'site-engineer', status: 'active', version: 2, projectIds: [] };
describe('Users and access screen', () => {
  let fixture: ComponentFixture<UsersPage>, page: UsersPage;
  const role = signal('administrator'), status = signal('authenticated');
  const api = { list: vi.fn(), roles: vi.fn(), projects: vi.fn(), save: vi.fn(), transition: vi.fn(), issue: vi.fn() };
  beforeEach(async () => {
    role.set('administrator'); status.set('authenticated');
    for (const mock of Object.values(api)) mock.mockReset();
    api.list.mockResolvedValue([staff]);
    api.roles.mockResolvedValue([{ code: 'administrator', name: 'Administrator' }, { code: 'site-engineer', name: 'Site Engineer' }]);
    api.projects.mockResolvedValue([{ id: 'p1', name: 'North site' }]); api.save.mockResolvedValue(staff);
    api.transition.mockResolvedValue({ ...staff, status: 'setup-required' });
    api.issue.mockResolvedValue({ userId: staff.id, resetLink: '/login#token=one-time-secret', expiresAt: '2026-11-04T00:00:00Z' });
    Object.defineProperty(HTMLDialogElement.prototype, 'showModal', { configurable: true, value() { this.setAttribute('open', ''); } });
    Object.defineProperty(HTMLDialogElement.prototype, 'close', { configurable: true, value() { this.removeAttribute('open'); } });
    await TestBed.configureTestingModule({ imports: [UsersPage], providers: [
      { provide: UsersApi, useValue: api }, { provide: SessionStore, useValue: { status, user: () => ({ role: role() }) } },
    ] }).compileComponents();
    fixture = TestBed.createComponent(UsersPage); page = fixture.componentInstance;
  });
  async function render(): Promise<void> { fixture.detectChanges(); await fixture.whenStable(); fixture.detectChanges(); }
  it('wires the route to the actual screen', async () => {
    expect(USERS_ROUTES[0].path).toBe('users');
    expect(await (USERS_ROUTES[0].loadComponent as () => Promise<unknown>)()).toBe(UsersPage);
  });
  it.each(['project-manager', 'site-engineer', 'procurement-officer', 'finance-officer', 'sales-executive', 'management-viewer'])('hides controls and fetches no staff for %s', async value => {
    role.set(value); await render();
    expect(fixture.nativeElement.querySelector('button')).toBeNull(); expect(api.list).not.toHaveBeenCalled();
    page.edit(staff); page.confirmAction(staff, 'deactivate'); await page.save(); await page.performAction();
    expect(api.save).not.toHaveBeenCalled(); expect(api.transition).not.toHaveBeenCalled();
  });
  it('shows status, role, no-project feedback and searches explicitly', async () => {
    await render(); expect(fixture.nativeElement.textContent).toContain('No projects assigned');
    expect(fixture.nativeElement.textContent).toContain('Site Engineer'); expect(fixture.nativeElement.textContent).toContain('active');
    page.search = 'Alex'; await page.load(); expect(api.list).toHaveBeenLastCalledWith('Alex');
  });
  it('creates with explicit assignments and never issues setup automatically', async () => {
    await render(); page.edit(null); page.draft = { displayName: 'New Staff', email: 'new@example.test', role: 'site-engineer', projectIds: [] };
    page.toggleProject('p1', true); page.toggleProject('p1', true); await page.save();
    expect(api.save).toHaveBeenCalledWith({ displayName: 'New Staff', email: 'new@example.test', role: 'site-engineer', projectIds: ['p1'] }, null, expect.any(String));
    expect(api.issue).not.toHaveBeenCalled(); expect(page.notice()).toContain('explicitly issue a setup link');
  });
  it('edits a detached draft with the original version', async () => {
    await render(); page.edit(staff); page.toggleProject('p1', true); page.draft.displayName = 'Updated';
    expect(staff.projectIds).toEqual([]); expect(staff.displayName).toBe('Alex Staff');
    await page.save(); expect(api.save.mock.calls[0][1].version).toBe(2);
  });
  it('requires reason and trusted identity confirmation before transient link display', async () => {
    await render(); page.confirmAction(staff, 'recovery'); await page.performAction(); expect(api.issue).not.toHaveBeenCalled();
    page.reason = 'Known-number conversation'; await page.performAction(); expect(api.issue).not.toHaveBeenCalled();
    page.identityVerified = true; await page.performAction(); await render();
    expect(page.mode()).toBe('link'); expect(fixture.nativeElement.querySelector('#issued-link').value).toContain('#token=one-time-secret');
    expect(localStorage.length).toBe(0); expect(sessionStorage.length).toBe(0);
    page.dismiss(); await render(); expect(page.link()).toBeNull(); expect(fixture.nativeElement.querySelector('#issued-link')).toBeNull();
  });
  it('requires lifecycle reasons and explains fresh setup after reactivation', async () => {
    await render(); page.confirmAction(staff, 'deactivate'); await page.performAction(); expect(api.transition).not.toHaveBeenCalled();
    page.reason = 'Employment ended'; await page.performAction(); expect(api.transition).toHaveBeenCalledWith(staff, 'deactivate', 'Employment ended', expect.any(String));
    page.confirmAction({ ...staff, status: 'deactivated' }, 'reactivate'); page.reason = 'Returning employee'; await page.performAction();
    expect(page.notice()).toContain('setup-required'); expect(api.issue).not.toHaveBeenCalled();
  });
  it('preserves input on conflict and explains the final Administrator safeguard', async () => {
    await render(); api.save.mockRejectedValue(new HttpErrorResponse({ status: 409 })); page.edit(staff); page.draft.displayName = 'Retained'; await page.save();
    expect(page.draft.displayName).toBe('Retained'); expect(page.dialogError()).toContain('final active Administrator'); expect(page.mode()).toBe('editor');
  });
  it('blocks duplicate saves and reuses the UUID after an uncertain result', async () => {
    await render(); let reject!: (error: unknown) => void;
    api.save.mockImplementation(() => new Promise((_, fail) => { reject = fail; }));
    page.edit(staff); const saving = page.save(); await page.save(); expect(api.save).toHaveBeenCalledTimes(1);
    reject(new HttpErrorResponse({ status: 0 })); await saving;
    const key = api.save.mock.calls[0][2]; expect(key).toMatch(/^[0-9a-f-]{36}$/);
    api.save.mockResolvedValue(staff); await page.save(); expect(api.save.mock.calls[1][2]).toBe(key);
  });
  it('clears secrets and staff on Administrator access loss', async () => {
    await render(); page.confirmAction(staff, 'recovery'); page.reason = 'Verified'; page.identityVerified = true; await page.performAction();
    role.set('site-engineer'); await render(); expect(page.link()).toBeNull(); expect(page.users()).toEqual([]); expect(fixture.nativeElement.querySelector('button')).toBeNull();
  });
  it('does not restore a late issuance response after logout', async () => {
    await render(); let resolve!: (value: unknown) => void; api.issue.mockImplementation(() => new Promise(done => { resolve = done; }));
    page.confirmAction(staff, 'recovery'); page.reason = 'Verified'; page.identityVerified = true; const issuing = page.performAction();
    status.set('anonymous'); await render(); resolve({ resetLink: '/login#token=late-secret' }); await issuing;
    expect(page.link()).toBeNull();
  });
  it('announces throttling without leaking response bodies', async () => {
    await render(); api.issue.mockRejectedValue(new HttpErrorResponse({ status: 429, error: { secret: 'private' } }));
    page.confirmAction(staff, 'recovery'); page.reason = 'Verified'; page.identityVerified = true; await page.performAction();
    expect(page.dialogError()).toContain('Too many attempts'); expect(page.dialogError()).not.toContain('private');
  });
});
