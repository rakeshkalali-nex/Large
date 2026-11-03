import { Component, input, output } from '@angular/core';

@Component({
  selector: 'be-pending-button', standalone: true,
  template: `<button type="button" [disabled]="pending() || disabled()" [attr.aria-busy]="pending()"
    [attr.data-variant]="variant()" (click)="activate()">
    @if (pending()) { <span class="spinner" aria-hidden="true"></span> }
    {{ pending() ? pendingLabel() : label() }}
  </button>`,
  styles: [`
    :host { display:inline-block; } button { display:inline-flex; align-items:center; justify-content:center; gap:8px;
      min-height:44px; padding:8px 16px; border:1px solid transparent; border-radius:8px; background:#0F766E; color:#fff; }
    button[data-variant=primary]:hover:not(:disabled) { background:#115E59; }
    button[data-variant=primary]:active:not(:disabled) { background:#134E4A; }
    button[data-variant=secondary] { background:#fff; color:#243447; border-color:#526170; }
    button[data-variant=secondary]:hover:not(:disabled) { background:#F5F6F3; }
    button[data-variant=secondary]:active:not(:disabled) { background:#E2E8F0; }
    button[data-variant=destructive] { background:#B91C1C; }
    button[data-variant=destructive]:hover:not(:disabled) { background:#991B1B; }
    button[data-variant=destructive]:active:not(:disabled) { background:#7F1D1D; }
    button:active:not(:disabled) { box-shadow:inset 0 0 0 2px currentColor; }
    button:disabled { background:#E2E8F0; color:#526170; cursor:not-allowed; }
    button:focus-visible { outline:3px solid #1D4ED8; outline-offset:2px; }
    .spinner { width:16px; height:16px; border:2px solid currentColor; border-right-color:transparent; border-radius:50%; animation:spin 1s linear infinite; }
    @keyframes spin { to { transform:rotate(360deg); } }
    @media(prefers-reduced-motion:reduce) { .spinner { animation:none; } }
    @media(max-width:480px) { :host, button { width:100%; } }
  `],
})
export class PendingButtonComponent {
  readonly label = input.required<string>();
  readonly pendingLabel = input('Saving…');
  readonly pending = input(false);
  readonly disabled = input(false);
  readonly variant = input<'primary' | 'secondary' | 'destructive'>('primary');
  readonly activated = output<void>();
  activate(): void { if (!this.pending() && !this.disabled()) this.activated.emit(); }
}

/** Wrap the actual async mutation, including form-submit paths, to reject concurrent attempts. */
export class PendingAction {
  pending = false;
  async run<T>(work: () => Promise<T>): Promise<T | undefined> {
    if (this.pending) return undefined;
    this.pending = true;
    try { return await work(); } finally { this.pending = false; }
  }
}
