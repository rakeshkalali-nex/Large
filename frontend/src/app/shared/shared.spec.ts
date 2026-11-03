import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialog, DialogModule } from '@angular/cdk/dialog';
import { OverlayContainer } from '@angular/cdk/overlay';
import { FormControl, Validators } from '@angular/forms';
import { describe, expect, it, vi } from 'vitest';
import { ConfirmationDialogComponent, ConfirmationService } from './confirmation';
import { FeedbackComponent } from './feedback';
import { FieldControlDirective, FormFieldComponent, validationMessage, ValidationSummaryComponent } from './form-field';
import { PendingAction, PendingButtonComponent } from './pending-button';
import { formatCompanyTimestamp, formatCurrencyMinor, formatDateOnly } from './formatting';

@Component({ standalone: true, imports: [FormFieldComponent, FieldControlDirective],
  template: `<be-form-field controlId="name" label="Name" hint="Your full name" [error]="error" [required]="true">
    <input beFieldControl value="Unsaved name" aria-describedby="external" />
  </be-form-field>` })
class FieldHost { error = ''; }

describe('shared accessible controls', () => {
  it('links permanent labels, help and errors without clearing input or stealing focus', () => {
    const fixture = TestBed.createComponent(FieldHost);
    fixture.detectChanges();
    const input = fixture.nativeElement.querySelector('input') as HTMLInputElement;
    expect(input.id).toBe('name');
    expect(fixture.nativeElement.querySelector('label').htmlFor).toBe('name');
    expect(input.required).toBe(true);
    expect(input.getAttribute('aria-describedby')).toBe('external name-hint');
    fixture.componentInstance.error = 'Name is already used.';
    fixture.detectChanges();
    expect(input.getAttribute('aria-invalid')).toBe('true');
    expect(input.getAttribute('aria-describedby')).toBe('external name-hint name-error');
    expect(input.value).toBe('Unsaved name');
    fixture.componentInstance.error = '';
    fixture.detectChanges();
    expect(input.getAttribute('aria-describedby')).toBe('external name-hint');
  });

  it('shows safe validation only after touch or submit and retains values', () => {
    const control = new FormControl('', Validators.required);
    expect(validationMessage(control)).toBe('');
    expect(validationMessage(control, true)).toBe('This field is required.');
    control.setValue('safe unsaved value');
    control.setErrors({ backend: 'private stack trace' });
    control.markAsTouched();
    expect(validationMessage(control)).toBe('Check this value.');
    expect(control.value).toBe('safe unsaved value');
  });

  it('focuses the validation summary only when explicitly requested', () => {
    const fixture = TestBed.createComponent(ValidationSummaryComponent);
    fixture.componentRef.setInput('errors', [{ controlId: 'field', message: 'Required' }]);
    fixture.detectChanges();
    fixture.componentInstance.focus();
    expect(document.activeElement).toBe(fixture.nativeElement.querySelector('[role=alert]'));
  });

  it.each(['loading', 'empty', 'success', 'error', 'warning', 'info'])('announces %s and retries only deliberately', kind => {
    const fixture = TestBed.createComponent(FeedbackComponent);
    fixture.componentRef.setInput('kind', kind);
    fixture.componentRef.setInput('title', 'State description');
    fixture.componentRef.setInput('actionLabel', 'Retry');
    const action = vi.fn();
    fixture.componentInstance.action.subscribe(action);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('section').getAttribute('role')).toBe(kind === 'error' ? 'alert' : 'status');
    expect(action).not.toHaveBeenCalled();
    fixture.nativeElement.querySelector('button').click();
    expect(action).toHaveBeenCalledTimes(1);
  });

  it('blocks pending and disabled button activation with visible pending text', () => {
    const fixture = TestBed.createComponent(PendingButtonComponent);
    fixture.componentRef.setInput('label', 'Save');
    const action = vi.fn();
    fixture.componentInstance.activated.subscribe(action);
    fixture.componentRef.setInput('pending', true);
    fixture.detectChanges();
    const button = fixture.nativeElement.querySelector('button');
    button.click();
    fixture.componentInstance.activate();
    expect(action).not.toHaveBeenCalled();
    expect(button.disabled).toBe(true);
    expect(button.getAttribute('aria-busy')).toBe('true');
    expect(button.textContent).toContain('Saving');
    fixture.componentRef.setInput('pending', false);
    fixture.detectChanges();
    button.click();
    expect(action).toHaveBeenCalledTimes(1);
  });

  it('blocks simultaneous mutations and unlocks after failure without retry', async () => {
    const pending = new PendingAction();
    let reject!: (reason: Error) => void;
    const work = vi.fn(() => new Promise<void>((_, fail) => { reject = fail; }));
    const first = pending.run(work);
    expect(pending.pending).toBe(true);
    await pending.run(work);
    expect(work).toHaveBeenCalledTimes(1);
    reject(new Error('Offline'));
    await expect(first).rejects.toThrow('Offline');
    expect(pending.pending).toBe(false);
    expect(work).toHaveBeenCalledTimes(1);
  });

  it('requires a nonblank reason, traps focus and defaults to safe cancel', async () => {
    TestBed.configureTestingModule({ imports: [DialogModule] });
    const service = TestBed.inject(ConfirmationService);
    const results = vi.fn();
    service.confirm({ title: 'Reject record', recordName: 'PO-100', message: 'Reject', requireReason: true }).subscribe(results);
    const dialog = TestBed.inject(Dialog).openDialogs[0];
    const instance = dialog.componentInstance as ConfirmationDialogComponent;
    await new Promise(resolve => setTimeout(resolve, 20));
    const overlay = TestBed.inject(OverlayContainer).getContainerElement();
    expect(overlay.querySelector('[role=alertdialog]')?.getAttribute('aria-labelledby')).toBeTruthy();
    expect(overlay.querySelectorAll('.cdk-focus-trap-anchor').length).toBe(2);
    expect(document.activeElement?.textContent).toBe('Cancel');
    instance.reason = '   ';
    instance.confirm();
    expect(results).not.toHaveBeenCalled();
    expect(instance.invalid).toBe(true);
    instance.reason = '  Incorrect quantity  ';
    instance.confirm();
    expect(results).toHaveBeenCalledWith({ confirmed: true, reason: 'Incorrect quantity' });
  });

  it('maps Escape dismissal to cancellation and restores trigger focus', async () => {
    TestBed.configureTestingModule({ imports: [DialogModule] });
    const trigger = document.createElement('button');
    document.body.append(trigger);
    trigger.focus();
    try {
      const result = vi.fn();
      TestBed.inject(ConfirmationService).confirm({ title: 'Archive', recordName: 'Project', message: 'Archive record?' }).subscribe(result);
      await new Promise(resolve => setTimeout(resolve, 20));
      document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', keyCode: 27, bubbles: true }));
      await new Promise(resolve => setTimeout(resolve, 20));
      expect(result).toHaveBeenCalledWith({ confirmed: false });
      expect(document.activeElement).toBe(trigger);
    } finally { trigger.remove(); }
  });
});

describe('shared exact formatting', () => {
  const config = { timezone: 'Asia/Kolkata', currencyCode: 'USD', currencyMinorDigits: 2, locale: 'en-US' };
  it('preserves values above safe integers and fractional negative amounts', () => {
    expect(formatCurrencyMinor('9223372036854775807', config)).toBe('$92,233,720,368,547,758.07');
    expect(formatCurrencyMinor('-1', config)).toBe('-$0.01');
    expect(formatCurrencyMinor('0', config)).toBe('$0.00');
    expect(formatCurrencyMinor('1234', { ...config, currencyCode: 'JPY', currencyMinorDigits: 0 })).toBe('¥1,234');
    expect(formatCurrencyMinor('1234', { ...config, currencyCode: 'KWD', currencyMinorDigits: 3 })).toContain('1.234');
  });
  it('rejects floating point/exponent strings and missing company configuration', () => {
    for (const value of ['1.2', '1e5', '', 'NaN']) expect(() => formatCurrencyMinor(value, config)).toThrow();
    expect(() => formatCurrencyMinor('1', { ...config, currencyCode: '' })).toThrow();
  });
  it('keeps date-only values stable and rejects normalized invalid dates', () => {
    expect(formatDateOnly('2024-02-29', 'en-US')).toBe('Feb 29, 2024');
    expect(() => formatDateOnly('2025-02-29')).toThrow();
    expect(() => formatDateOnly('2025-01-01T00:00:00Z')).toThrow();
  });
  it('uses company timezone across midnight and rejects timezone-free timestamps', () => {
    expect(formatCompanyTimestamp('2026-01-01T23:30:00Z', config)).toContain('Jan 2, 2026');
    expect(formatCompanyTimestamp('2026-01-01T23:30:00Z', { timezone: 'America/Los_Angeles', locale: 'en-US' })).toContain('Jan 1, 2026');
    expect(() => formatCompanyTimestamp('2026-01-01T23:30:00', config)).toThrow();
  });
});
