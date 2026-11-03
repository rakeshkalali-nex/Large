import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { BreakpointObserver, BreakpointState } from '@angular/cdk/layout';
import { OverlayContainer } from '@angular/cdk/overlay';
import { BehaviorSubject } from 'rxjs';
import { describe, expect, it } from 'vitest';
import { AppShell } from './app-shell';
import { SessionStore } from '../state/session-store';
import type { components } from '../contracts/generated/api';

@Component({ standalone: true, template: '<h1>Project details</h1>' }) class TestPage {}
const verifiedIdentity: components['schemas']['Identity'] = { user: { id: 'u', displayName: 'Alex', email: 'alex@example.test', role: 'project-manager' }, projectIds: ['project-1'], capabilities: ['projects.read'], landingPath: '/projects', csrfToken: 'test-only', accessRevision: 1, idleExpiresAt: '2026-11-03T10:00:00Z', absoluteExpiresAt: '2026-11-03T20:00:00Z', setupRequired: false };
function setup(width = 1440) {
  const breakpoints = new BehaviorSubject<BreakpointState>({ matches: width <= 1024, breakpoints: { '(max-width: 768px)': width <= 768, '(max-width: 1024px)': width <= 1024 } });
  TestBed.configureTestingModule({ providers: [provideRouter([{ path: 'projects/:id', component: TestPage, title: 'Project overview' }]), { provide: BreakpointObserver, useValue: { observe: () => breakpoints } }] });
  const fixture = TestBed.createComponent(AppShell); return { fixture, breakpoints };
}
describe('core shell', () => {
  it('shows only server-granted destinations and hides shell before restoration', () => {
    const { fixture } = setup(); fixture.detectChanges(); expect(fixture.nativeElement.querySelector('aside')).toBeNull();
    TestBed.inject(SessionStore).accept(verifiedIdentity); fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Construction'); expect(fixture.nativeElement.textContent).not.toContain('Finance');
    expect(fixture.nativeElement.querySelector('a[href="/projects"]')).not.toBeNull();
    fixture.nativeElement.querySelector('.collapse').click(); fixture.detectChanges(); expect(fixture.componentInstance.collapsed()).toBe(true);
  });
  it.each([1440, 1024, 768, 390])('adapts navigation at %spx', width => {
    const { fixture } = setup(width); TestBed.inject(SessionStore).accept(verifiedIdentity); fixture.detectChanges();
    expect(fixture.componentInstance.mobile()).toBe(width <= 768); expect(fixture.componentInstance.collapsed()).toBe(width <= 1024);
    expect(!!fixture.nativeElement.querySelector('aside')).toBe(width > 768);
  });
  it('opens a labelled CDK mobile dialog and supports Escape', async () => {
    const { fixture } = setup(390); TestBed.inject(SessionStore).accept(verifiedIdentity); fixture.detectChanges();
    fixture.componentInstance.openNavigation(); fixture.detectChanges(); await fixture.whenStable();
    const overlay = TestBed.inject(OverlayContainer).getContainerElement();
    expect(overlay.querySelector('[role="dialog"]')?.getAttribute('aria-label')).toBe('Application navigation');
    overlay.querySelector('[role="dialog"]')?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', keyCode: 27, bubbles: true }));
    fixture.detectChanges(); await fixture.whenStable(); expect(overlay.querySelector('[role="dialog"]')).toBeNull();
  });
  it('derives detail breadcrumbs and project context from the route', async () => {
    const { fixture } = setup(); TestBed.inject(SessionStore).accept(verifiedIdentity); fixture.detectChanges();
    await TestBed.inject(Router).navigateByUrl('/projects/project-1'); fixture.detectChanges();
    expect(fixture.componentInstance.title()).toBe('Project overview'); expect(fixture.componentInstance.routeProjectId()).toBe('project-1');
    expect(fixture.nativeElement.querySelector('[aria-label="Breadcrumb"]').textContent).toContain('Projects');
  });
});
