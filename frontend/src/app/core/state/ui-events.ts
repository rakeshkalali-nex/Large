import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';
import type { components } from '../contracts/generated/api';

type Schema = components['schemas'];
export type SessionStatus = 'restoring' | 'anonymous' | 'setupRequired' | 'authenticated';
export interface UiEventMap {
  'session.changed': { status: SessionStatus; userId: string | null; accessRevision: number | null };
  'project.changed': { projectId: string | null; revision: number };
  'record.committed': Schema['RecordEffect'] & { requestId: string };
  'stock.committed': Schema['StockEffect'] & { requestId: string };
  'unit.committed': Schema['UnitEffect'] & { requestId: string };
  'finance.committed': Schema['FinanceEffect'] & { requestId: string };
}
export type UiEvent = { [K in keyof UiEventMap]: { name: K; payload: UiEventMap[K] } }[keyof UiEventMap];

/** Transient invalidation hints only; never a source of business truth. */
@Injectable({ providedIn: 'root' })
export class UiEvents {
  private readonly source = new Subject<UiEvent>();
  readonly events = this.source.asObservable();
  emit(event: UiEvent): void { this.source.next(event); }
  committed(effects: Schema['Effects'], requestId: string): void {
    for (const record of effects.records) this.emit({ name: 'record.committed', payload: { ...record, requestId } });
    if (effects.stock) this.emit({ name: 'stock.committed', payload: { ...effects.stock, requestId } });
    if (effects.unit) this.emit({ name: 'unit.committed', payload: { ...effects.unit, requestId } });
    if (effects.finance) this.emit({ name: 'finance.committed', payload: { ...effects.finance, requestId } });
  }
}
