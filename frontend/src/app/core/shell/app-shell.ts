import { Component, computed, DestroyRef, ElementRef, inject, InjectionToken, signal, TemplateRef, viewChild } from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';
import { Dialog, DialogModule, DialogRef } from '@angular/cdk/dialog';
import { BreakpointObserver } from '@angular/cdk/layout';
import { Overlay } from '@angular/cdk/overlay';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterLinkActive } from '@angular/router';
import { filter } from 'rxjs';
import { SessionStore } from '../state/session-store';
import { ProjectContext } from '../state/project-context';
import { NAVIGATION } from './navigation';
import { IconComponent } from '../../shared/icon';

/** Authentication supplies a real server-revoking logout action, never a local login bypass. */
export const SHELL_LOGOUT = new InjectionToken<(() => Promise<void>) | null>('SHELL_LOGOUT', { providedIn: 'root', factory: () => null });

@Component({
  selector: 'be-app-shell', standalone: true,
  imports: [NgTemplateOutlet, RouterLink, RouterLinkActive, DialogModule, IconComponent],
  templateUrl: './app-shell.html', styleUrl: './app-shell.scss'
})
export class AppShell {
  readonly session = inject(SessionStore);
  readonly projects = inject(ProjectContext);
  private readonly router = inject(Router);
  private readonly dialog = inject(Dialog);
  private readonly host: ElementRef<HTMLElement> = inject(ElementRef);
  private readonly overlay = inject(Overlay);
  private readonly logoutAction = inject(SHELL_LOGOUT);
  readonly canLogout = this.logoutAction !== null;
  readonly mobile = signal(false);
  readonly collapsed = signal(false);
  readonly userOpen = signal(false);
  readonly loggingOut = signal(false);
  readonly logoutError = signal('');
  readonly title = signal('Workspace');
  readonly parent = signal<{ label: string; path: string } | null>(null);
  readonly routeProjectId = signal<string | null>(null);
  readonly drawerTemplate = viewChild.required<TemplateRef<unknown>>('drawer');
  readonly groups = computed(() => NAVIGATION.map(group => ({ ...group, items: group.items.filter(item => this.session.has(item.capability)) })).filter(group => group.items.length));
  private drawerRef?: DialogRef;
  private focusTimer?: ReturnType<typeof setTimeout>;
  constructor() {
    inject(BreakpointObserver).observe(['(max-width: 768px)', '(max-width: 1024px)']).pipe(takeUntilDestroyed()).subscribe(state => {
      this.mobile.set(state.breakpoints['(max-width: 768px)']);
      this.collapsed.set(state.breakpoints['(max-width: 1024px)']);
      if (!this.mobile()) this.drawerRef?.close();
    });
    this.router.events.pipe(filter(event => event instanceof NavigationEnd), takeUntilDestroyed()).subscribe(() => {
      this.drawerRef?.close(); this.userOpen.set(false);
      let route = this.router.routerState.snapshot.root;
      let projectId: string | null = null;
      while (route.firstChild) { route = route.firstChild; projectId = route.paramMap.get('projectId') ?? projectId; }
      const path = this.router.url.split(/[?#]/)[0];
      const item = NAVIGATION.flatMap(group => group.items).find(entry => path === entry.path || path.startsWith(entry.path + '/'));
      const detail = !!item && path !== item.path;
      if (item?.path === '/projects' && detail) projectId = route.paramMap.get('id') ?? projectId;
      this.routeProjectId.set(projectId);
      this.title.set(route.title ?? (detail ? `${item?.label} detail` : item?.label) ?? 'Workspace');
      this.parent.set(detail && item ? { label: item.label, path: item.path } : null);
      clearTimeout(this.focusTimer);
      this.focusTimer = setTimeout(() => {
        const heading = this.host.nativeElement.querySelector<HTMLElement>('[data-page-content] h1, main h1');
        if (heading) { heading.tabIndex = -1; heading.focus(); }
      });
    });
    inject(DestroyRef).onDestroy(() => { clearTimeout(this.focusTimer); this.drawerRef?.close(); });
  }
  openNavigation(): void {
    if (this.drawerRef) return;
    const ref = this.dialog.open(this.drawerTemplate(), { ariaLabel: 'Application navigation', autoFocus: 'first-tabbable', restoreFocus: true, maxWidth: '100vw', width: 'min(320px, 100vw)', height: '100dvh', positionStrategy: this.overlay.position().global().left('0').top('0') });
    this.drawerRef = ref;
    ref.closed.subscribe(() => { this.drawerRef = undefined; });
  }
  closeNavigation(): void { this.drawerRef?.close(); }
  async selectProject(event: Event): Promise<void> {
    const select = event.target as HTMLSelectElement;
    await this.projects.change(select.value || null);
    select.value = this.projects.context().projectId ?? '';
  }
  async logout(): Promise<void> {
    if (!this.logoutAction || this.loggingOut()) return;
    this.loggingOut.set(true); this.logoutError.set('');
    try { await this.logoutAction(); }
    catch { this.logoutError.set('Sign out could not be confirmed. Please retry.'); }
    finally { this.loggingOut.set(false); }
  }
}
