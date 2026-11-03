import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { SessionService } from '../session/session.service';
@Component({ standalone: true, imports: [RouterLink], template: '<main><h1 tabindex="-1">Access denied</h1><p>Your account does not have access to this page.</p><a [routerLink]="session.destination(null)">Return to your workspace</a></main>' })
export class AccessDeniedPage { readonly session = inject(SessionService); }
