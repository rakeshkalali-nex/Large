import { describe, it, expect } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { HealthClient } from '../core/contracts/health-client';

describe('setup typed health client', () => {
  it('reads the same-origin versioned readiness envelope', () => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    const http = TestBed.inject(HttpTestingController);
    TestBed.inject(HealthClient).readiness().subscribe(result => expect(result.data.status).toBe('ready'));
    const request = http.expectOne('/api/v1/health/ready');
    expect(request.request.method).toBe('GET');
    request.flush({ data: { status: 'ready' }, requestId: '00000000-0000-4000-8000-000000000001' });
    http.verify();
  });
});
