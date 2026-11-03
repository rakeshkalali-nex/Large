import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { firstValueFrom, map } from 'rxjs';

export interface User {
  id: string; displayName: string; email: string; role: string;
  status: 'setup-required' | 'active' | 'deactivated'; version: number; projectIds: string[];
}
export interface UserInput { displayName: string; email: string; role: string; projectIds: string[]; }
export interface Role { code: string; name: string; }
export interface Project { id: string; name: string; }
export interface IssuedLink { userId: string; resetLink: string; expiresAt: string; }
export type AccountAction = 'deactivate' | 'reactivate' | 'setup' | 'recovery';
interface Envelope<T> { data: T; }

@Injectable({ providedIn: 'root' })
export class UsersApi {
  private readonly http = inject(HttpClient);
  list(search: string): Promise<User[]> {
    return firstValueFrom(this.http.get<Envelope<{ items: User[] }>>('/api/v1/users', { params: { search } }).pipe(map(r => r.data.items)));
  }
  roles(): Promise<Role[]> {
    return firstValueFrom(this.http.get<Envelope<{ items: Role[] }>>('/api/v1/roles').pipe(map(r => r.data.items)));
  }
  projects(): Promise<Project[]> {
    return firstValueFrom(this.http.get<Envelope<{ items: Project[] }>>('/api/v1/users/projects').pipe(map(r => r.data.items)));
  }
  save(input: UserInput, user: User | null, key: string): Promise<User> {
    const headers = { 'Idempotency-Key': key };
    const request = user
      ? this.http.put<Envelope<User>>(`/api/v1/users/${encodeURIComponent(user.id)}`, { ...input, version: user.version }, { headers })
      : this.http.post<Envelope<User>>('/api/v1/users', input, { headers });
    return firstValueFrom(request.pipe(map(r => r.data)));
  }
  transition(user: User, action: 'deactivate' | 'reactivate', reason: string, key: string): Promise<User> {
    return firstValueFrom(this.http.post<Envelope<User>>(`/api/v1/users/${encodeURIComponent(user.id)}/${action}`,
      { version: user.version, reason }, { headers: { 'Idempotency-Key': key } }).pipe(map(r => r.data)));
  }
  issue(user: User, purpose: 'setup' | 'recovery', reason: string, key: string): Promise<IssuedLink> {
    return firstValueFrom(this.http.post<Envelope<IssuedLink>>(`/api/v1/users/${encodeURIComponent(user.id)}/password-reset`,
      { version: user.version, purpose, reason, identityVerified: true }, { headers: { 'Idempotency-Key': key } }).pipe(map(r => r.data)));
  }
}
