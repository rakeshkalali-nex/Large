import { inject, Injectable, signal } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { Router } from '@angular/router';
import { SessionStore } from '../state/session-store';
import type { components } from '../contracts/generated/api';
import { authorizedPath, roleLanding } from './destinations';

type Identity = components['schemas']['Identity'];
interface Envelope<T> { data: T; requestId: string; }
@Injectable({ providedIn: 'root' })
export class SessionService {
  readonly store = inject(SessionStore);
  readonly restoreFailure = signal(false);
  readonly expired = signal(false);
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private restoring?: Promise<void>;
  restore(): Promise<void> {
    if (this.restoring) return this.restoring;
    if (this.store.status() !== 'restoring' && !this.restoreFailure()) return Promise.resolve();
    this.restoring = firstValueFrom(this.http.get<Envelope<Identity>>('/api/v1/auth/me'))
      .then(response => { this.store.accept(response.data); this.restoreFailure.set(false); })
      .catch((error: unknown) => {
        this.store.clear();
        this.restoreFailure.set(!(error instanceof HttpErrorResponse && error.status === 401));
        if (error instanceof HttpErrorResponse && error.error?.error?.code === 'SESSION_EXPIRED') this.expired.set(true);
      }).finally(() => { this.restoring = undefined; });
    return this.restoring;
  }
  private async mutate<T>(path: string, body: unknown): Promise<T> {
    // Fetch a fresh session-bound nonce; never retry a credential mutation.
    const csrf = await firstValueFrom(this.http.get<Envelope<{ csrfToken: string }>>('/api/v1/auth/csrf'));
    return firstValueFrom(this.http.post<T>('/api/v1/auth/' + path, body, {
      headers: { 'X-CSRF-Token': csrf.data.csrfToken, 'Idempotency-Key': crypto.randomUUID() }
    }));
  }
  async login(email: string, password: string): Promise<void> {
    const result = await this.mutate<Envelope<Identity>>('login', { email, password });
    this.store.accept(result.data); this.restoreFailure.set(false); this.expired.set(false);
  }
  async logout(): Promise<void> {
    await this.mutate('logout', {}); this.store.clear();
    await this.router.navigateByUrl('/login');
  }
  async guidance(): Promise<{ instructions: string; contactLabel: string }> {
    return (await firstValueFrom(this.http.get<Envelope<{ instructions: string; contactLabel: string }>>('/api/v1/auth/recovery-guidance'))).data;
  }
  async completePassword(token: string, newPassword: string): Promise<void> {
    await this.mutate('reset-password', { token, newPassword }); this.store.clear();
  }
  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await this.mutate('change-password', { currentPassword, newPassword }); this.store.clear();
  }
  destination(returnUrl: string | null): string {
    if (this.store.status() === 'setupRequired') return '/login?state=setup';
    if (this.store.status() !== 'authenticated') return '/login';
    return authorizedPath(returnUrl, this.store.capabilities()) ?? roleLanding(this.store.user()?.role);
  }
}
