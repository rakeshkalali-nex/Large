import { Component, DestroyRef, ElementRef, inject, signal, ViewChild } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Subscription } from 'rxjs';
import { UiEvents } from '../../../core/state/ui-events';
import { SessionStore } from '../../../core/state/session-store';
import { ProjectContext } from '../../../core/state/project-context';
import { AuditApi, AuditEvent, AuditQuery } from './audit-api';

@Component({
  standalone: true, selector: 'be-audit-page', imports: [ReactiveFormsModule, RouterLink, DatePipe],
  templateUrl: './audit-page.html', styleUrl: './audit-page.scss'
})
export class AuditPage {
  private readonly api = inject(AuditApi);
  readonly session = inject(SessionStore);
  private readonly project = inject(ProjectContext);
  private readonly destroy = inject(DestroyRef);
  @ViewChild('drawer', { static: true }) drawer!: ElementRef<HTMLDialogElement>;
  private listRequest?: Subscription;
  private detailRequest?: Subscription;
  private opener?: HTMLElement;
  private detailId = '';
  private query: AuditQuery = {};
  private cursor: string | undefined;
  readonly events = signal<AuditEvent[]>([]);
  readonly nextCursor = signal<string | null>(null);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly denied = signal(false);
  readonly validation = signal('');
  readonly selected = signal<AuditEvent | null>(null);
  readonly detailLoading = signal(false);
  readonly detailError = signal('');
  readonly detailRetry = signal(false);
  readonly form = inject(FormBuilder).nonNullable.group({
    q: '', actorId: '', action: '', projectId: this.project.context().projectId ?? '', recordType: '', recordId: '',
    from: '', to: '', direction: 'desc' as 'desc' | 'asc', limit: 25
  });

  constructor() {
    inject(UiEvents).events.pipe(takeUntilDestroyed()).subscribe(event => {
      if (event.name === 'session.changed') {
        this.clearScope(); this.form.reset();
        if (event.payload.status === 'authenticated') this.apply();
        else this.error.set('Your session has ended. Sign in again to view audit history.');
      } else if (event.name === 'project.changed') {
        this.clearScope(); this.form.controls.projectId.setValue(event.payload.projectId ?? '');
        if (this.session.status() === 'authenticated') this.apply();
      }
    });
    this.destroy.onDestroy(() => { this.listRequest?.unsubscribe(); this.detailRequest?.unsubscribe(); });
    if (this.session.status() === 'authenticated') this.apply();
  }

  apply(): void {
    if (this.session.status() !== 'authenticated') return;
    const value = this.form.getRawValue();
    this.validation.set('');
    const from = this.utc(value.from), to = this.utc(value.to);
    if (from === null || to === null || (from && to && from > to)) {
      this.validation.set('Enter valid UTC dates with From no later than To.'); return;
    }
    if (!Number.isInteger(value.limit) || value.limit < 1 || value.limit > 100) {
      this.validation.set('Page size must be a whole number from 1 to 100.'); return;
    }
    this.query = { ...value, q: value.q.trim(), actorId: value.actorId.trim(), action: value.action.trim(),
      projectId: value.projectId.trim(), recordType: value.recordType.trim(), recordId: value.recordId.trim(),
      from, to, sort: 'occurredAt' };
    this.cursor = undefined; this.closeDetails(); this.load();
  }
  reset(): void { this.form.reset(); this.form.controls.projectId.setValue(this.project.context().projectId ?? ''); this.apply(); }
  next(): void {
    const cursor = this.nextCursor();
    if (!cursor || this.loading()) return;
    this.cursor = cursor; this.closeDetails(); this.load();
  }
  load(): void {
    if (this.session.status() !== 'authenticated') return;
    this.listRequest?.unsubscribe(); this.events.set([]); this.nextCursor.set(null);
    this.loading.set(true); this.error.set(''); this.denied.set(false);
    this.listRequest = this.api.list({ ...this.query, cursor: this.cursor }).subscribe({
      next: result => { this.events.set(result.data); this.nextCursor.set(result.page.nextCursor); this.loading.set(false); },
      error: error => { this.loading.set(false); this.denied.set(this.isDenied(error)); this.error.set(this.failure(error)); }
    });
  }
  openDetails(event: AuditEvent, opener: HTMLElement): void {
    this.opener = opener; this.detailId = event.id;
    this.drawer.nativeElement.showModal(); this.loadDetail();
  }
  loadDetail(): void {
    this.detailRequest?.unsubscribe(); this.selected.set(null); this.detailLoading.set(true);
    this.detailError.set(''); this.detailRetry.set(false);
    this.detailRequest = this.api.detail(this.detailId).subscribe({
      next: result => { this.selected.set(result.data); this.detailLoading.set(false); },
      error: error => { this.detailLoading.set(false); this.detailError.set(this.failure(error)); this.detailRetry.set(!this.isDenied(error) && error?.status !== 404); }
    });
  }
  closeDetails(): void {
    this.detailRequest?.unsubscribe(); this.selected.set(null); this.detailLoading.set(false); this.detailError.set('');
    if (this.drawer?.nativeElement.open) this.drawer.nativeElement.close();
    this.opener?.focus(); this.opener = undefined;
  }
  safeLink(link: string | null): string | null {
    return link && /^\/(?!\/)[a-zA-Z0-9/_?=&.%#-]*$/.test(link) ? link : null;
  }
  private clearScope(): void {
    this.listRequest?.unsubscribe(); this.closeDetails(); this.events.set([]); this.nextCursor.set(null);
    this.loading.set(false); this.error.set(''); this.denied.set(false); this.cursor = undefined;
  }
  private utc(value: string): string | undefined | null {
    if (!value) return undefined;
    if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d{1,3})?)?$/.test(value)) return null;
    const date = new Date(`${value}Z`);
    return Number.isNaN(date.getTime()) ? null : date.toISOString();
  }
  private isDenied(error: unknown): boolean { return error instanceof HttpErrorResponse && (error.status === 401 || error.status === 403); }
  private failure(error: unknown): string {
    const status = error instanceof HttpErrorResponse ? error.status : 0;
    if (status === 401) return 'Your session has ended. Sign in again to view audit history.';
    if (status === 403) return 'Access denied. Your current permissions do not allow this audit trail.';
    if (status === 404) return 'This audit event was not found or is no longer available to you.';
    if (status === 400 || status === 422) return 'The audit filters or cursor were not accepted. Adjust the filters and search again.';
    return 'Audit history could not be loaded. Check your connection and retry.';
  }
}
