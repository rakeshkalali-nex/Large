import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

export interface ReferenceValue { key: string; value: string; label: string; }
export interface Checklist { id: string; name: string; items: string[]; }
export interface Settings {
  version: number; companyName: string; branding: { wordmark: string; logoDocumentId: string | null };
  currencyCode: string | null; currencyMinorDigits: number | null; currencyLocked: boolean;
  timezone: string; timezoneConfirmed: boolean; recoveryInstructions: string; recoveryContactLabel: string;
  reservationHours: number; referenceValues: ReferenceValue[]; propertyTypes: string[]; inspectionChecklists: Checklist[];
  operatingPolicy: { countries: string[]; noticeVersion: string | null; retentionScheduleVersion: string | null; confirmedAt: string | null };
}
export type SettingsInput = Omit<Settings, 'currencyLocked' | 'operatingPolicy'> & {
  operatingPolicy: Omit<Settings['operatingPolicy'], 'confirmedAt'>; confirmOperationalPolicy: boolean;
};
export interface Envelope<T> { data: T; requestId: string; }
export interface HistoryEntry { id: string; action: string; actorId: string; occurredAt: string; reason?: string; settings?: Settings; }
export interface OperatorStatus {
  version: string; startedAt: string; ready: boolean;
  database: { status: string; diagnosticId?: string | null };
  attachments: { status: string; diagnosticId?: string | null };
  backup: { status: string; lastSuccessAt: string | null; nextDueAt: string | null; diagnosticId: string | null };
}
export interface PrivacyRequest { id: string; version: number; subjectType: string; subjectId: string; kind: string; reason: string; status: string; accessResult?: {name: string; email: string | null; phone: string | null} | null; review?: {reviewedBy: string; reviewedAt: string; reason: string} | null; }
export interface Hold { id: string; version: number; recordType: string; recordId: string; reason: string; status: string; }
export interface RetentionReview { id: string; version: number; status: string; candidates: { recordType: string; recordId: string; eligibleAt: string; holdIds: string[]; action?: string }[]; }
export interface Overrides { projectId?: string; version: number; reservationHours: number | null; stockMinimums: { siteId: string | null; materialId: string; minimumQuantity: string }[]; inspectionChecklistIds: string[]; referenceValues: {key: string; value: string}[]; }

@Injectable({ providedIn: 'root' })
export class SettingsApi {
  private readonly http = inject(HttpClient);
  async get<T>(path: string): Promise<T> { return (await firstValueFrom(this.http.get<Envelope<T>>(`/api/v1/${path}`))).data; }
  async mutate<T>(method: 'POST' | 'PUT', path: string, body: unknown, key: string): Promise<T> {
    const csrf = await this.get<{csrfToken: string}>('auth/csrf');
    return (await firstValueFrom(this.http.request<Envelope<T>>(method, `/api/v1/${path}`, { body, headers: { 'Idempotency-Key': key, 'X-CSRF-Token': csrf.csrfToken } }))).data;
  }
}
