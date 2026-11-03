import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';

export interface AuditEvent {
  id: string; actorId: string | null; actorName: string | null; occurredAt: string; action: string;
  projectId: string | null; projectName: string | null; recordType: string; recordId: string;
  recordLink: string | null; requestId: string; reason: string | null;
  changes: { field: string; before: string | null; after: string | null }[];
  retention: { class: string; eligibleAt: string | null; holdIds: string[] };
}
export interface AuditQuery {
  q?: string; actorId?: string; action?: string; projectId?: string; recordType?: string; recordId?: string;
  from?: string; to?: string; limit?: number; cursor?: string; sort?: 'occurredAt'; direction?: 'asc' | 'desc';
}
export interface AuditEnvelope { data: AuditEvent; requestId: string; }
export interface AuditPageEnvelope { data: AuditEvent[]; page: { nextCursor: string | null; limit: number }; requestId: string; }

@Injectable({ providedIn: 'root' })
export class AuditApi {
  private readonly http = inject(HttpClient);
  list(query: AuditQuery = {}) {
    let params = new HttpParams();
    for (const [key, value] of Object.entries({ limit: 25, sort: 'occurredAt', direction: 'desc', ...query })) {
      if (value !== undefined && value !== '') params = params.set(key, String(value));
    }
    return this.http.get<AuditPageEnvelope>('/api/v1/audit-events', { params });
  }
  detail(id: string) { return this.http.get<AuditEnvelope>(`/api/v1/audit-events/${encodeURIComponent(id)}`); }
}
