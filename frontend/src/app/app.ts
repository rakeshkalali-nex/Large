import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AppShell } from './core/shell/app-shell';
@Component({ selector: 'app-root', standalone: true, imports: [RouterOutlet, AppShell], template: '<be-app-shell><router-outlet /></be-app-shell>' })
export class App {}
