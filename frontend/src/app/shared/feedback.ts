import { Component, input, output } from '@angular/core';
import { IconComponent } from './icon';

export type FeedbackKind = 'loading' | 'empty' | 'success' | 'error' | 'warning' | 'info';
@Component({
  selector: 'be-feedback', standalone: true, imports: [IconComponent],
  template: `
    <section [attr.role]="kind() === 'error' ? 'alert' : 'status'" aria-atomic="true"
      [attr.aria-busy]="kind() === 'loading' ? 'true' : null" [attr.data-kind]="kind()">
      <be-icon [name]="kind()" [size]="24" />
      <div><h2>{{ title() }}</h2><p>{{ message() }}</p>
        @if (actionLabel()) { <button type="button" (click)="action.emit()">{{ actionLabel() }}</button> }
      </div>
    </section>`,
  styles: [`
    section { display:flex; gap:16px; padding:24px; border:1px solid #CBD5E1; border-radius:12px; background:#fff; }
    h2 { margin:0; font-size:20px; } p { margin:8px 0; overflow-wrap:anywhere; }
    [data-kind=error] { color:#B91C1C; background:#FEF2F2; }
    [data-kind=success] { color:#166534; background:#DCFCE7; }
    [data-kind=warning] { color:#B45309; background:#FFF7ED; }
    button { min-height:44px; } button:focus-visible { outline:3px solid #1D4ED8; outline-offset:2px; }
  `],
})
export class FeedbackComponent {
  readonly kind = input<FeedbackKind>('info');
  readonly title = input.required<string>();
  readonly message = input('');
  readonly actionLabel = input('');
  readonly action = output<void>();
}
