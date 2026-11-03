import { computed, inject, Injectable, signal } from '@angular/core';
import type { components } from '../contracts/generated/api';
import { SessionStatus, UiEvents } from './ui-events';

/** The authentication owner supplies verified /auth/me or login responses here. */
@Injectable({ providedIn: 'root' })
export class SessionStore {
  private readonly events = inject(UiEvents);
  private readonly identityState = signal<components['schemas']['Identity'] | null>(null);
  private readonly statusState = signal<SessionStatus>('restoring');
  readonly status = this.statusState.asReadonly();
  readonly user = computed(() => this.identityState()?.user ?? null);
  readonly projectIds = computed(() => this.identityState()?.projectIds ?? []);
  readonly capabilities = computed(() => this.status() === 'authenticated' ? this.identityState()?.capabilities ?? [] : []);
  readonly landingPath = computed(() => this.identityState()?.landingPath ?? '/login');
  readonly csrfToken = computed(() => this.identityState()?.csrfToken ?? null);
  readonly idleExpiresAt = computed(() => this.identityState()?.idleExpiresAt ?? null);
  readonly absoluteExpiresAt = computed(() => this.identityState()?.absoluteExpiresAt ?? null);
  has(capability: string): boolean { return this.capabilities().includes(capability); }
  accept(identity: components['schemas']['Identity']): void {
    this.identityState.set(structuredClone(identity));
    this.statusState.set(identity.setupRequired ? 'setupRequired' : 'authenticated');
    this.events.emit({ name: 'session.changed', payload: { status: this.status(), userId: identity.user.id, accessRevision: identity.accessRevision } });
  }
  clear(): void {
    this.identityState.set(null);
    this.statusState.set('anonymous');
    this.events.emit({ name: 'session.changed', payload: { status: 'anonymous', userId: null, accessRevision: null } });
  }
}
