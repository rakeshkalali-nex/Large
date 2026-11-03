import { Component, inject } from '@angular/core';
import { Router, RouterOutlet } from '@angular/router';
import { AppShell } from './core/shell/app-shell';
import { SessionStore } from './core/state/session-store';
@Component({ selector: 'app-root', standalone: true, imports: [RouterOutlet, AppShell], template: '@if (session.status() === "authenticated" && !router.url.startsWith("/login")) { <be-app-shell><router-outlet /></be-app-shell> } @else { <router-outlet /> }' })
export class App { readonly session = inject(SessionStore); readonly router = inject(Router); }
