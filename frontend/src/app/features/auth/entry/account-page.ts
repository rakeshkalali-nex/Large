import { DOCUMENT } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, DestroyRef, ElementRef, inject, OnDestroy, OnInit, signal, ViewChild } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { combineLatest } from 'rxjs';
import { SessionService } from '../../../core/session/session.service';

type AccountState = 'login' | 'recovery' | 'setup' | 'reset' | 'change' | 'expired' | 'unavailable' | 'invalid-link' | 'success' | 'denied';

@Component({
  selector: 'app-account-page',
  standalone: true,
  imports: [ReactiveFormsModule],
  templateUrl: './account-page.html',
  styleUrl: './account-page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AccountPage implements OnInit, AfterViewInit, OnDestroy {
  readonly session = inject(SessionService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly document = inject(DOCUMENT);
  private readonly changeDetector = inject(ChangeDetectorRef);
  private readonly destroyRef = inject(DestroyRef);
  @ViewChild('heading') private heading?: ElementRef<HTMLElement>;
  @ViewChild('feedback') private feedback?: ElementRef<HTMLElement>;
  readonly state = signal<AccountState>('login');
  readonly pending = signal(false);
  readonly error = signal('');
  readonly status = signal('');
  readonly visible = signal(false);
  readonly instructions = signal('');
  readonly contactLabel = signal('');
  readonly form = new FormGroup({
    email: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.email] }),
    password: new FormControl('', { nonNullable: true }),
    currentPassword: new FormControl('', { nonNullable: true }),
    confirmPassword: new FormControl('', { nonNullable: true }),
  });
  private token = '';
  private extractedFragment: string | null = null;
  private destroyed = false;
  private viewReady = false;

  ngOnInit(): void {
    combineLatest([this.route.queryParamMap, this.route.fragment]).pipe(takeUntilDestroyed(this.destroyRef)).subscribe(([params, fragment]) => {
      const rawFragment = fragment ?? this.document.defaultView?.location.hash.slice(1) ?? '';
      if (rawFragment && rawFragment !== this.extractedFragment) {
        // Read the backend's /login#token= format before any request or rendering.
        // Never persist the token, echo it into the DOM, or send it to navigation.
        this.extractedFragment = rawFragment;
        const values = new URLSearchParams(rawFragment);
        const tokens = values.getAll('token');
        this.token = tokens.length === 1 && /^[A-Za-z0-9_-]{43}$/.test(tokens[0]) ? tokens[0] : '';
        this.removeFragment();
        this.setState(this.token ? (params.get('state') === 'setup' ? 'setup' : 'reset') : 'invalid-link');
        return;
      }
      if (this.token) return;
      const requested = params.get('state');
      const states: Record<string, AccountState> = {
        recovery: 'recovery', setup: 'setup', reset: 'invalid-link', change: 'change',
        expired: 'expired', 'session-expired': 'expired',
        service: 'unavailable', unavailable: 'unavailable', 'service-error': 'unavailable',
      };
      let next = states[requested ?? ''] ?? (this.route.snapshot.routeConfig?.path === 'access-denied' ? 'denied' : 'login');
      if (next === 'login' || next === 'expired' || next === 'unavailable') {
        if (this.session.restoreFailure()) next = 'unavailable';
        else if (this.session.expired()) next = 'expired';
      }
      if (this.session.store.status() === 'setupRequired' && next !== 'recovery') next = 'setup';
      if (next === 'change' && this.session.store.status() !== 'authenticated') next = 'expired';
      this.setState(next);
      if (next === 'recovery') void this.loadGuidance();
    });
  }

  ngAfterViewInit(): void { this.viewReady = true; this.focus('heading'); }
  ngOnDestroy(): void { this.destroyed = true; this.token = ''; this.extractedFragment = null; this.clearPasswords(); }

  title(): string {
    return {
      login: 'Welcome back', recovery: 'Recover your account', setup: 'Set up your password',
      reset: 'Set a new password', change: 'Change your password', expired: 'Your session has expired',
      unavailable: 'Unable to connect', 'invalid-link': 'This link is no longer available',
      success: 'Password updated', denied: 'Access denied',
    }[this.state()];
  }
  isSignIn(): boolean { return this.state() === 'login' || this.state() === 'expired'; }
  isPasswordForm(): boolean { return this.state() === 'change' || ((this.state() === 'reset' || this.state() === 'setup') && !!this.token); }
  submitLabel(): string {
    if (this.pending()) return this.isSignIn() ? 'Signing in…' : 'Saving password…';
    return this.isSignIn() ? 'Sign in' : this.state() === 'setup' ? 'Set password' : 'Save password';
  }
  togglePassword(): void { this.visible.update(value => !value); }

  async submit(): Promise<void> {
    if (this.pending() || (!this.isSignIn() && !this.isPasswordForm())) return;
    const signingIn = this.isSignIn();
    const changing = this.state() === 'change';
    const values = this.form.getRawValue();
    this.error.set('');
    if (signingIn && (this.form.controls.email.invalid || !values.password)) {
      this.clearPasswords();
      this.fail('Enter a valid email address and your password.');
      return;
    }
    const length = Array.from(values.password).length;
    if (!signingIn && (length < 12 || length > 128 || values.password !== values.confirmPassword || (changing && !values.currentPassword))) {
      this.clearPasswords();
      this.fail('Enter a password of 12 to 128 characters, confirm it exactly, and enter your current password if requested.');
      return;
    }
    this.pending.set(true);
    this.status.set(signingIn ? 'Signing in. Please wait.' : 'Saving your password. Please wait.');
    try {
      if (signingIn) {
        await this.session.login(values.email, values.password);
        if (this.destroyed) return;
        if (this.session.store.status() === 'setupRequired') this.setState('setup');
        else await this.router.navigateByUrl(this.session.destination(this.route.snapshot.queryParamMap.get('returnUrl')));
      } else {
        if (changing) await this.session.changePassword(values.currentPassword, values.password);
        else await this.session.completePassword(this.token, values.password);
        if (this.destroyed) return;
        this.token = '';
        this.setState('success');
      }
    } catch (error: unknown) {
      if (!this.destroyed) this.handleError(error, changing);
    } finally {
      this.clearPasswords();
      this.pending.set(false);
      this.status.set('');
      if (!this.destroyed) this.focus(this.error() ? 'feedback' : 'heading');
    }
  }

  async recover(): Promise<void> {
    if (this.pending()) return;
    this.token = '';
    this.setState('recovery');
    await this.loadGuidance();
  }
  async loadGuidance(): Promise<void> {
    if (this.pending() || this.destroyed) return;
    this.pending.set(true);
    this.error.set('');
    this.status.set('Loading recovery instructions…');
    try {
      const guidance = await this.session.guidance();
      if (!this.destroyed) { this.instructions.set(guidance.instructions); this.contactLabel.set(guidance.contactLabel); }
    } catch {
      if (!this.destroyed) this.fail('Recovery instructions could not be loaded. Check your connection and try again, or contact your company Administrator through a known private channel.');
    } finally {
      this.pending.set(false);
      this.status.set('');
      if (!this.destroyed) this.focus(this.error() ? 'feedback' : 'heading');
    }
  }
  signIn(): void {
    if (this.pending()) return;
    this.token = '';
    this.setState(this.session.store.status() === 'setupRequired' ? 'setup' : 'login');
  }
  async retryConnection(): Promise<void> {
    if (this.pending()) return;
    this.pending.set(true);
    this.error.set('');
    this.status.set('Checking your session…');
    try {
      await this.session.restore();
      if (this.destroyed) return;
      if (this.session.restoreFailure()) {
        this.setState('unavailable');
        this.fail('The service is still unavailable. Check your connection and try again.');
      } else if (this.session.store.status() === 'authenticated') await this.router.navigateByUrl(this.session.destination(this.route.snapshot.queryParamMap.get('returnUrl')));
      else if (this.session.expired()) this.setState('expired');
      else this.setState(this.session.store.status() === 'setupRequired' ? 'setup' : 'login');
    } catch { if (!this.destroyed) this.fail('The service is still unavailable. Check your connection and try again.'); }
    finally { this.pending.set(false); this.status.set(''); }
  }
  async logout(): Promise<void> {
    if (this.pending()) return;
    this.pending.set(true);
    this.status.set('Signing out…');
    try { await this.session.logout(); if (!this.destroyed) { this.token = ''; this.setState('login'); } }
    catch { if (!this.destroyed) this.fail('Sign out could not be confirmed. Please try again.'); }
    finally { this.clearPasswords(); this.pending.set(false); this.status.set(''); }
  }
  async safeDestination(): Promise<void> {
    if (!this.pending()) await this.router.navigateByUrl(this.session.destination(null));
  }
  private setState(state: AccountState): void {
    this.clearPasswords();
    this.error.set('');
    this.state.set(state);
    this.focus('heading');
  }
  private clearPasswords(): void {
    this.form.controls.password.reset('');
    this.form.controls.currentPassword.reset('');
    this.form.controls.confirmPassword.reset('');
    this.visible.set(false);
  }
  private removeFragment(): void {
    const window = this.document.defaultView;
    if (window) window.history.replaceState(window.history.state, '', window.location.pathname + window.location.search);
  }
  private handleError(error: unknown, changing: boolean): void {
    const http = error instanceof HttpErrorResponse ? error : null;
    const code: unknown = http?.error?.error?.code;
    if (code === 'INVALID_RESET_TOKEN') { this.token = ''; this.setState('invalid-link'); return; }
    if (code === 'SETUP_REQUIRED') { this.setState('setup'); return; }
    if (code === 'SESSION_EXPIRED' || code === 'AUTH_REQUIRED') {
      this.setState('expired'); this.fail('Your session has expired. Sign in again to continue.'); return;
    }
    if (http?.status === 429) { this.fail('Too many attempts. Wait before trying again. No request will be retried automatically.'); return; }
    if (code === 'INVALID_CREDENTIALS' || http?.status === 401) {
      this.fail(changing ? 'Your current password is incorrect. Please try again.' : 'Email or password is incorrect.'); return;
    }
    if (code === 'VALIDATION_FAILED') { this.fail('Check your details. New passwords must contain 12 to 128 characters.'); return; }
    if (code === 'CSRF_INVALID') { this.fail('Your security check expired. Please try again.'); return; }
    this.fail('We could not reach the service or confirm this request. Check your connection and try again.');
  }
  private fail(message: string): void { this.error.set(message); this.focus('feedback'); }
  private focus(target: 'heading' | 'feedback'): void {
    if (!this.viewReady || this.destroyed) return;
    this.changeDetector.detectChanges();
    (target === 'heading' ? this.heading : this.feedback)?.nativeElement.focus();
  }
}
