import { Component, input } from '@angular/core';
import {
  LucideAngularModule, LUCIDE_ICONS, LucideIconProvider,
  Menu, X, House, Building2, Users, Wallet, Settings, ChartNoAxesCombined,
  ClipboardList, Package, FileText, LogOut, ChevronDown,
  LoaderCircle, Inbox, CircleCheck, CircleAlert, TriangleAlert, Info,
} from 'lucide-angular';

const shellIcons = {
  menu: Menu, close: X, home: House, projects: Building2, users: Users,
  finance: Wallet, settings: Settings, reports: ChartNoAxesCombined,
  construction: ClipboardList, procurement: Package, documents: FileText,
  logout: LogOut, chevronDown: ChevronDown,
  loading: LoaderCircle, empty: Inbox, success: CircleCheck,
  error: CircleAlert, warning: TriangleAlert, info: Info,
};
export type SharedIconName = keyof typeof shellIcons;

/** Decorative by default: the containing link/button must retain its accessible name. */
@Component({
  selector: 'be-icon', standalone: true, imports: [LucideAngularModule],
  providers: [{ provide: LUCIDE_ICONS, multi: true, useValue: new LucideIconProvider({}) }],
  template: `<lucide-icon [img]="icons[name()]" [size]="size()" aria-hidden="true" />`,
  host: { 'aria-hidden': 'true' },
  styles: [':host { display:inline-flex; flex:none; vertical-align:middle; }'],
})
export class IconComponent {
  readonly name = input.required<SharedIconName>();
  readonly size = input(20);
  readonly icons = shellIcons;
}
