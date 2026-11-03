const landings: Record<string, string> = {
  administrator: '/dashboard', 'project-manager': '/projects', 'site-engineer': '/site-logs',
  'procurement-officer': '/purchase-orders', 'finance-officer': '/invoices',
  'sales-executive': '/leads', 'management-viewer': '/dashboard'
};
export function roleLanding(role: string | undefined): string { return landings[role ?? ''] ?? '/login'; }
const collections: Record<string, string> = {
  dashboard: 'dashboard.read', reports: 'reports.read', projects: 'projects.read',
  'site-logs': 'siteLogs.read', inspections: 'inspections.read', issues: 'issues.read',
  materials: 'materials.read', inventory: 'inventory.read', 'purchase-orders': 'purchaseOrders.read',
  'goods-receipts': 'goodsReceipts.read', 'stock-movements': 'inventory.read', contractors: 'contractors.read',
  'work-orders': 'workOrders.read', budgets: 'budgets.read', expenses: 'expenses.read', invoices: 'invoices.read',
  payments: 'payments.read', properties: 'properties.read', leads: 'leads.read', bookings: 'bookings.read',
  documents: 'documents.parent', users: 'users.manage', settings: 'settings.manage', audit: 'audit.scoped'
};
/** Closed browser-route shapes, not prefix authorization. Go still authorizes each record. */
export function authorizedPath(value: string | null, capabilities: readonly string[]): string | null {
  if (!value || !/^\/[A-Za-z0-9/?=&._~-]+$/.test(value) || value.includes('//') || value.includes('..')) return null;
  const [path] = value.split('?');
  const parts = path.slice(1).split('/');
  const [area, id, child] = parts;
  let capability = collections[area];
  if (!capability || parts.length > 3 || parts.some(part => !part)) return null;
  if (parts.length > 1) {
    if (!['projects', 'site-logs', 'purchase-orders', 'contractors', 'work-orders', 'properties', 'leads', 'bookings'].includes(area)) return null;
    if (id === 'new') {
      capability = ({ projects: 'projects.create', 'site-logs': 'siteLogs.manage', 'purchase-orders': 'purchaseOrders.manage' } as Record<string, string>)[area];
      if (!capability || child) return null;
    }
    if (child) {
      if (area !== 'projects' || child !== 'plan') return null;
      capability = 'plan.read';
    }
  }
  return capabilities.includes(capability) ? value : null;
}
