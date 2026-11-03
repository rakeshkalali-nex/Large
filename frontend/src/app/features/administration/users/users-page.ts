import { HttpErrorResponse } from '@angular/common/http';
import { ChangeDetectorRef, Component, ElementRef, OnDestroy, computed, effect, inject, signal, untracked, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { SessionStore } from '../../../core/state/session-store';
import { safeFailure } from '../../../core/http/safe-error';
import { AccountAction, IssuedLink, Project, Role, User, UserInput, UsersApi } from './users-api';

@Component({
  selector: 'be-users-page', standalone: true, imports: [FormsModule],
  templateUrl: './users-page.html', styleUrl: './users-page.scss',
})
export class UsersPage implements OnDestroy {
  private readonly api = inject(UsersApi);
  private readonly changeDetector = inject(ChangeDetectorRef);
  private readonly session = inject(SessionStore);
  readonly authorized = computed(() => this.session.status() === 'authenticated' && this.session.user()?.role === 'administrator');
  readonly modal = viewChild<ElementRef<HTMLDialogElement>>('modal');
  readonly users = signal<User[]>([]);
  readonly roles = signal<Role[]>([]);
  readonly projects = signal<Project[]>([]);
  readonly loading = signal(false);
  readonly pending = signal(false);
  readonly error = signal('');
  readonly notice = signal('');
  readonly dialogError = signal('');
  readonly mode = signal<'editor' | 'action' | 'link' | null>(null);
  readonly link = signal<IssuedLink | null>(null);
  search = '';
  selected: User | null = null;
  draft: UserInput = this.emptyDraft();
  action: AccountAction = 'deactivate';
  reason = '';
  identityVerified = false;
  private generation = 0;
  private alive = true;
  private attempt: { fingerprint: string; key: string } | null = null;
  private returnFocus: HTMLElement | null = null;

  constructor() {
    effect(() => {
      const authorized = this.authorized();
      untracked(() => {
        if (authorized) void this.load();
        else { this.generation++; this.users.set([]); this.roles.set([]); this.projects.set([]); this.dismiss(); }
      });
    });
  }
  ngOnDestroy(): void { this.alive = false; this.generation++; this.link.set(null); }
  private emptyDraft(): UserInput { return { displayName: '', email: '', role: '', projectIds: [] }; }
  async load(): Promise<void> {
    if (!this.authorized()) return;
    const generation = ++this.generation;
    this.loading.set(true); this.error.set('');
    try {
      const [users, roles, projects] = await Promise.all([this.api.list(this.search.trim()), this.api.roles(), this.api.projects()]);
      if (!this.alive || !this.authorized() || generation !== this.generation) return;
      this.users.set(users); this.roles.set(roles); this.projects.set(projects);
    } catch (error) {
      if (this.alive && this.authorized() && generation === this.generation) this.error.set(this.message(error));
    } finally { if (generation === this.generation) this.loading.set(false); }
  }
  roleName(code: string): string { return this.roles().find(r => r.code === code)?.name ?? code; }
  projectNames(user: User): string {
    if (user.role === 'administrator') return 'Company-wide access';
    return user.projectIds.map(id => this.projects().find(p => p.id === id)?.name ?? id).join(', ') || 'No projects assigned';
  }
  edit(user: User | null): void {
    if (!this.authorized() || this.pending()) return;
    this.selected = user;
    this.draft = user ? { displayName: user.displayName, email: user.email, role: user.role, projectIds: [...user.projectIds] } : this.emptyDraft();
    this.open('editor');
  }
  toggleProject(id: string, checked: boolean): void {
    this.draft.projectIds = checked ? [...new Set([...this.draft.projectIds, id])] : this.draft.projectIds.filter(value => value !== id);
  }
  confirmAction(user: User, action: AccountAction): void {
    if (!this.authorized() || this.pending()) return;
    this.selected = user; this.action = action; this.reason = ''; this.identityVerified = false;
    this.open('action');
  }
  private open(mode: 'editor' | 'action'): void {
    this.returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    this.attempt = null; this.dialogError.set(''); this.link.set(null); this.mode.set(mode);
    // Render the dialog contents before native focus placement and trapping.
    this.changeDetector.detectChanges();
    this.modal()?.nativeElement.showModal();
  }
  dismiss(): void {
    this.modal()?.nativeElement.close(); this.mode.set(null); this.link.set(null);
    this.selected = null; this.draft = this.emptyDraft(); this.reason = ''; this.identityVerified = false; this.attempt = null;
    this.returnFocus?.focus(); this.returnFocus = null;
  }
  cancel(event: Event): void { event.preventDefault(); if (!this.pending()) this.dismiss(); }
  private key(payload: unknown): string {
    const fingerprint = JSON.stringify(payload);
    if (this.attempt?.fingerprint !== fingerprint) this.attempt = { fingerprint, key: crypto.randomUUID() };
    return this.attempt!.key;
  }
  async save(): Promise<void> {
    if (!this.authorized() || this.pending()) return;
    const input = { ...this.draft, displayName: this.draft.displayName.trim(), email: this.draft.email.trim(), projectIds: [...this.draft.projectIds] };
    if (!input.displayName || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input.email) || !this.roles().some(r => r.code === input.role)) {
      this.dialogError.set('Enter a display name, valid email and role.'); return;
    }
    const selected = this.selected;
    this.pending.set(true); this.dialogError.set('');
    try {
      const saved = await this.api.save(input, selected, this.key({ id: selected?.id, version: selected?.version, input }));
      if (!this.alive || !this.authorized()) return;
      this.dismiss(); this.notice.set(selected ? 'Account and access saved. Changed access invalidates affected sessions.' : 'Account created. Verify identity, then explicitly issue a setup link.');
      await this.load();
      // Keep a newly created account reachable even when the current search excludes it.
      if (!selected && this.authorized() && !this.users().some(u => u.id === saved.id)) this.users.update(users => [saved, ...users]);
    } catch (error) { if (this.alive && this.authorized()) this.dialogError.set(this.message(error)); }
    finally { this.pending.set(false); }
  }
  async performAction(): Promise<void> {
    if (!this.authorized() || this.pending() || !this.selected) return;
    const reason = this.reason.trim();
    const issuance = this.action === 'setup' || this.action === 'recovery';
    if (!reason || reason.length > 1000 || (issuance && !this.identityVerified)) {
      this.dialogError.set('Enter a reason (up to 1000 characters) and confirm identity verification when issuing a link.'); return;
    }
    const user = this.selected, action = this.action;
    const key = this.key({ id: user.id, version: user.version, action, reason });
    this.pending.set(true); this.dialogError.set('');
    try {
      if (action === 'setup' || action === 'recovery') {
        const link = await this.api.issue(user, action, reason, key);
        if (!this.alive || !this.authorized()) return;
        this.link.set(link); this.mode.set('link'); this.attempt = null;
        this.changeDetector.detectChanges();
        this.modal()?.nativeElement.querySelector<HTMLTextAreaElement>('#issued-link')?.focus();
      } else {
        await this.api.transition(user, action, reason, key);
        if (!this.alive || !this.authorized()) return;
        this.dismiss(); this.notice.set(action === 'reactivate' ? 'Account reactivated as setup-required. Verify identity and issue a fresh setup link.' : 'Account deactivated. Business attribution is retained.');
        await this.load();
      }
    } catch (error) { if (this.alive && this.authorized()) this.dialogError.set(this.message(error)); }
    finally { this.pending.set(false); }
  }
  deliveryLink(): string { return this.link() ? new URL(this.link()!.resetLink, window.location.origin).href : ''; }
  private message(error: unknown): string {
    if (error instanceof HttpErrorResponse) {
      if (error.status === 429) return 'Too many attempts. Wait before deliberately trying again.';
      if (error.status === 422) return 'Check the account identity, unique email, role, project assignments and reason. Your input has been retained.';
      if (error.status === 409) return 'The account changed or this action is not allowed (including removing the final active Administrator). Close and refresh the list before editing again. Your input has been retained.';
    }
    return safeFailure(error).message;
  }
}
