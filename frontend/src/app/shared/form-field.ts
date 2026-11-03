import { Component, Directive, ElementRef, contentChild, effect, inject, input } from '@angular/core';
import { AbstractControl } from '@angular/forms';

@Directive({ selector: 'input[beFieldControl],textarea[beFieldControl],select[beFieldControl]', standalone: true })
export class FieldControlDirective {
  readonly element = inject<ElementRef<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>>(ElementRef);
}

@Component({
  selector: 'be-form-field', standalone: true,
  template: `<label [for]="controlId()">{{ label() }} @if (required()) { <span>(required)</span> }</label>
    <ng-content />
    @if (hint()) { <p [id]="controlId() + '-hint'">{{ hint() }}</p> }
    @if (error()) { <p class="error" [id]="controlId() + '-error'" role="alert">{{ error() }}</p> }`,
  styles: [`:host { display:block; } label { display:block; margin-bottom:8px; font-weight:500; }
    p { margin:8px 0; } .error { color:#B91C1C; }`],
})
export class FormFieldComponent {
  readonly controlId = input.required<string>();
  readonly label = input.required<string>();
  readonly hint = input('');
  readonly error = input('');
  readonly required = input(false);
  readonly control = contentChild(FieldControlDirective);
  constructor() {
    effect(() => {
      const element = this.control()?.element.nativeElement;
      if (!element) return;
      element.id = this.controlId();
      element.required = this.required();
      element.setAttribute('aria-invalid', this.error() ? 'true' : 'false');
      const ownPrefix = `${this.controlId()}-`;
      const existing = (element.getAttribute('aria-describedby') ?? '').split(/\s+/)
        .filter(id => id && id !== `${ownPrefix}hint` && id !== `${ownPrefix}error`);
      if (this.hint()) existing.push(`${ownPrefix}hint`);
      if (this.error()) existing.push(`${ownPrefix}error`);
      if (existing.length) element.setAttribute('aria-describedby', existing.join(' '));
      else element.removeAttribute('aria-describedby');
    });
  }
}

/** Does not reset values or expose arbitrary server error content. */
export function validationMessage(control: AbstractControl, submitted = false): string {
  if (!control.invalid || !(control.touched || submitted)) return '';
  if (control.hasError('required')) return 'This field is required.';
  if (control.hasError('email')) return 'Enter a valid email address.';
  if (control.hasError('minlength')) return 'This value is too short.';
  if (control.hasError('maxlength')) return 'This value is too long.';
  if (control.hasError('min')) return 'This value is below the minimum.';
  if (control.hasError('max')) return 'This value exceeds the maximum.';
  return 'Check this value.';
}

@Component({
  selector: 'be-validation-summary', standalone: true,
  template: `@if (errors().length) { <div tabindex="-1" role="alert" aria-label="Form errors">
    <h2>Check the following fields</h2><ul>@for (error of errors(); track error.controlId) {
      <li><a [href]="'#' + error.controlId" (click)="focusField($event, error.controlId)">{{ error.message }}</a></li>
    }</ul></div> }`,
})
export class ValidationSummaryComponent {
  readonly errors = input<ReadonlyArray<{ controlId: string; message: string }>>([]);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  /** Call after a rejected submit and change detection, never during typing. */
  focus(): void { this.host.nativeElement.querySelector<HTMLElement>('[role=alert]')?.focus(); }
  focusField(event: Event, id: string): void {
    event.preventDefault();
    this.host.nativeElement.ownerDocument.getElementById(id)?.focus();
  }
}
