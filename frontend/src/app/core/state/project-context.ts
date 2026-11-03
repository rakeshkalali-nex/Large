import { inject, Injectable, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { UiEvents } from './ui-events';
import { SessionStore } from './session-store';

/** Feature forms register a confirmation callback; returning false preserves their values. */
@Injectable({ providedIn: 'root' })
export class UnsavedChanges {
  private readonly guards = new Set<() => boolean | Promise<boolean>>();
  register(guard: () => boolean | Promise<boolean>): () => void {
    this.guards.add(guard);
    return () => this.guards.delete(guard);
  }
  async canLeave(): Promise<boolean> {
    for (const guard of this.guards) if (!await guard()) return false;
    return true;
  }
}

@Injectable({ providedIn: 'root' })
export class ProjectContext {
  private readonly events = inject(UiEvents);
  private readonly session = inject(SessionStore);
  private readonly unsaved = inject(UnsavedChanges);
  private readonly value = signal({ projectId: null as string | null, revision: 0 });
  readonly context = this.value.asReadonly();
  private changing = false;
  constructor() {
    this.events.events.pipe(takeUntilDestroyed()).subscribe(event => {
      if (event.name === 'session.changed') this.set(null);
    });
  }
  async change(projectId: string | null): Promise<boolean> {
    if (this.changing || (projectId !== null && !this.session.projectIds().includes(projectId))) return false;
    if (projectId === this.context().projectId) return true;
    this.changing = true;
    const revision = this.context().revision;
    try {
      if (!await this.unsaved.canLeave() || revision !== this.context().revision) return false;
      this.set(projectId); return true;
    } finally { this.changing = false; }
  }
  effectiveProject(routeProjectId?: string | null): string | null {
    return routeProjectId ?? this.context().projectId;
  }
  private set(projectId: string | null): void {
    this.value.set({ projectId, revision: this.context().revision + 1 });
    this.events.emit({ name: 'project.changed', payload: this.context() });
  }
}
