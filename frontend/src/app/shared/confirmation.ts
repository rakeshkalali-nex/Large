import { Component, Injectable, inject } from '@angular/core';
import { Dialog, DialogModule, DialogRef, DIALOG_DATA } from '@angular/cdk/dialog';
import { FormsModule } from '@angular/forms';
import { Observable, map } from 'rxjs';

export interface ConfirmationOptions {
  title: string;
  recordName: string;
  message: string;
  confirmLabel?: string;
  requireReason?: boolean;
  destructive?: boolean;
}
export type ConfirmationResult = { confirmed: false } | { confirmed: true; reason?: string };
let nextDialogId = 0;

@Component({
  selector: 'be-confirmation-dialog',
  standalone: true,
  imports: [FormsModule],
  template: `
    <h2 [id]="data.headingId">{{ data.title }}</h2>
    <p [id]="data.descriptionId">{{ data.message }} <strong>{{ data.recordName }}</strong></p>
    <form (ngSubmit)="confirm()">
      @if (data.requireReason) {
        <label [for]="data.reasonId">Reason (required)</label>
        <textarea [id]="data.reasonId" name="reason" [(ngModel)]="reason" required
          [attr.aria-invalid]="invalid ? 'true' : null"
          [attr.aria-describedby]="invalid ? data.reasonId + '-error' : null"></textarea>
        @if (invalid) { <p [id]="data.reasonId + '-error'" role="alert">Enter a reason before confirming.</p> }
      }
      <div class="actions">
        <button type="button" data-cancel (click)="cancel()">Cancel</button>
        <button type="submit" [class.destructive]="data.destructive">{{ data.confirmLabel || 'Confirm' }}</button>
      </div>
    </form>
  `,
  styles: [`
    :host { display:block; box-sizing:border-box; padding:24px; background:#fff; color:#243447;
      border-radius:12px; max-height:85dvh; overflow:auto; box-shadow:0 2px 8px #14283F0D; }
    h2 { margin:0 0 16px; font-size:24px; } p { overflow-wrap:anywhere; }
    label { display:block; margin-bottom:8px; } textarea { box-sizing:border-box; width:100%; min-height:96px; }
    .actions { display:flex; flex-wrap:wrap; gap:8px; justify-content:flex-end; margin-top:24px; }
    button { min-height:44px; padding:8px 16px; } .destructive { background:#B91C1C; color:#fff; }
    [role=alert] { color:#B91C1C; } :focus-visible { outline:3px solid #1D4ED8; outline-offset:2px; }
    @media(max-width:480px) { .actions button { width:100%; } }
  `],
})
export class ConfirmationDialogComponent {
  readonly data = inject<ConfirmationOptions & { headingId: string; descriptionId: string; reasonId: string }>(DIALOG_DATA);
  private readonly ref = inject<DialogRef<ConfirmationResult>>(DialogRef);
  reason = '';
  invalid = false;
  confirm(): void {
    this.invalid = !!this.data.requireReason && !this.reason.trim();
    if (this.invalid) return;
    this.ref.close(this.data.requireReason ? { confirmed: true, reason: this.reason.trim() } : { confirmed: true });
  }
  cancel(): void { this.ref.close({ confirmed: false }); }
}

@Injectable({ providedIn: 'root' })
export class ConfirmationService {
  private readonly dialog = inject(Dialog);
  /** Confirmation only: callers start mutations after a confirmed result, never inside the overlay. */
  confirm(options: ConfirmationOptions): Observable<ConfirmationResult> {
    const id = `be-confirm-${++nextDialogId}`;
    return this.dialog.open<ConfirmationResult>(ConfirmationDialogComponent, {
      data: { ...options, headingId: `${id}-title`, descriptionId: `${id}-description`, reasonId: `${id}-reason` },
      role: 'alertdialog', ariaLabelledBy: `${id}-title`, ariaDescribedBy: `${id}-description`,
      autoFocus: '[data-cancel]', restoreFocus: true, disableClose: false,
      width: 'min(480px, calc(100vw - 32px))', maxWidth: 'calc(100vw - 32px)',
    }).closed.pipe(map(result => result ?? { confirmed: false }));
  }
}

// Import this module in standalone consumers (or provide Dialog at the application root).
export { DialogModule };
