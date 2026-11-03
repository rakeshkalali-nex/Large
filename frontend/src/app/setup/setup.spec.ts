import { describe, it, expect } from 'vitest';
import { getPlatform, PlatformRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { UnavailablePage } from './unavailable';
import { APP_ROUTES } from '../app.routes';
import { protectedEntryGuard } from '../core/guards/entry.guards';
describe('setup', () => {
  it('shares the initialized platform between Angular core and TestBed', () => {
    const platform = getPlatform();
    expect(platform).not.toBeNull();
    expect(platform).toBe(TestBed.platform);
    expect(TestBed.inject(PlatformRef)).toBe(platform);
    TestBed.resetTestingModule();
    expect(getPlatform()).toBe(platform);
    expect(TestBed.inject(PlatformRef)).toBe(platform);
  });
  it('renders an honest unavailable boundary without business controls', () => {
    const fixture = TestBed.createComponent(UnavailablePage); fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[role="status"]').textContent).toContain('not available');
  });
  it('reserves entry and lazy feature routes', () => {
    const publicFeatures = APP_ROUTES.filter(route => route.loadChildren);
    const protectedGroup = APP_ROUTES.find(route => route.canActivateChild?.includes(protectedEntryGuard));
    expect(publicFeatures).toHaveLength(1);
    expect(publicFeatures[0].path).toBe('');
    expect(protectedGroup).toBeDefined();
    expect(protectedGroup?.children?.filter(route => route.loadChildren)).toHaveLength(9);
    expect(protectedGroup?.children?.some(route => route.path === 'access-denied')).toBe(true);
    expect(protectedGroup?.children?.at(-1)?.path).toBe('**');
  });
});
