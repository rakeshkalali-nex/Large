import { describe, it, expect } from 'vitest';
import { getPlatform, PlatformRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { UnavailablePage } from './unavailable';
import { APP_ROUTES } from '../app.routes';
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
    expect(APP_ROUTES.some(route => route.path === '')).toBe(true);
    expect(APP_ROUTES.filter(route => route.loadChildren)).toHaveLength(10);
  });
});
