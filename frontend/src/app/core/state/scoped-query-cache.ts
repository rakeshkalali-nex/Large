import { inject, Injectable } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { CanDeactivateFn } from '@angular/router';
import { UiEvents } from './ui-events';
import { UnsavedChanges } from './project-context';

/** Optional transient read cache; changes invalidate it, never update business values optimistically. */
@Injectable({ providedIn: 'root' })
export class ScopedQueryCache {
  private readonly values = new Map<string, unknown>();
  private generation = 0;
  constructor() {
    inject(UiEvents).events.pipe(takeUntilDestroyed()).subscribe(() => this.clear());
  }
  get<T>(key: string): T | undefined { return this.values.get(key) as T | undefined; }
  snapshot(): number { return this.generation; }
  put<T>(key: string, value: T, generation: number): void {
    if (generation === this.generation) this.values.set(key, value);
  }
  clear(): void { this.values.clear(); this.generation++; }
}

/** Feature editor routes attach this guard and unregister their confirmation on destroy. */
export const unsavedChangesGuard: CanDeactivateFn<unknown> = () => inject(UnsavedChanges).canLeave();
