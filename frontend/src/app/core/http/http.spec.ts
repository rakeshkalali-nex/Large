import { TestBed } from '@angular/core/testing';
import { HttpClient, HttpErrorResponse, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { describe, expect, it, vi } from 'vitest';
import { MutationState } from './mutation-state';
import { safeErrorInterceptor, safeFailure } from './safe-error';
import { SessionStore } from '../state/session-store';

describe('core request failures', () => {
  it.each([0, 401, 403, 404, 409, 422, 500, 503])('maps %s without exposing raw server content', status => {
    const result = safeFailure(new HttpErrorResponse({ status, error: { message: 'private stack /secret/file' } }));
    expect(result.message).not.toContain('private'); expect(result.message).not.toContain('/secret');
  });
  it('clears expired identity without replaying a request', () => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(withInterceptors([safeErrorInterceptor])), provideHttpClientTesting()] });
    const http = TestBed.inject(HttpClient); const controller = TestBed.inject(HttpTestingController);
    http.post('/api/v1/projects', {}).subscribe({ error: () => undefined });
    controller.expectOne('/api/v1/projects').flush({}, { status: 401, statusText: 'Unauthorized' });
    expect(TestBed.inject(SessionStore).status()).toBe('anonymous'); controller.verify();
  });
  it('blocks duplicates, preserves safe input, and retries only deliberately with the same key', async () => {
    const state = new MutationState<{ notes: string }, string>();
    const draft = { notes: 'Keep this note' }; let reject!: (error: unknown) => void;
    const send = vi.fn(() => new Promise<string>((_, fail) => { reject = fail; }));
    const first = state.submit(draft, send);
    expect(state.pending()).toBe(true); await state.submit(draft, send); expect(send).toHaveBeenCalledTimes(1);
    reject(new HttpErrorResponse({ status: 0 })); await first;
    expect(state.failure()?.uncertain).toBe(true); expect(draft.notes).toBe('Keep this note'); expect(send).toHaveBeenCalledTimes(1);
    const retry = vi.fn(async () => 'saved'); await state.submit(draft, retry);
    expect(retry.mock.calls[0]).toEqual(send.mock.calls[0]); expect(state.result()).toBe('saved');
  });
  it('uses a new key when a failed payload is edited', async () => {
    const state = new MutationState<{ notes: string }, string>();
    const keys: string[] = [];
    const send = async (_: { notes: string }, key: string) => { keys.push(key); throw new HttpErrorResponse({ status: 422 }); };
    await state.submit({ notes: 'old' }, send); await state.submit({ notes: 'new' }, send);
    expect(keys[0]).not.toBe(keys[1]);
  });
});
