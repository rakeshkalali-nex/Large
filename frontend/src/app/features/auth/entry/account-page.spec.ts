import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, Router } from '@angular/router';
import { BehaviorSubject } from 'rxjs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SessionService } from '../../../core/session/session.service';
import { AccountPage } from './account-page';

function mount(state = '', fragment: string | null = null, status = 'anonymous', restoreFailure = false, expired = false) {
  const query = new BehaviorSubject(convertToParamMap({ state, returnUrl: '/projects' }));
  const fragments = new BehaviorSubject(fragment);
  const service = {
    store: { status: signal(status) },
    restoreFailure: signal(restoreFailure),
    expired: signal(expired),
    login: vi.fn(async (_email: string, _password: string) => { service.store.status.set('authenticated'); }),
    logout: vi.fn(async () => { service.store.status.set('anonymous'); }),
    restore: vi.fn(async () => {}),
    guidance: vi.fn(async () => ({ instructions: 'Call your known company number to verify your identity.', contactLabel: 'Company Administrator' })),
    completePassword: vi.fn(async (_token: string, _password: string) => {}),
    changePassword: vi.fn(async (_current: string, _password: string) => {}),
    destination: vi.fn((_returnUrl: string | null) => '/projects'),
  };
  const router = { navigateByUrl: vi.fn(async (_url: string) => true) };
  TestBed.configureTestingModule({ providers: [
    { provide: SessionService, useValue: service },
    { provide: Router, useValue: router },
    { provide: ActivatedRoute, useValue: {
      queryParamMap: query.asObservable(), fragment: fragments.asObservable(),
      snapshot: { queryParamMap: query.value, routeConfig: { path: 'login' } },
    } },
  ] });
  const fixture = TestBed.createComponent(AccountPage);
  fixture.detectChanges();
  const page = fixture.componentInstance;
  const element: HTMLElement = fixture.nativeElement;
  return { fixture, page, element, service, router, query, fragments };
}

function credentials(page: AccountPage): void {
  page.form.patchValue({ email: 'Person@example.com', password: 'private password', confirmPassword: 'private password' });
}
function failure(status: number, code: string): HttpErrorResponse {
  return new HttpErrorResponse({ status, error: { error: { code, message: 'Unsafe internal detail' } } });
}

afterEach(() => { TestBed.resetTestingModule(); vi.restoreAllMocks(); window.history.replaceState(null, '', '/'); });

describe('auth AccountPage sign in', () => {
  it('renders visible labels, autocomplete, a masked password and an accessible visibility control', () => {
    const { page, element, fixture } = mount();
    credentials(page);
    fixture.detectChanges();
    const password = element.querySelector<HTMLInputElement>('#password')!;
    expect(element.querySelector('label[for="email"]')?.textContent).toBe('Email address');
    expect(element.querySelector('label[for="password"]')?.textContent).toBe('Password');
    expect(element.querySelector('#email')?.getAttribute('autocomplete')).toBe('username');
    expect(password.autocomplete).toBe('current-password');
    expect(password.type).toBe('password');
    element.querySelector<HTMLButtonElement>('[aria-label="Show password"]')!.click();
    fixture.detectChanges();
    expect(password.type).toBe('text');
    expect(password.value).toBe('private password');
    expect(element.querySelector('[aria-label="Hide password"]')?.getAttribute('aria-pressed')).toBe('true');
  });

  it('submits through the native form event and delegates return-path selection', async () => {
    const { page, service, element, router, fixture } = mount();
    credentials(page);
    element.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await fixture.whenStable();
    expect(service.login).toHaveBeenCalledWith('Person@example.com', 'private password');
    expect(service.destination).toHaveBeenCalledWith('/projects');
    expect(router.navigateByUrl).toHaveBeenCalledWith('/projects');
    expect(page.form.controls.email.value).toBe('Person@example.com');
    expect(page.form.controls.password.value).toBe('');
  });

  it('announces loading and blocks duplicate submissions until completion', async () => {
    const { page, service, fixture, element } = mount();
    let resolve!: () => void;
    service.login.mockImplementation(() => new Promise<void>(done => { resolve = done; }));
    credentials(page);
    const request = page.submit();
    await page.submit();
    fixture.detectChanges();
    expect(service.login).toHaveBeenCalledTimes(1);
    expect(element.querySelector('fieldset')?.disabled).toBe(true);
    expect(element.querySelector('[role="status"]')?.textContent).toContain('Signing in');
    resolve();
    await request;
    expect(page.pending()).toBe(false);
  });

  it.each([
    [401, 'INVALID_CREDENTIALS', 'Email or password is incorrect.'],
    [429, 'RATE_LIMITED', 'Too many attempts.'],
    [0, '', 'We could not reach the service'],
    [503, 'SERVICE_UNAVAILABLE', 'We could not reach the service'],
    [403, 'CSRF_INVALID', 'Your security check expired.'],
  ])('handles %s safely without replay and clears only sensitive input', async (status, code, message) => {
    const { page, service, fixture, element } = mount();
    service.login.mockRejectedValue(failure(status as number, code as string));
    credentials(page);
    page.visible.set(true);
    await page.submit();
    fixture.detectChanges();
    expect(page.error()).toContain(message);
    expect(element.textContent).not.toContain('Unsafe internal detail');
    expect(service.login).toHaveBeenCalledTimes(1);
    expect(page.form.controls.email.value).toBe('Person@example.com');
    expect(page.form.controls.password.value).toBe('');
    expect(page.visible()).toBe(false);
    expect(document.activeElement).toBe(element.querySelector('[role="alert"]'));
  });

  it('rejects missing credentials locally and clears password after the attempt', async () => {
    const { page, service } = mount();
    page.form.patchValue({ email: 'not-an-email', password: 'secret' });
    await page.submit();
    expect(service.login).not.toHaveBeenCalled();
    expect(page.error()).toContain('valid email');
    expect(page.form.controls.password.value).toBe('');
  });

  it('keeps setup-required login in the account shell rather than entering protected content', async () => {
    const { page, service, router, element, fixture } = mount();
    service.login.mockImplementation(async () => { service.store.status.set('setupRequired'); });
    credentials(page);
    await page.submit();
    fixture.detectChanges();
    expect(page.state()).toBe('setup');
    expect(router.navigateByUrl).not.toHaveBeenCalled();
    expect(element.textContent).toContain('Sign out');
    expect(element.querySelector('form')).toBeNull();
  });
});

describe('auth AccountPage password lifecycle', () => {
  it('uses restoration failure and expiry signals on ordinary login visits', () => {
    const { page, service, query } = mount('', null, 'anonymous', true);
    expect(page.state()).toBe('unavailable');
    service.restoreFailure.set(false);
    service.expired.set(true);
    query.next(convertToParamMap({}));
    expect(page.state()).toBe('expired');
  });

  it('recognizes state=service and keeps absorbed restoration failures actionable', async () => {
    const { page, service, router } = mount('service', null, 'anonymous', true);
    expect(page.state()).toBe('unavailable');
    await page.retryConnection();
    expect(page.state()).toBe('unavailable');
    expect(page.error()).toContain('still unavailable');
    expect(router.navigateByUrl).not.toHaveBeenCalled();
    service.restore.mockImplementation(async () => { service.restoreFailure.set(false); service.expired.set(true); });
    await page.retryConnection();
    expect(page.state()).toBe('expired');
  });

  it('extracts backend fragment immediately, removes it from history and never renders the token', async () => {
    const token = 'a'.repeat(43);
    window.history.replaceState({ navigationId: 7 }, '', `/login#token=${token}`);
    const replace = vi.spyOn(window.history, 'replaceState');
    const { page, service, element } = mount('', `token=${token}`);
    expect(replace).toHaveBeenCalledWith({ navigationId: 7 }, '', '/login');
    expect(window.location.hash).toBe('');
    expect(page.state()).toBe('reset');
    expect(element.innerHTML).not.toContain(token);
    credentials(page);
    await page.submit();
    expect(service.completePassword).toHaveBeenCalledWith(token, 'private password');
    expect(page.state()).toBe('success');
    expect(page.form.controls.password.value).toBe('');
    expect(page.form.controls.confirmPassword.value).toBe('');
  });

  it.each(['token=bad', `token=${'a'.repeat(43)}&token=${'b'.repeat(43)}`, 'unknown=value'])('rejects malformed fragment %s and still removes it', fragment => {
    window.history.replaceState(null, '', '/login#secret');
    const { page, service } = mount('', fragment);
    expect(page.state()).toBe('invalid-link');
    expect(window.location.hash).toBe('');
    expect(service.completePassword).not.toHaveBeenCalled();
  });

  it('supports explicit setup links and validates Unicode character count rather than UTF-16 units', async () => {
    const { page, service } = mount('setup', `token=${'a'.repeat(43)}`);
    expect(page.state()).toBe('setup');
    page.form.patchValue({ password: '😀'.repeat(6), confirmPassword: '😀'.repeat(6) });
    await page.submit();
    expect(service.completePassword).not.toHaveBeenCalled();
    page.form.patchValue({ password: '😀'.repeat(12), confirmPassword: '😀'.repeat(12) });
    await page.submit();
    expect(service.completePassword).toHaveBeenCalledTimes(1);
  });

  it('rejects mismatched confirmation without a request', async () => {
    const { page, service } = mount('reset', `token=${'a'.repeat(43)}`);
    page.form.patchValue({ password: 'private password', confirmPassword: 'different password' });
    await page.submit();
    expect(service.completePassword).not.toHaveBeenCalled();
    expect(page.error()).toContain('confirm it exactly');
    expect(page.form.controls.confirmPassword.value).toBe('');
  });

  it('turns expired, consumed or revoked tokens into replacement guidance', async () => {
    const { page, service } = mount('reset', `token=${'a'.repeat(43)}`);
    service.completePassword.mockRejectedValue(failure(422, 'INVALID_RESET_TOKEN'));
    credentials(page);
    await page.submit();
    expect(page.state()).toBe('invalid-link');
    expect(page.isPasswordForm()).toBe(false);
    await page.recover();
    expect(service.guidance).toHaveBeenCalledTimes(1);
    expect(page.instructions()).toContain('verify your identity');
  });

  it('uses authenticated password change and clears every password field', async () => {
    const { page, service } = mount('change', null, 'authenticated');
    credentials(page);
    page.form.controls.currentPassword.setValue('old private password');
    await page.submit();
    expect(service.changePassword).toHaveBeenCalledWith('old private password', 'private password');
    expect(service.completePassword).not.toHaveBeenCalled();
    expect(page.form.controls.currentPassword.value).toBe('');
    expect(page.state()).toBe('success');
  });

  it('keeps incorrect current-password feedback distinct from session expiry', async () => {
    const { page, service } = mount('change', null, 'authenticated');
    credentials(page);
    page.form.controls.currentPassword.setValue('wrong');
    service.changePassword.mockRejectedValue(failure(401, 'INVALID_CREDENTIALS'));
    await page.submit();
    expect(page.state()).toBe('change');
    expect(page.error()).toContain('current password is incorrect');
    credentials(page);
    page.form.controls.currentPassword.setValue('old private password');
    service.changePassword.mockRejectedValue(failure(401, 'SESSION_EXPIRED'));
    await page.submit();
    expect(page.state()).toBe('expired');
  });

  it('requires sign in before an anonymous password change', () => {
    const { page } = mount('change');
    expect(page.state()).toBe('expired');
  });

  it('loads configured recovery guidance, announces failure and waits for deliberate retry', async () => {
    const { page, service } = mount();
    service.guidance.mockRejectedValueOnce(failure(503, 'SERVICE_UNAVAILABLE'));
    await page.recover();
    expect(page.error()).toContain('could not be loaded');
    expect(service.guidance).toHaveBeenCalledTimes(1);
    await page.loadGuidance();
    expect(page.contactLabel()).toBe('Company Administrator');
    expect(page.error()).toBe('');
  });

  it('supports expiry and service failure routes without submitting credentials automatically', async () => {
    const { page, service, query } = mount('expired');
    expect(page.state()).toBe('expired');
    query.next(convertToParamMap({ state: 'unavailable' }));
    expect(page.state()).toBe('unavailable');
    expect(service.restore).not.toHaveBeenCalled();
    await page.retryConnection();
    expect(service.restore).toHaveBeenCalledTimes(1);
    expect(page.state()).toBe('login');
    expect(service.login).not.toHaveBeenCalled();
  });

  it('allows setup-only logout and ignores successful pending login after destruction', async () => {
    const { page, service, fixture, router } = mount('setup', null, 'setupRequired');
    await page.logout();
    expect(service.logout).toHaveBeenCalledTimes(1);
    expect(page.state()).toBe('login');
    let resolve!: () => void;
    service.login.mockImplementation(() => new Promise<void>(done => { resolve = done; }));
    credentials(page);
    const request = page.submit();
    fixture.destroy();
    resolve();
    await request;
    expect(router.navigateByUrl).not.toHaveBeenCalled();
    expect(page.form.controls.password.value).toBe('');
  });
});
