import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import type { components } from './generated/api';

/** Typed foundation client; feature clients consume the same generated contract. */
@Injectable({ providedIn: 'root' })
export class HealthClient {
  private readonly http = inject(HttpClient);
  readiness() {
    return this.http.get<components['schemas']['HealthReadyResponse']>('/api/v1/health/ready');
  }
}
