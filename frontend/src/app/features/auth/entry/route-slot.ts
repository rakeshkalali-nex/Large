import { Routes } from '@angular/router';
import { loginGuard, rootEntryGuard } from '../../../core/guards/entry.guards';
import { AccountPage } from './account-page';

export const ENTRY_ROUTES: Routes = [
  { path: '', pathMatch: 'full', canActivate: [rootEntryGuard], component: AccountPage },
  { path: 'login', canActivate: [loginGuard], component: AccountPage },
];
