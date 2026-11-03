import { Component, inject, signal, DestroyRef } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { JsonPipe } from '@angular/common';
import { SettingsApi, Settings, SettingsInput, HistoryEntry, OperatorStatus } from './settings-api';
import { MutationState } from '../../../core/http/mutation-state';
import { safeFailure } from '../../../core/http/safe-error';
import { StewardshipPanel } from './stewardship-panel';
import { UiEvents } from '../../../core/state/ui-events';

@Component({
  standalone: true, selector: 'be-settings-page', imports: [ReactiveFormsModule, JsonPipe, StewardshipPanel],
  templateUrl: './settings-page.html', styleUrl: './settings-page.scss'
})
export class SettingsPage {
  private readonly api = inject(SettingsApi);
  private readonly fb = inject(FormBuilder).nonNullable;
  readonly record = signal<Settings | null>(null);
  readonly history = signal<HistoryEntry[]>([]);
  readonly historyOffset = signal(0);
  readonly operator = signal<OperatorStatus | null>(null);
  readonly error = signal('');
  readonly auxiliaryError = signal('');
  readonly loading = signal(false);
  readonly saved = signal(false);
  readonly comparison = signal<Settings | null>(null);
  readonly mutation = new MutationState<SettingsInput, Settings>();
  readonly form = this.fb.group({
    companyName: ['', [Validators.required, Validators.maxLength(200)]], wordmark: ['', [Validators.required, Validators.maxLength(100)]],
    logoDocumentId: [''], currencyCode: ['', [Validators.required, Validators.pattern(/^[A-Z]{3}$/)]],
    currencyMinorDigits: [2, [Validators.required, Validators.min(0), Validators.max(4), Validators.pattern(/^\d$/)]],
    timezone: ['UTC', Validators.required], timezoneConfirmed: [false, Validators.requiredTrue],
    recoveryContactLabel: ['', Validators.required], recoveryInstructions: ['', Validators.required],
    reservationHours: [48, [Validators.required, Validators.min(1), Validators.max(8760), Validators.pattern(/^\d+$/)]],
    countries: ['', [Validators.required, Validators.pattern(/^[A-Z]{2}(\s*,\s*[A-Z]{2})*$/)]],
    noticeVersion: ['', Validators.required], retentionScheduleVersion: ['', Validators.required], confirmOperationalPolicy: [false],
    propertyTypes: [''], referenceValues: this.fb.array<ReturnType<SettingsPage['referenceForm']>>([]),
    inspectionChecklists: this.fb.array<ReturnType<SettingsPage['checklistForm']>>([])
  });
  constructor() {
    inject(UiEvents).events.pipe(takeUntilDestroyed(inject(DestroyRef))).subscribe(event => {
      if (event.name === 'session.changed' && event.payload.status !== 'authenticated') {
        this.record.set(null); this.history.set([]); this.operator.set(null); this.comparison.set(null);
        this.error.set('Your session has ended. Sign in again before viewing company settings.');
      }
    });
    void this.load();
  }
  referenceForm(key = '', value = '', label = '') { return this.fb.group({key: [key, Validators.required], value: [value, Validators.required], label: [label, Validators.required]}); }
  checklistForm(id = '', name = '', items = '') { return this.fb.group({id: [id], name: [name, Validators.required], items: [items, Validators.required]}); }
  addReference(): void { this.form.controls.referenceValues.push(this.referenceForm()); }
  addChecklist(): void { this.form.controls.inspectionChecklists.push(this.checklistForm()); }
  consentRequired(): boolean { return this.record()?.referenceValues.find(v => v.key === 'privacy.consent-required')?.value !== 'false'; }
  async load(): Promise<void> {
    if (this.loading()) return;
    this.loading.set(true); this.error.set('');
    try { this.accept(await this.api.get<Settings>('settings')); await this.refreshAuxiliary(); }
    catch (error) { this.error.set(safeFailure(error).message); }
    finally { this.loading.set(false); }
  }
  async compare(): Promise<void> {
    try { this.comparison.set(await this.api.get<Settings>('settings')); }
    catch (error) { this.error.set(safeFailure(error).message); }
  }
  private accept(value: Settings): void {
    this.record.set(value);
    this.form.patchValue({ companyName: value.companyName, timezone: value.timezone, timezoneConfirmed: value.timezoneConfirmed,
      reservationHours: value.reservationHours, recoveryContactLabel: value.recoveryContactLabel, recoveryInstructions: value.recoveryInstructions,
      wordmark: value.branding.wordmark, logoDocumentId: value.branding.logoDocumentId ?? '',
      currencyCode: value.currencyCode ?? '', currencyMinorDigits: value.currencyMinorDigits ?? 2,
      countries: value.operatingPolicy.countries.join(', '), noticeVersion: value.operatingPolicy.noticeVersion ?? '',
      retentionScheduleVersion: value.operatingPolicy.retentionScheduleVersion ?? '', confirmOperationalPolicy: false,
      propertyTypes: value.propertyTypes.join('\n') });
    this.form.controls.referenceValues.clear();
    value.referenceValues.forEach(v => this.form.controls.referenceValues.push(this.referenceForm(v.key, v.value, v.label)));
    this.form.controls.inspectionChecklists.clear();
    value.inspectionChecklists.forEach(v => this.form.controls.inspectionChecklists.push(this.checklistForm(v.id, v.name, v.items.join('\n'))));
    for (const control of [this.form.controls.currencyCode, this.form.controls.currencyMinorDigits]) value.currencyLocked ? control.disable() : control.enable();
    this.form.markAsPristine(); this.comparison.set(null);
  }
  async refreshAuxiliary(): Promise<void> {
    this.auxiliaryError.set('');
    const results = await Promise.allSettled([this.api.get<HistoryEntry[]>(`settings/history?limit=100&offset=${this.historyOffset()}`), this.api.get<OperatorStatus>('operator/status')]);
    if (results[0].status === 'fulfilled') this.history.set(results[0].value); else this.auxiliaryError.set('Change history could not be loaded.');
    if (results[1].status === 'fulfilled') this.operator.set(results[1].value); else this.auxiliaryError.update(v => `${v} Operator status could not be loaded.`.trim());
  }
  async historyPage(direction: number): Promise<void> { this.historyOffset.update(v => Math.max(0, v + direction * 100)); await this.refreshAuxiliary(); }
  async save(): Promise<void> {
    this.saved.set(false); this.error.set('');
    if (this.form.invalid || !this.record()) { this.form.markAllAsTouched(); this.error.set('Check the company, currency, timezone and policy fields. Required confirmations must be explicit.'); return; }
    const v = this.form.getRawValue();
    const payload: SettingsInput = {version: this.record()!.version, companyName: v.companyName, branding: {wordmark: v.wordmark, logoDocumentId: v.logoDocumentId || null},
      currencyCode: v.currencyCode, currencyMinorDigits: v.currencyMinorDigits, timezone: v.timezone, timezoneConfirmed: v.timezoneConfirmed,
      recoveryContactLabel: v.recoveryContactLabel, recoveryInstructions: v.recoveryInstructions, reservationHours: v.reservationHours,
      propertyTypes: lines(v.propertyTypes), referenceValues: v.referenceValues,
      inspectionChecklists: v.inspectionChecklists.map(c => ({...c, items: lines(c.items)})),
      operatingPolicy: {countries: v.countries.split(',').map(c => c.trim()), noticeVersion: v.noticeVersion, retentionScheduleVersion: v.retentionScheduleVersion},
      confirmOperationalPolicy: v.confirmOperationalPolicy};
    const result = await this.mutation.submit(payload, (body, key) => this.api.mutate<Settings>('PUT', 'settings', body, key));
    if (result) { this.accept(result); this.saved.set(true); await this.refreshAuxiliary(); }
  }
}
function lines(value: string): string[] { return value.split('\n').map(v => v.trim()).filter(Boolean); }
