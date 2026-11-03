import { TestBed } from '@angular/core/testing';
import { provideHttpClient, HttpErrorResponse } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, vi } from 'vitest';
import { SettingsPage } from './settings-page';
import { Settings, SettingsApi } from './settings-api';
import { StewardshipPanel } from './stewardship-panel';
import { SETTINGS_ROUTES } from './route-slot';
import { settingsAccess } from './settings-access';
import { provideRouter, Router, ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { By } from '@angular/platform-browser';

const settings = (): Settings => ({ version: 1, companyName: 'Test company', branding: {wordmark: 'Test', logoDocumentId: null}, currencyCode: null, currencyMinorDigits: null, currencyLocked: false,
  timezone: 'UTC', timezoneConfirmed: false, recoveryInstructions: 'Contact the administrator in person', recoveryContactLabel: 'Administrator', reservationHours: 48,
  referenceValues: [], inspectionChecklists: [], propertyTypes: [], operatingPolicy: {countries: [], noticeVersion: null, retentionScheduleVersion: null, confirmedAt: null} });
async function setup(value = settings()) {
  const api = { get: vi.fn(async (path: string) => path === 'settings' ? structuredClone(value) : path === 'operator/status' ? {version: '1', startedAt: '2026-01-01T00:00:00Z', ready: false, database: {status: 'ok'}, attachments: {status: 'unavailable'}, backup: {status: 'not-configured', lastSuccessAt: null, nextDueAt: null, diagnosticId: 'backup-unconfigured'}} : []), mutate: vi.fn() };
  await TestBed.configureTestingModule({imports: [SettingsPage], providers: [{provide: SettingsApi, useValue: api}]}).compileComponents();
  const fixture = TestBed.createComponent(SettingsPage); fixture.detectChanges(); await vi.waitFor(() => expect(fixture.componentInstance.loading()).toBe(false)); fixture.detectChanges();
  return { fixture, page: fixture.componentInstance, api };
}
function complete(page: SettingsPage) { page.form.patchValue({currencyCode: 'USD', currencyMinorDigits: 2, timezoneConfirmed: true, countries: 'US', noticeVersion: 'notice-1', retentionScheduleVersion: 'retention-1', confirmOperationalPolicy: true}); }

describe('Company settings', () => {
  it('routes to the implemented settings page', async () => { const load = SETTINGS_ROUTES[0].loadComponent as () => Promise<unknown>; expect(await load()).toBe(SettingsPage); });
  it('starts with UTC but requires explicit currency and timezone confirmation', async () => {
    const {page, api, fixture} = await setup();
    expect(page.form.controls.timezone.value).toBe('UTC'); expect(page.form.controls.currencyCode.value).toBe('');
    await page.save(); expect(api.mutate).not.toHaveBeenCalled();
    expect(fixture.nativeElement.textContent).toContain('Operational setup is incomplete');
    expect(fixture.nativeElement.textContent).toContain('not-configured');
  });
  it('saves only editable fields and keeps policy confirmation explicit', async () => {
    const {page, api} = await setup(); complete(page);
    api.mutate.mockResolvedValue({...settings(), currencyCode: 'USD', currencyMinorDigits: 2, version: 2});
    await page.save();
    const [method, path, body, key] = api.mutate.mock.calls[0] as unknown as [string, string, Record<string, unknown>, string];
    expect(method).toBe('PUT'); expect(path).toBe('settings'); expect(body['confirmOperationalPolicy']).toBe(true);
    expect(body).not.toHaveProperty('currencyLocked'); expect(body['operatingPolicy']).not.toHaveProperty('confirmedAt'); expect(key).toMatch(/^[0-9a-f-]{36}$/);
    expect(page.form.controls.confirmOperationalPolicy.value).toBe(false);
  });
  it('locks currency controls when posting has locked the server record', async () => {
    const value = settings(); value.currencyCode = 'USD'; value.currencyMinorDigits = 2; value.currencyLocked = true;
    const {page} = await setup(value); expect(page.form.controls.currencyCode.disabled).toBe(true); expect(page.form.controls.currencyMinorDigits.disabled).toBe(true);
  });
  it('creates checklist revisions with server-assigned IDs and newline items', async () => {
    const {page, api} = await setup(); complete(page); page.addChecklist();
    page.form.controls.inspectionChecklists.at(0).patchValue({name: 'Safety', items: 'Access clear\nSigns visible'});
    api.mutate.mockResolvedValue(settings()); await page.save();
    expect(api.mutate.mock.calls[0][2].inspectionChecklists).toEqual([{id: '', name: 'Safety', items: ['Access clear', 'Signs visible']}]);
  });
  it('uses saved policy rather than an editable consent downgrade', async () => {
    const value = settings(); value.referenceValues = [{key: 'privacy.consent-required', value: 'false', label: 'Owner-approved policy'}];
    const {page, fixture} = await setup(value); expect(page.consentRequired()).toBe(false);
    fixture.detectChanges(); const panel = fixture.debugElement.query(By.directive(StewardshipPanel)).componentInstance as StewardshipPanel;
    expect(panel.acknowledgment.controls.consentRequired.value).toBe(false);
  });
  it('preserves input and idempotency key on uncertain failure without automatic retry', async () => {
    const {page, api} = await setup(); complete(page); api.mutate.mockRejectedValue(new HttpErrorResponse({status: 0}));
    await page.save(); expect(api.mutate).toHaveBeenCalledTimes(1); expect(page.form.controls.countries.value).toBe('US');
    const key = api.mutate.mock.calls[0][3]; await page.save(); expect(api.mutate.mock.calls[1][3]).toBe(key);
    page.form.controls.companyName.setValue('Edited'); await page.save(); expect(api.mutate.mock.calls[2][3]).not.toBe(key);
  });
  it('blocks duplicate submissions while the original is pending', async () => {
    const {page, api} = await setup(); complete(page);
    let finish!: (value: Settings) => void; api.mutate.mockReturnValue(new Promise<Settings>(resolve => { finish = resolve; }));
    const pending = page.save(); await page.save(); expect(api.mutate).toHaveBeenCalledTimes(1); finish(settings()); await pending;
  });
  it('loads conflict comparison without overwriting safe input', async () => {
    const {page, api} = await setup(); complete(page); page.form.controls.companyName.setValue('Unsaved company');
    api.mutate.mockRejectedValue(new HttpErrorResponse({status: 409})); await page.save(); await page.compare();
    expect(page.comparison()?.version).toBe(1); expect(page.form.controls.companyName.value).toBe('Unsaved company');
  });
  it('renders labeled stewardship controls and requires execution confirmation', async () => {
    const {fixture, api} = await setup();
    const instance = fixture.debugElement.query(By.directive(StewardshipPanel)).componentInstance as StewardshipPanel;
    await instance.executeRequest({id: 'request-1', version: 1, subjectType: 'lead', subjectId: 'lead-1', kind: 'anonymization', reason: 'Requested', status: 'approved'});
    expect(api.mutate).not.toHaveBeenCalled(); expect(fixture.nativeElement.textContent).toContain('Legal holds and reviewed retention');
  });
});

describe('Settings API', () => {
  it('denies non-administrators before loading the settings page', async () => {
    TestBed.configureTestingModule({providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])]});
    const http = TestBed.inject(HttpTestingController);
    const result = TestBed.runInInjectionContext(() => settingsAccess({} as ActivatedRouteSnapshot, {} as RouterStateSnapshot));
    http.expectOne('/api/v1/auth/me').flush({data: {user: {id: 'user', displayName: 'Viewer', email: 'viewer@example.test', role: 'management-viewer'}, setupRequired: false, capabilities: [], projectIds: [], accessRevision: 1, landingPath: '/dashboard'}});
    const decision = await result; expect(TestBed.inject(Router).serializeUrl(decision as ReturnType<Router['createUrlTree']>)).toBe('/settings/access-denied'); http.verify();
  });
  it('bootstraps CSRF and sends the deliberate request key', async () => {
    TestBed.configureTestingModule({providers: [provideHttpClient(), provideHttpClientTesting()]});
    const api = TestBed.inject(SettingsApi), http = TestBed.inject(HttpTestingController);
    const promise = api.mutate('PUT', 'settings', {version: 1}, 'request-key');
    http.expectOne('/api/v1/auth/csrf').flush({data: {csrfToken: 'csrf-nonce'}, requestId: 'id'});
    const request = await vi.waitFor(() => http.expectOne('/api/v1/settings'));
    expect(request.request.headers.get('X-CSRF-Token')).toBe('csrf-nonce');
    expect(request.request.headers.get('Idempotency-Key')).toBe('request-key');
    request.flush({data: {version: 2}, requestId: 'id'});
    expect(await promise).toEqual({version: 2}); http.verify();
  });
});
