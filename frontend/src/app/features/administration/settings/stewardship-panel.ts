import { Component, inject, signal, input, effect } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { SettingsApi, PrivacyRequest, Hold, RetentionReview, Overrides } from './settings-api';
import { MutationState } from '../../../core/http/mutation-state';
import { safeFailure } from '../../../core/http/safe-error';

@Component({standalone: true, selector: 'be-stewardship-panel', imports: [ReactiveFormsModule], templateUrl: './stewardship-panel.html', styleUrl: './settings-page.scss'})
export class StewardshipPanel {
  private readonly api = inject(SettingsApi);
  private readonly fb = inject(FormBuilder).nonNullable;
  readonly mutation = new MutationState<{path: string; body: unknown; method: 'POST' | 'PUT'}, unknown>();
  readonly error = signal(''); readonly success = signal('');
  readonly requests = signal<PrivacyRequest[]>([]); readonly holds = signal<Hold[]>([]); readonly reviews = signal<RetentionReview[]>([]);
  readonly consentRequired = input(true);
  readonly offsets = signal<Record<string, number>>({'privacy-requests': 0, 'retention-holds': 0, 'retention-reviews': 0});
  readonly selectedRequest = signal<PrivacyRequest | null>(null);
  readonly override = signal<Overrides | null>(null);
  readonly privacy = this.fb.group({subjectType: ['lead', Validators.required], subjectId: ['', Validators.required], kind: ['access', Validators.required], reason: ['', Validators.required]});
  readonly review = this.fb.group({decision: ['approve', Validators.required], reason: ['', Validators.required], name: [''], email: [''], phone: ['']});
  readonly hold = this.fb.group({recordType: ['lead', Validators.required], recordId: ['', Validators.required], reason: ['', Validators.required], reviewAt: ['']});
  readonly retention = this.fb.group({asOfDate: ['', Validators.required], projectId: [''], reason: ['', Validators.required]});
  readonly acknowledgment = this.fb.group({subjectType: ['lead', Validators.required], subjectId: ['', Validators.required], noticeVersion: ['', Validators.required], acknowledgedAt: ['', Validators.required], consentRequired: [false], consentGiven: [false], recordedByReason: ['', Validators.required]});
  readonly overrides = this.fb.group({projectId: ['', Validators.required], reservationHours: [48, [Validators.min(1), Validators.pattern(/^\d+$/)]], inspectionChecklistIds: [''],
    stockMinimums: this.fb.array<ReturnType<StewardshipPanel['minimumForm']>>([]), referenceValues: this.fb.array<ReturnType<StewardshipPanel['referenceForm']>>([])});
  readonly execution = this.fb.group({confirmed: [false, Validators.requiredTrue]});
  constructor() { effect(() => this.acknowledgment.controls.consentRequired.setValue(this.consentRequired())); void this.refresh(); }
  minimumForm(siteId = '', materialId = '', minimumQuantity = '0') { return this.fb.group({siteId: [siteId], materialId: [materialId, Validators.required], minimumQuantity: [minimumQuantity, [Validators.required, Validators.pattern(/^\d+(\.\d{1,3})?$/)]]}); }
  referenceForm(key = '', value = '') { return this.fb.group({key: [key, Validators.required], value: [value, Validators.required]}); }
  async refresh(): Promise<void> {
    this.error.set('');
    try { const [requests, holds, reviews] = await Promise.all([this.api.get<PrivacyRequest[]>(this.listPath('privacy-requests')), this.api.get<Hold[]>(this.listPath('retention-holds')), this.api.get<RetentionReview[]>(this.listPath('retention-reviews'))]); this.requests.set(requests); this.holds.set(holds); this.reviews.set(reviews); }
    catch (error) { this.error.set(safeFailure(error).message); }
  }
  private listPath(resource: string): string { return `${resource}?limit=100&offset=${this.offsets()[resource]}`; }
  async changePage(resource: string, direction: number): Promise<void> {
    this.offsets.update(v => ({...v, [resource]: Math.max(0, v[resource] + direction * 100)})); await this.refresh();
  }
  private valid(form: {invalid: boolean; markAllAsTouched(): void}): boolean { if (!form.invalid) return true; form.markAllAsTouched(); this.error.set('Complete the required fields before submitting.'); return false; }
  async send(path: string, body: unknown, method: 'POST' | 'PUT' = 'POST'): Promise<unknown> {
    this.error.set(''); this.success.set('');
    const result = await this.mutation.submit({path, body, method}, (p, key) => this.api.mutate(p.method, p.path, p.body, key));
    if (result) { this.success.set('Action saved.'); await this.refresh(); }
    return result;
  }
  async createRequest(): Promise<void> { if (this.valid(this.privacy)) await this.send('privacy-requests', this.privacy.getRawValue()); }
  selectRequest(request: PrivacyRequest): void { this.selectedRequest.set(request); this.review.reset({decision: 'approve', reason: '', name: '', email: '', phone: ''}); }
  async reviewRequest(): Promise<void> {
    const request = this.selectedRequest(); if (!request || !this.valid(this.review)) return;
    const v = this.review.getRawValue(); const corrections = Object.fromEntries(Object.entries({name: v.name, email: v.email, phone: v.phone}).filter(([, value]) => value !== ''));
    const body = {version: request.version, decision: v.decision, reason: v.reason, ...(request.kind === 'correction' && v.decision === 'approve' ? {corrections} : {})};
    if (await this.send(`privacy-requests/${encodeURIComponent(request.id)}/review`, body)) this.selectedRequest.set(null);
  }
  async executeRequest(request: PrivacyRequest): Promise<void> { if (this.valid(this.execution)) { await this.send(`privacy-requests/${encodeURIComponent(request.id)}/execute`, {version: request.version}); this.execution.reset(); } }
  async createHold(): Promise<void> { if (!this.valid(this.hold)) return; const v = this.hold.getRawValue(); await this.send('retention-holds', {...v, reviewAt: v.reviewAt ? new Date(v.reviewAt).toISOString() : null}); }
  async releaseHold(hold: Hold): Promise<void> { if (this.valid(this.execution) && this.hold.controls.reason.valid) { await this.send(`retention-holds/${encodeURIComponent(hold.id)}/release`, {version: hold.version, reason: this.hold.controls.reason.value}); this.execution.reset(); } else this.error.set('Confirm the action and enter the hold release reason.'); }
  async createRetentionReview(): Promise<void> { if (this.retention.controls.asOfDate.invalid) { this.error.set('Choose a retention review date.'); return; } const v = this.retention.getRawValue(); await this.send('retention-reviews', {asOfDate: v.asOfDate, projectId: v.projectId || null}); }
  async executeRetention(review: RetentionReview): Promise<void> {
    if (!this.valid(this.execution)) return;
    if (this.retention.controls.reason.invalid) { this.error.set('Enter an execution reason.'); return; }
    await this.send(`retention-reviews/${encodeURIComponent(review.id)}/execute`, {version: review.version, recordIds: review.candidates.filter(c => !c.holdIds.length).map(c => c.recordId), reason: this.retention.controls.reason.value}); this.execution.reset();
  }
  async recordAcknowledgment(): Promise<void> { if (!this.valid(this.acknowledgment)) return; const v = this.acknowledgment.getRawValue(); await this.send('policy-acknowledgments', {...v, acknowledgedAt: new Date(v.acknowledgedAt).toISOString(), consentGiven: v.consentRequired ? v.consentGiven : null}); }
  async loadOverrides(): Promise<void> {
    if (!this.overrides.controls.projectId.valid) { this.error.set('Enter a project ID.'); return; }
    try {
      const value = await this.api.get<Overrides>(`projects/${encodeURIComponent(this.overrides.controls.projectId.value)}/overrides`);
      this.override.set(value); this.overrides.patchValue({reservationHours: value.reservationHours ?? 48, inspectionChecklistIds: value.inspectionChecklistIds.join('\n')});
      this.overrides.controls.stockMinimums.clear(); value.stockMinimums.forEach(v => this.overrides.controls.stockMinimums.push(this.minimumForm(v.siteId ?? '', v.materialId, v.minimumQuantity)));
      this.overrides.controls.referenceValues.clear(); value.referenceValues.forEach(v => this.overrides.controls.referenceValues.push(this.referenceForm(v.key, v.value)));
    } catch (error) { this.error.set(safeFailure(error).message); }
  }
  async saveOverrides(): Promise<void> {
    if (!this.override() || !this.valid(this.overrides)) return;
    const v = this.overrides.getRawValue();
    if (await this.send(`projects/${encodeURIComponent(v.projectId)}/overrides`, {version: this.override()!.version, reservationHours: v.reservationHours,
      inspectionChecklistIds: v.inspectionChecklistIds.split('\n').map(v => v.trim()).filter(Boolean), referenceValues: v.referenceValues,
      stockMinimums: v.stockMinimums.map(m => ({...m, siteId: m.siteId || null}))}, 'PUT')) await this.loadOverrides();
  }
}
