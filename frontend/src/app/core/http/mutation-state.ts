import { signal } from '@angular/core';
import { SafeFailure, safeFailure } from './safe-error';

/** One instance per form. Only safe, non-secret payloads belong here. No automatic retries. */
export class MutationState<T, R> {
  readonly pending = signal(false);
  readonly failure = signal<SafeFailure | null>(null);
  readonly result = signal<R | null>(null);
  private attempt: { fingerprint: string; key: string } | null = null;
  async submit(payload: T, send: (payload: T, idempotencyKey: string) => Promise<R>): Promise<R | undefined> {
    if (this.pending()) return undefined;
    const fingerprint = JSON.stringify(payload);
    if (!this.attempt || this.attempt.fingerprint !== fingerprint) this.attempt = { fingerprint, key: crypto.randomUUID() };
    this.pending.set(true); this.failure.set(null);
    try {
      const result = await send(payload, this.attempt.key);
      this.result.set(result); this.attempt = null; return result;
    } catch (error) { this.failure.set(safeFailure(error)); return undefined; }
    finally { this.pending.set(false); }
  }
  clear(): void { if (!this.pending()) { this.attempt = null; this.result.set(null); this.failure.set(null); } }
}
