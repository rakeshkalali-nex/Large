// Authoring source for OpenAPI 3.1. JSON output is also valid YAML 1.2.
// Explicit compact field notation: name:Schema, ? means nullable (not omitted).
export const schemas = {};
const ref = name => ({ $ref: `#/components/schemas/${name}` });
const array = items => ({ type: 'array', items });
const nullable = schema => ({ anyOf: [schema, { type: 'null' }] });
const en = values => ({ type: 'string', enum: values.split('|') });
const object = properties => ({ type: 'object', additionalProperties: false, properties, required: Object.keys(properties) });
function type(value) {
  if (value.endsWith('[]')) return array(type(value.slice(0, -2)));
  if (value.includes('|')) return en(value);
  if (/^[a-z]/.test(value)) return { type: 'string', const: value };
  return ref(value);
}
function fields(text) {
  return Object.fromEntries(text.trim().split(/\s+/).filter(Boolean).map(field => {
    const [raw, kind = 'Text'] = field.split(':');
    return [raw.replace('?', ''), raw.endsWith('?') ? nullable(type(kind)) : type(kind)];
  }));
}
function define(name, text, extra = {}) { schemas[name] = { ...object(fields(text)), ...extra }; return ref(name); }
function merge(name, ...names) { schemas[name] = object(Object.assign({}, ...names.map(n => schemas[n].properties))); return ref(name); }
Object.assign(schemas, {
  Text: { type: 'string' }, UUID: { type: 'string', format: 'uuid' },
  Date: { type: 'string', format: 'date' }, Timestamp: { type: 'string', format: 'date-time', pattern: 'Z$', description: 'RFC3339 UTC timestamp.' },
  Boolean: { type: 'boolean' }, Integer: { type: 'integer', minimum: 0, maximum: 2147483647 },
  Version: { type: 'integer', minimum: 1, maximum: 9007199254740991 },
  Money: { type: 'string', pattern: '^(0|[1-9][0-9]*)$', description: 'Nonnegative exact minor units, bounded to signed int64 by the server. Never a JSON number.', examples: ['1543'] },
  SignedMoney: { type: 'string', pattern: '^-?(0|[1-9][0-9]*)$', description: 'Signed exact minor units; signed int64 bounds.', examples: ['-1543'] },
  ExactInteger: { type: 'string', pattern: '^(0|[1-9][0-9]*)$', description: 'Exact integer encoded as decimal string.' },
  Quantity: { type: 'string', pattern: '^(0|[1-9][0-9]*)(\\.[0-9]{1,3})?$', examples: ['1.250'] },
  SignedQuantity: { type: 'string', pattern: '^-?(0|[1-9][0-9]*)(\\.[0-9]{1,3})?$' },
  UnitPrice: { type: 'string', pattern: '^(0|[1-9][0-9]*)(\\.[0-9]{1,4})?$', examples: ['12.3456'] },
  Percent: { type: 'string', pattern: '^(100(\\.0{1,2})?|[0-9]{1,2}(\\.[0-9]{1,2})?)$', description: 'Exact decimal percent; stored as basis points.' },
  BasisPoints: { type: 'integer', minimum: 0, maximum: 10000 },
  Email: { type: 'string', format: 'email', maxLength: 254 },
  Password: { type: 'string', minLength: 12, maxLength: 128, writeOnly: true },
  Role: en('administrator|project-manager|site-engineer|procurement-officer|finance-officer|sales-executive|management-viewer'),
  Availability: en('available|reserved|booked|sold|unavailable'),
  Priority: en('low|normal|high|critical'), Counterparty: en('customer|supplier|contractor'),
  PaymentMethod: en('cash|cheque|bank-transfer|other'),
  PlanStatus: en('planned|in-progress|completed|cancelled'), LeadStage: en('new|contacted|qualified|visit|booked|lost'),
  ReportName: en('progress|procurement|budget|inventory|sales|collections'),
  Scalar: { type: ['string', 'number', 'boolean', 'null'], description: 'Safe, allowlisted audit/report scalar; never credentials or internal diagnostics.' },
});
define('Metadata', 'id:UUID createdAt:Timestamp updatedAt:Timestamp createdBy:UUID version:Version');
define('Actions', 'allowedActions:Text[]');
define('Contact', 'name email?:Email phone?');
define('Review', 'id:UUID actorId:UUID occurredAt:Timestamp decision reason?');
define('Change', 'field before:Scalar after:Scalar');
define('SourceVersion', 'resource id:UUID version:Version');
define('DuplicateWarning', 'recordId:UUID code message');
define('Allocation', 'id:UUID invoiceId:UUID amountMinor:Money status:active|reversed');
define('CollectionSummary', 'invoicedMinor:Money collectedMinor:Money outstandingMinor:Money');
define('RefundCalculation', 'collectionsMinor:Money deductionMinor:Money paidRefundsMinor:Money remainingRefundableMinor:Money termsId:UUID termsVersion:Version');
define('RefundSummary', 'paidMinor:Money remainingRefundableMinor?:Money');
define('Cancellation', 'id:UUID status:pending-review|approved|rejected requestedAt:Timestamp reason termsId?:UUID calculation?:RefundCalculation reviews:Review[]');
define('SaleMilestone', 'agreementDocumentId:UUID milestoneDate:Date requiredPaymentMinor:Money notes verification?:PaymentVerification confirmedBy?:UUID confirmedAt?:Timestamp');
define('PaymentVerification', 'verifierId:UUID verifiedAt:Timestamp purpose:sale|handover allocationIds:UUID[] documentIds:UUID[] sourceVersions:SourceVersion[] verifiedAmountMinor:Money notes');
define('HandoverItemInput', 'id?:UUID label completed:Boolean documentIds:UUID[]');
define('Handover', 'items:HandoverItemInput[] clearance?:PaymentVerification handoverDate?:Date notes? completedBy?:UUID');
define('PriceHistory', 'askingPriceMinor:Money reason actorId:UUID occurredAt:Timestamp');
define('Image', 'documentId:UUID altText sortOrder:Integer');
define('SiteInput', 'code name address');
define('Worker', 'category count:Integer');
define('InspectionItem', 'itemId:UUID label result:pass|fail|not-applicable finding issueId?:UUID');
define('PurchaseLineInput', 'lineId?:UUID materialId:UUID unit quantity:Quantity unitPrice:UnitPrice categoryId:UUID');
define('PurchaseLine', 'lineId:UUID materialId:UUID unit quantity:Quantity unitPrice:UnitPrice categoryId:UUID receivedQuantity:Quantity remainingQuantity:Quantity');
define('ReceiptLineInput', 'lineId?:UUID purchaseOrderLineId:UUID quantity:Quantity unit');
define('WorkMilestoneInput', 'lineId?:UUID name dueDate:Date amountMinor:Money categoryId:UUID');
define('BudgetLineInput', 'categoryId:UUID amountMinor:Money');
define('BudgetLine', 'categoryId:UUID amountMinor:Money actualMinor:Money commitmentMinor:Money varianceMinor:SignedMoney');
define('InvoiceLineInput', 'lineId?:UUID description quantity:Quantity unitPrice:UnitPrice categoryId?:UUID sourceCostId?:UUID purchaseOrderLineId?:UUID workOrderMilestoneId?:UUID receiptLineIds:UUID[]');
define('InstallmentInput', 'id?:UUID label dueDate:Date amountMinor:Money');
define('Installment', 'id:UUID label dueDate:Date amountMinor:Money invoiceId?:UUID outstandingMinor:Money');
define('ReconciliationLine', 'id:UUID sourceType sourceId:UUID sourceCostId?:UUID amountMinor:Money discrepancy?:Text resolution?:Text documentIds:UUID[] reviews:Review[]');
define('CostTotals', 'budgetMinor:Money actualMinor:Money commitmentMinor:Money varianceMinor:SignedMoney');
define('RetentionCandidate', 'recordType recordId:UUID eligibleAt:Timestamp holdIds:UUID[]');
define('PrivacyReview', 'decision:approve|reject actorId:UUID occurredAt:Timestamp reason');
define('Page', 'nextCursor?:Text limit:Integer');
schemas.Page.properties.limit = { type: 'integer', minimum: 1, maximum: 100, default: 25 };
define('RecordEffect', 'resource id:UUID projectId?:UUID version:Version');
define('StockEffect', 'transactionId:UUID projectId:UUID siteIds:UUID[] materialIds:UUID[]');
define('UnitEffect', 'propertyId:UUID projectId:UUID availability:Availability bookingId?:UUID');
define('FinanceEffect', 'projectId:UUID counterpartyId:UUID bookingId?:UUID invoiceIds:UUID[] paymentIds:UUID[]');
define('Effects', 'records:RecordEffect[] stock?:StockEffect unit?:UnitEffect finance?:FinanceEffect');
define('FieldError', 'field code message');
define('ErrorDetails', 'currentVersion:Version availableQuantity:Quantity remainingQuantity:Quantity availability:Availability');
schemas.ErrorDetails.required = [];
schemas.ErrorCode = en('MALFORMED_REQUEST|AUTH_REQUIRED|SESSION_EXPIRED|INVALID_CREDENTIALS|FORBIDDEN|CSRF_INVALID|NOT_FOUND|BODY_TOO_LARGE|VALIDATION_FAILED|INVALID_RESET_TOKEN|RATE_LIMITED|INTERNAL_ERROR|STORAGE_BUSY|SERVICE_UNAVAILABLE|STALE_VERSION|INVALID_TRANSITION|IDEMPOTENCY_MISMATCH|UNIT_UNAVAILABLE|INSUFFICIENT_STOCK|EXCESS_RECEIPT|ALLOCATION_CONFLICT|INDEPENDENT_REVIEW_REQUIRED|EVIDENCE_REQUIRED|UNSUPPORTED_MEDIA_TYPE|EXPORT_TOO_LARGE');
define('ErrorBody', 'code:ErrorCode message fieldErrors:FieldError[] details:ErrorDetails');
schemas.ErrorBody.required = ['code', 'message'];
define('Error', 'error:ErrorBody requestId:UUID');
define('VersionInput', 'version:Version'); define('ReasonInput', 'version:Version reason');
schemas.ReasonInput.properties.reason = { type: 'string', minLength: 1, maxLength: 4000 };

// Fields in these rows are exhaustive. Server additions never enter create/update DTOs.
const resources = [
 ['Project','/projects','LCRU','code name address projectType startDate:Date endDate:Date managerId:UUID initialSites?:SiteInput[]','status:draft|active|on-hold|completed|archived progressPercent?:Percent progressState:configured|unconfigured sites:Site[]','managerId status','name code updatedAt'],
 ['Site','/projects/{projectId}/sites','LCRU','code name address','projectId:UUID status:active|archived','','code name'],
 ['Milestone','/projects/{projectId}/milestones','LR','','projectId:UUID name responsibleId:UUID startDate:Date endDate:Date status:PlanStatus completionPercent:Percent weightPercent:Percent','','startDate'],
 ['Task','/projects/{projectId}/tasks','LCRU','milestoneId?:UUID name responsibleId:UUID startDate:Date endDate:Date','projectId:UUID status:PlanStatus completionPercent:Percent','responsibleId dueBefore','endDate name'],
 ['SiteLog','/site-logs','LCRU','projectId:UUID siteId:UUID date:Date workPerformed workers:Worker[] weather blockers notes documentIds:UUID[]','status:draft|submitted|approved|returned reviewHistory:Review[]','siteId date reviewQueue','date updatedAt'],
 ['Inspection','/inspections','LCRU','projectId:UUID siteId:UUID checklistId:UUID milestoneId?:UUID propertyId?:UUID items:InspectionItem[] documentIds:UUID[]','status:draft|submitted|completed history:Review[]','siteId result','updatedAt'],
 ['Issue','/issues','LCRU','projectId:UUID inspectionItemId?:UUID milestoneId?:UUID propertyId?:UUID title description ownerId:UUID priority:Priority dueDate:Date documentIds:UUID[]','status:open|in-progress|resolved|reopened resolutionNotes? history:Review[]','ownerId priority overdue','priority dueDate'],
 ['Material','/materials','LCRU','code name unit minimumQuantity:Quantity','status:active|archived','unit','code name'],
 ['Supplier','/suppliers','LCRU','name address contacts:Contact[] reference?','status:active|archived','','name'],
 ['PurchaseOrder','/purchase-orders','LCRU','projectId:UUID siteId:UUID supplierId:UUID deliveryDate:Date lines:PurchaseLineInput[] notes','number status:draft|submitted|approved|part-received|received|cancelled totalMinor:Money approvalHistory:Review[] revision:Version lines:PurchaseLine[]','supplierId deliveryStatus reviewQueue','createdAt deliveryDate'],
 ['GoodsReceipt','/goods-receipts','LCRU','projectId:UUID purchaseOrderId:UUID siteId:UUID receivedDate:Date reference lines:ReceiptLineInput[] documentIds:UUID[]','status:draft|posted|reversed stockTransactionId?:UUID reversalId?:UUID','purchaseOrderId','receivedDate'],
 ['StockMovement','/stock-movements','LR','','projectId:UUID transactionId:UUID type:receipt|issue|return|transfer|adjustment|reversal materialId:UUID siteId:UUID unit quantity:SignedQuantity sourceType sourceId:UUID actorId:UUID occurredAt:Timestamp reason?','siteId materialId type','occurredAt'],
 ['Contractor','/contractors','LCRU','name contacts:Contact[] tradeCategories:Text[] projectIds:UUID[]','status:active|archived','tradeCategory','name'],
 ['WorkOrder','/work-orders','LCRU','projectId:UUID contractorId:UUID scope startDate:Date endDate:Date managerId:UUID milestones:WorkMilestoneInput[] documentIds:UUID[]','number status:draft|submitted|approved|in-progress|completed|cancelled totalMinor:Money completionPercent:Percent approvalHistory:Review[] revision:Version','contractorId managerId reviewQueue','endDate createdAt'],
 ['Budget','/budgets','LCR','projectId:UUID periodStart:Date periodEnd:Date lines:BudgetLineInput[]','status:draft|approved|revised revision:Version lines:BudgetLine[]','period','updatedAt'],
 ['Expense','/expenses','LCRU','projectId:UUID categoryId:UUID amountMinor:Money reference date:Date description workOrderId?:UUID sourceCostId?:UUID documentIds:UUID[]','status:submitted|approved|rejected reviewHistory:Review[] sourceCostId:UUID','categoryId reviewQueue','date amountMinor'],
 ['Invoice','/invoices','LCRU','type:receivable|payable projectId:UUID number counterpartyType:Counterparty counterpartyId:UUID bookingId?:UUID installmentId?:UUID supplierReference? dueDate:Date lines:InvoiceLineInput[] taxMinor:Money documentIds:UUID[]','status:draft|posted|reversed subtotalMinor:Money totalMinor:Money outstandingMinor:Money duplicateWarnings:DuplicateWarning[] allocations:Allocation[] reversalId?:UUID','type counterpartyId overdue','dueDate number'],
 ['Payment','/payments','LCRU','direction:incoming|outgoing projectId:UUID counterpartyType:Counterparty counterpartyId:UUID bookingId?:UUID amountMinor:Money method:PaymentMethod reference date:Date notes documentIds:UUID[]','status:draft|posted|reversed unallocatedMinor:Money allocations:Allocation[] refundOfPaymentIds:UUID[] reversalId?:UUID','direction counterpartyId allocationStatus','date'],
 ['Property','/properties','LCRU','projectId:UUID siteId:UUID unitCode block floor? propertyType area:Quantity areaUnit askingPriceMinor:Money description constructionStatus','availability:Availability priceHistory:PriceHistory[] images:Image[] bookingId?:UUID','propertyType minPriceMinor maxPriceMinor availability','unitCode askingPriceMinor'],
 ['Lead','/leads','LCRU','projectId:UUID contact:Contact source salesExecutiveId:UUID minBudgetMinor?:Money maxBudgetMinor?:Money preferredPropertyId?:UUID notes nextFollowUpDate?:Date noticeAcknowledgmentId?:UUID consentId?:UUID','stage:LeadStage duplicateWarnings:DuplicateWarning[] customerId?:UUID bookingId?:UUID','salesExecutiveId stage overdue','nextFollowUpDate updatedAt'],
 ['LeadActivity','/leads/{id}/activities','LC','kind:note|call|follow-up notes occurredAt:Timestamp nextFollowUpDate?:Date','projectId:UUID','','occurredAt'],
 ['SiteVisit','/site-visits','LCRU','leadId:UUID projectId:UUID propertyId?:UUID salesExecutiveId:UUID scheduledAt:Timestamp notes','status:scheduled|completed|cancelled actualAt?:Timestamp outcome?','leadId salesExecutiveId','scheduledAt'],
 ['Customer','/customers','LCRU','projectId:UUID name contacts:Contact[] sourceLeadId?:UUID noticeAcknowledgmentId?:UUID consentId?:UUID','status:active|archived reservationIds:UUID[] bookingIds:UUID[] collectionSummary:CollectionSummary','sourceLeadId','name'],
 ['Reservation','/reservations','LCR','projectId:UUID propertyId:UUID customerId:UUID expiresAt?:Timestamp','status:active|converted|expired|cancelled expiresAt:Timestamp bookingId?:UUID','propertyId customerId','expiresAt'],
 ['Booking','/bookings','LCRU','projectId:UUID propertyId:UUID customerId:UUID salesExecutiveId:UUID reservationId?:UUID sourceLeadId?:UUID agreedPriceMinor:Money depositRequiredMinor:Money agreementDate:Date installments:InstallmentInput[] cancellationTermsId?:UUID handoverRequiredMinor:Money','status:draft|active|cancellation-requested|cancelled saleStatus:unconfirmed|confirmed collectionSummary:CollectionSummary cancellation?:Cancellation refundSummary:RefundSummary handover?:Handover saleMilestone?:SaleMilestone','customerId propertyId salesExecutiveId','agreementDate'],
 ['Document','/documents','LR','','projectId:UUID entityType entityId:UUID originalName mimeType:application/pdf|image/jpeg|image/png|image/webp sizeBytes:ExactInteger sha256 status:quarantine|active|archived','entityType entityId mimeType','createdAt originalName'],
 ['AuditEvent','/audit-events','LR','','actorId:UUID occurredAt:Timestamp action projectId?:UUID recordType recordId:UUID requestId:UUID reason? changes:Change[]','actorId action recordType recordId','occurredAt'],
 ['Reconciliation','/reconciliations','LCR','projectId:UUID periodStart:Date periodEnd:Date documentIds:UUID[]','status:draft|reviewed|approved generatedAt:Timestamp sourceVersions:SourceVersion[] lines:ReconciliationLine[] totals:CostTotals reviews:Review[]','period','createdAt'],
 ['PrivacyRequest','/privacy-requests','LCR','subjectType subjectId:UUID kind:access|correction|anonymization reason','status:submitted|approved|rejected|completed review?:PrivacyReview completedAt?:Timestamp','','createdAt'],
 ['RetentionReview','/retention-reviews','LCR','asOfDate:Date projectId?:UUID','candidates:RetentionCandidate[] status:pending|completed','','createdAt'],
];
const meta = schemas.Metadata.properties;
for (const [name, path, ops, editable, additions] of resources) {
  define(`${name}Input`, editable);
  schemas[`${name}Update`] = object({ version: ref('Version'), ...fields(editable) });
  schemas[name] = object({ ...meta, ...fields(editable), ...fields(additions), ...schemas.Actions.properties });
  if (name === 'Project') { delete schemas.Project.properties.initialSites; delete schemas.ProjectUpdate.properties.initialSites; }
  if (name === 'Property') delete schemas.PropertyUpdate.properties.askingPriceMinor;
  for (const key of [name, `${name}Update`]) schemas[key].required = Object.keys(schemas[key].properties);
}
// Named least-data projections. Omitted confidential fields are not nullable placeholders.
function projection(name, base, keep) { schemas[name] = object(Object.fromEntries(keep.split(' ').map(k => [k, schemas[base].properties[k]]))); }
projection('WorkOrderSiteEngineer', 'WorkOrder', 'id createdAt updatedAt createdBy version projectId contractorId scope startDate endDate managerId status completionPercent documentIds allowedActions');
projection('PropertyConstruction', 'Property', 'id createdAt updatedAt createdBy version projectId siteId unitCode block floor propertyType area areaUnit description constructionStatus images allowedActions');
projection('InvoiceProcurement', 'Invoice', 'id createdAt updatedAt createdBy version projectId number supplierReference status documentIds allowedActions');
projection('ProjectSelector', 'Project', 'id code name status allowedActions');
projection('CustomerAssignedSales', 'Customer', Object.keys(schemas.Customer.properties).join(' '));
define('FinancialSummaryProjectManager', 'projectId:UUID approvedCostMinor:Money payableMinor:Money');

define('IdentityUser', 'id:UUID displayName email:Email role:Role');
define('Identity', 'user:IdentityUser projectIds:UUID[] capabilities:Text[] accessRevision:Version landingPath setupRequired:Boolean csrfToken idleExpiresAt:Timestamp absoluteExpiresAt:Timestamp');
define('StaffInput', 'displayName email:Email role:Role projectIds:UUID[]');
merge('Staff', 'Metadata', 'StaffInput', 'Actions');
schemas.Staff.properties.status = en('setup-required|active|deactivated'); schemas.Staff.required.push('status');
define('StaffUpdate', 'version:Version displayName email:Email role:Role');
merge('StaffCreated', 'Staff');
Object.assign(schemas.StaffCreated.properties, fields('setupLink expiresAt:Timestamp')); schemas.StaffCreated.required = Object.keys(schemas.StaffCreated.properties);
define('RoleDefinition', 'code:Role name capabilities:Text[]');
define('Csrf', 'csrfToken expiresAt:Timestamp');
define('PasswordChanged', 'passwordChanged:Boolean signInRequired:Boolean');
schemas.PasswordChanged.properties.passwordChanged.const = true; schemas.PasswordChanged.properties.signInRequired.const = true;
define('RecoveryGuidance', 'instructions contactLabel');
define('PasswordResetIssued', 'userId:UUID resetLink expiresAt:Timestamp');
define('HealthLive', 'status:ok'); define('HealthReady', 'status:ready');
define('StorageStatus', 'status:ready|unavailable');
define('BackupStatus', 'status:unavailable|healthy|degraded lastSuccessAt?:Timestamp nextDueAt?:Timestamp diagnosticId?:UUID');
define('OperatorStatus', 'version startedAt:Timestamp ready:Boolean database:StorageStatus attachments:StorageStatus backup:BackupStatus');
define('PlanMilestoneInput', 'id?:UUID name responsibleId:UUID startDate:Date endDate:Date status:PlanStatus completionPercent:Percent weightPercent:Percent');
define('Plan', 'projectId:UUID version:Version milestones:Milestone[] progressPercent?:Percent progressState:configured|unconfigured');
define('Link', 'label path');
define('ReportFilters', 'projectId?:UUID from?:Date to?:Date view?:Text');
define('Metric', 'metricKey value:Scalar unit formula sourceReport:ReportName sourceFilters:ReportFilters generatedAt:Timestamp');
define('ProjectSummary', 'project:Project sites:Site[] progressPercent?:Percent progressState:configured|unconfigured approvedActivity:SiteLog[] metrics:Metric[] links:Link[]');
define('LedgerFilter', 'projectId:UUID siteId:UUID materialId:UUID');
define('InventoryBalance', 'siteId:UUID materialId:UUID unit balanceQuantity:Quantity minimumQuantity:Quantity lowStock:Boolean ledgerFilter:LedgerFilter');
define('SiteMinimum', 'version:Version projectId:UUID siteId:UUID materialId:UUID minimumQuantity:Quantity');
define('PurchaseAmendmentLine', 'lineId:UUID quantity:Quantity unitPrice:UnitPrice categoryId:UUID');
define('PurchaseAmendment', 'id:UUID purchaseOrderId:UUID version:Version status:submitted reason lines:PurchaseAmendmentLine[] deliveryDate:Date');
define('StockEntry', 'id:UUID siteId:UUID materialId:UUID unit quantity:SignedQuantity');
define('StockTransaction', 'id:UUID projectId:UUID type:issue|return|transfer|adjustment entries:StockEntry[] actorId:UUID occurredAt:Timestamp reason');
define('AdjustmentInput', 'projectId:UUID siteId:UUID materialId:UUID quantityDelta:SignedQuantity unit reason');
merge('Adjustment', 'Metadata', 'AdjustmentInput', 'Actions');
schemas.Adjustment.properties.status = en('submitted|approved|rejected'); schemas.Adjustment.required.push('status');
define('CompletionMilestone', 'id:UUID completionPercent:Percent documentIds:UUID[]');
define('WorkAmendmentMilestone', 'id:UUID name dueDate:Date amountMinor:Money categoryId:UUID');
define('WorkAmendment', 'id:UUID workOrderId:UUID version:Version status:submitted reason milestones:WorkAmendmentMilestone[] endDate:Date');
define('AllocationInput', 'invoiceId:UUID amountMinor:Money');
define('ConversionCustomer', 'name contacts:Contact[]');
define('ConversionBooking', 'propertyId:UUID reservationId?:UUID agreedPriceMinor:Money depositRequiredMinor:Money agreementDate:Date installments:InstallmentInput[] cancellationTermsId?:UUID handoverRequiredMinor:Money');
define('LeadConversion', 'leadId:UUID customer:Customer booking:Booking');
define('ReservationCancellation', 'id:UUID reservationId:UUID version:Version status:pending-review reason');
define('CancellationBand', 'fromDate:Date toDate?:Date deductionType:fixed|percentage fixedMinor?:Money percentageBps?:BasisPoints');
schemas.CancellationBand.oneOf = [
  { properties: { deductionType: { const: 'fixed' }, fixedMinor: ref('Money'), percentageBps: { type: 'null' } } },
  { properties: { deductionType: { const: 'percentage' }, fixedMinor: { type: 'null' }, percentageBps: ref('BasisPoints') } },
];
define('CancellationTerms', 'id:UUID version:Version status:submitted bands:CancellationBand[] proposedBy:UUID');
define('RefundProposal', 'id:UUID version:Version status:submitted|approved|rejected amountMinor:Money remainingRefundableMinor:Money calculation:RefundCalculation decision?:Review');
define('Corrections', 'name? email?:Email phone?');
define('RetentionHold', 'id:UUID version:Version recordType recordId:UUID reason reviewAt?:Timestamp status:active|released');
define('Branding', 'wordmark logoDocumentId?:UUID');
define('ReferenceValue', 'key value');
define('ChecklistItem', 'id:UUID label required:Boolean');
define('Checklist', 'id:UUID name items:ChecklistItem[]');
define('OperatingPolicy', 'countries:Text[] noticeVersion? retentionScheduleVersion? confirmedAt?:Timestamp');
define('OperatingPolicyInput', 'countries:Text[] noticeVersion? retentionScheduleVersion?');
define('Settings', 'version:Version companyName branding:Branding currencyCode?:Text currencyMinorDigits?:Integer currencyLocked:Boolean timezone timezoneConfirmed:Boolean recoveryInstructions recoveryContactLabel reservationHours:Integer referenceValues:ReferenceValue[] inspectionChecklists:Checklist[] propertyTypes:Text[] operatingPolicy:OperatingPolicy');
schemas.SettingsInput = structuredClone(schemas.Settings);
delete schemas.SettingsInput.properties.currencyLocked;
schemas.SettingsInput.properties.operatingPolicy = ref('OperatingPolicyInput');
schemas.SettingsInput.properties.confirmOperationalPolicy = ref('Boolean');
schemas.SettingsInput.required = Object.keys(schemas.SettingsInput.properties);
define('StockMinimumOverride', 'siteId?:UUID materialId:UUID minimumQuantity:Quantity');
define('OverridesInput', 'version:Version reservationHours?:Integer stockMinimums:StockMinimumOverride[] inspectionChecklistIds:UUID[] referenceValues:ReferenceValue[]');
merge('Overrides', 'OverridesInput'); schemas.Overrides.properties.projectId = ref('UUID'); schemas.Overrides.required.push('projectId');
define('PolicyAcknowledgmentInput', 'subjectType subjectId:UUID noticeVersion acknowledgedAt:Timestamp consentRequired:Boolean consentGiven?:Boolean recordedByReason');
merge('PolicyAcknowledgment', 'PolicyAcknowledgmentInput'); Object.assign(schemas.PolicyAcknowledgment.properties, fields('id:UUID createdAt:Timestamp')); schemas.PolicyAcknowledgment.required = Object.keys(schemas.PolicyAcknowledgment.properties);
define('OverdueItem', 'resource id:UUID projectId:UUID label dueDate:Date path');
define('Dashboard', 'generatedAt:Timestamp metrics:Metric[] projectSummaries:ProjectSummary[] overdueItems:OverdueItem[] lowStockItems:InventoryBalance[]');
define('ReportColumn', 'key label type:text|date|timestamp|money|quantity|percent|integer|boolean');
define('ReportSnapshot', 'id:UUID report:ReportName generatedAt:Timestamp filters:ReportFilters totals:ReportTotals sourceVersions:SourceVersion[] documentId:UUID');
define('ReportTotals', 'count:ExactInteger amountMinor?:SignedMoney outstandingMinor?:Money actualMinor?:Money commitmentMinor?:Money varianceMinor?:SignedMoney quantity?:SignedQuantity');
define('CustomerBalanceRow', 'projectId:UUID customerId:UUID name invoicedMinor:Money collectedMinor:Money outstandingMinor:Money');
define('CommitmentRow', 'projectId:UUID sourceType:purchase-order|work-order sourceId:UUID lineId:UUID approvedMinor:Money recognizedMinor:Money remainingMinor:Money');
define('CostRow', 'projectId:UUID sourceCostId:UUID categoryId:UUID amountMinor:Money sourceType sourceId:UUID');
define('VarianceRow', 'projectId:UUID categoryId:UUID budgetMinor:Money actualMinor:Money commitmentMinor:Money varianceMinor:SignedMoney');
const reportViews = {
 progress: { milestones: 'Milestone', overdue: 'Task', inspections: 'Inspection' },
 procurement: { orders: 'PurchaseOrder', outstanding: 'PurchaseOrder' },
 budget: { variance: 'VarianceRow', costs: 'CostRow', commitments: 'CommitmentRow', reconciliation: 'Reconciliation' },
 inventory: { balances: 'InventoryBalance', movements: 'StockMovement', 'low-stock': 'InventoryBalance' },
 sales: { availability: 'Property', 'follow-ups': 'Lead', bookings: 'Booking' },
 collections: { receivables: 'Invoice', payables: 'Invoice', 'customer-balances': 'CustomerBalanceRow' },
};
const reportSchemaNames = [];
for (const [report, views] of Object.entries(reportViews)) for (const [view, row] of Object.entries(views)) {
  const name = `${report}_${view.replaceAll('-', '_')}_Report`;
  define(name, `generatedAt:Timestamp filters:ReportFilters formulae:Text[] columns:ReportColumn[] rows:${row}[] totals:ReportTotals page:Page sourceVersions:SourceVersion[]`);
  reportSchemaNames.push(name);
}
schemas.Report = { anyOf: reportSchemaNames.map(ref), description: 'Select the named schema for the report/view. Role policy further projects source rows before aggregation and pagination.' };

export const paths = {};
const groupPolicies = {
 auth: 'Anonymous only for csrf/login/reset-password/recovery-guidance; current authenticated identity for me/change-password; logout clears expired sessions too.',
 users: 'Administrator only. Final active Administrator guard. Access changes revoke sessions.', roles: 'Administrator only.',
 projects: 'Project Manager/Administrator manage; other roles receive only authorized operational selectors/plans/context. Management Viewer uses reports only. Site Engineer assigned task progress only.',
 'site-logs': 'Site Engineer creates/edits own draft or returned logs. Project Manager/Administrator independently review. Finance evidence only through permitted cost projections.',
 inspections: 'Site Engineer creates/updates own assigned checks. Project Manager/Administrator review independently. Failed items require linked issues.',
 issues: 'Assigned owner performs issue actions; Project Manager/Administrator manage and independently reopen.',
 materials: 'Procurement Officer/Administrator manage; Site Engineer operational selector; Project Manager/Finance scoped read.',
 suppliers: 'Procurement Officer/Administrator manage; Project Manager/Finance scoped read.',
 'purchase-orders': 'Procurement Officer/Project Manager/Administrator create and submit. Independent Project Manager approves Procurement Officer submissions; independent Finance approves Project Manager submissions; Administrator remains independent.',
 'goods-receipts': 'Procurement Officer/Administrator manage and post; related evidence only for permitted readers.',
 inventory: 'Procurement Officer/Administrator maintain minimums; scoped operational readers see quantities, never monetary stock valuation.',
 'stock-movements': 'Procurement Officer/Administrator operations; Site Engineer issue/return only. Adjustment approval requires independent Administrator. Both transfer sites authorized.',
 contractors: 'Project Manager/Administrator manage; Finance reads; Site Engineer assigned evidence only, no prices.',
 'work-orders': 'Project Manager/Administrator manage and submit; independent Finance/Administrator approves; Site Engineer assigned completion evidence excludes prices.',
 budgets: 'Finance/Administrator manage and independently approve; Project Manager reads.',
 expenses: 'Assigned staff submit own expense/evidence without project totals; Finance/Administrator independently review.',
 invoices: 'Finance/Administrator manage; Project Manager approved-cost/payable summary only; Procurement order/receipt evidence only; Sales collection projections only, never finance mutation routes.',
 payments: 'Finance/Administrator manage; Sales assigned customer collection projections only.',
 properties: 'Assigned Sales/Administrator manage; Project Manager construction/checklist actions; Site Engineer assigned inspection evidence only; Finance payment checks.',
 leads: 'Assigned Sales/Administrator only. Related customers and balances remain assignment scoped.',
 'site-visits': 'Assigned Sales/Administrator only.', customers: 'Assigned Sales/Administrator manage; Finance permitted financial evidence.',
 reservations: 'Assigned Sales/Administrator acquire; cancellation requires independent Administrator approval and retains claim until approved.',
 bookings: 'Assigned Sales/Administrator manage sales. Finance proposes terms/refunds and verifies payments; independent Administrator approves terms/cancellation/refunds. Finance/Administrator independent of booking creator confirms sale. Project Manager checklist, Finance clearance, Sales handover date.',
 documents: 'Inherit most restrictive typed-parent policy and project scope. No access by arbitrary document ID.',
 'audit-events': 'Administrator search; Project Manager assigned approval trails; Finance relevant financial trails; no mutation.',
 reconciliations: 'Project Manager and Finance independently sign off, actual server role checked.',
 'privacy-requests': 'Administrator only; reviewed field allowlist and legal holds.', 'retention-reviews': 'Administrator only; eligibility and legal holds rechecked.', 'retention-holds': 'Administrator only.',
 settings: 'Administrator only. Currency locks after first financial posting; operating policy confirmation required.',
 'policy-acknowledgments': 'Authorized subject record owner; project/assignment and configured consent policy.',
 dashboard: 'Authorized role/project summary only. Financial metrics omitted for unauthorized roles. Management Viewer reports only.',
 reports: 'Authorized project and role views from policy matrix. Sales assigned records only; Management Viewer report summaries only. No operational access inferred from report access.',
 health: 'Anonymous sanitized health only; no internal paths or database details.', operator: 'Administrator only.',
};
const errorCodes = {400:'MALFORMED_REQUEST',401:'AUTH_REQUIRED',403:'FORBIDDEN',404:'NOT_FOUND',409:'STALE_VERSION',413:'BODY_TOO_LARGE',415:'UNSUPPORTED_MEDIA_TYPE',422:'VALIDATION_FAILED',429:'RATE_LIMITED',500:'INTERNAL_ERROR',503:'SERVICE_UNAVAILABLE'};
const exampleUUID = '00000000-0000-4000-8000-000000000001';
function example(schema, seen = []) {
  if (schema.$ref) { const name = schema.$ref.split('/').at(-1); if (seen.includes(name)) return null; return example(schemas[name], [...seen, name]); }
  if (schema.const !== undefined) return schema.const;
  if (schema.examples) return schema.examples[0];
  if (schema.enum) return schema.enum[0];
  if (schema.anyOf) return schema.anyOf.some(s => s.type === 'null') ? null : example(schema.anyOf[0], seen);
  if (schema.type === 'object') return Object.fromEntries((schema.required ?? []).map(k => [k, example(schema.properties[k], seen)]));
  if (schema.type === 'array') return [];
  if (schema.type === 'boolean') return false;
  if (schema.type === 'integer') return schema.minimum || 1;
  if (Array.isArray(schema.type)) return null;
  if (schema.format === 'uuid') return exampleUUID;
  if (schema.format === 'date') return '2026-01-01';
  if (schema.format === 'date-time') return '2026-01-01T00:00:00.000Z';
  if (schema.format === 'email') return 'person@example.invalid';
  if (schema.pattern) return '0';
  if (schema.minLength) return 'example-value-not-a-real-secret'.padEnd(schema.minLength, 'x');
  return 'example';
}
function envelope(name, mutation = false, list = false) {
 const id = `${name}${list ? 'List' : mutation ? 'Mutation' : 'Response'}`;
 const projections = {WorkOrder:['WorkOrder','WorkOrderSiteEngineer'],Property:['Property','PropertyConstruction'],Invoice:['Invoice','InvoiceProcurement'],Project:['Project','ProjectSelector'],Customer:['Customer','CustomerAssignedSales']};
 const data = projections[name] ? { anyOf:projections[name].map(ref), description:'Server selects the named authorized role projection, never sends confidential fields to unauthorized roles.' } : ref(name);
 if (!schemas[id]) schemas[id] = object({ data: list ? array(data) : data, ...(list ? { page: ref('Page') } : {}), ...(mutation ? { effects: ref('Effects') } : {}), requestId: ref('UUID') });
 return ref(id);
}
const queryKinds = { projectId:'UUID',cursor:'Text',limit:'Integer',from:'Date',to:'Date',dueBefore:'Date',date:'Date',overdue:'Boolean',lowStock:'Boolean',reviewQueue:'Boolean',minPriceMinor:'Money',maxPriceMinor:'Money' };
function query(name, sort) {
 let schema = ref(queryKinds[name] || (name.endsWith('Id') ? 'UUID' : 'Text'));
 if (name === 'limit') schema = { type:'integer',minimum:1,maximum:100,default:25 };
 if (name === 'sort') schema = en(sort.split(' ').join('|'));
 if (name === 'direction') schema = en('asc|desc');
 return { name, in:'query', required:false, schema };
}
function operation(method, path, input, output, options = {}) {
 const mutation = method !== 'get';
 const group = path.split('/')[1];
 const operationId = `${method}_${path.replace(/[{}]/g, '').replace(/[^a-zA-Z0-9]+/g, '_').replace(/^_|_$/g, '')}`;
 const parameters = [...path.matchAll(/\{([^}]+)\}/g)].map(([,name]) => ({ name, in:'path',required:true,schema:name === 'report' ? ref('ReportName') : ref('UUID') }));
 if (mutation) parameters.push({ name:'Idempotency-Key',in:'header',required:true,schema:ref('UUID'),description:'Reuse only for an identical deliberate retry; scoped to principal and operation. Changed payload returns 409 IDEMPOTENCY_MISMATCH.' }, { name:'X-CSRF-Token',in:'header',required:true,schema:{type:'string',minLength:1},description:'Signed nonce matching buildestate_csrf cookie, exact same-origin Origin required.' });
 if (options.query) parameters.push(...[...new Set(options.query.split(' ').filter(Boolean))].filter(n => !parameters.some(p => p.name === n)).map(n => query(n,options.sort || 'updatedAt')));
 const code = options.status || (options.create ? 201 : 200);
 const result = options.raw ? ref(output) : envelope(output, mutation, options.list);
 const responses = Object.fromEntries(Object.entries(errorCodes).map(([status, error]) => [status, { description: `${error}; see shared stable error code registry. Inaccessible IDs are indistinguishable from absent IDs.`, ...(status === '429' ? {headers:{'Retry-After':{schema:{type:'integer',minimum:1},description:'Seconds until deliberate retry.'}}} : {}), content:{'application/json':{schema:ref('Error'),example:{error:{code:error,message:'The request could not be completed.'},requestId:exampleUUID}}} }]));
 responses[code] = code === 204 ? {description:'Cookies cleared; no body.'} : { description:'Designed success contract; not a claim of implemented behavior.',content:{'application/json':{schema:result,example:example(result)}} };
 const op = { operationId,tags:[group], summary:options.summary || `${method.toUpperCase()} ${path}`, description:`${groupPolicies[group]} All related IDs are scope checked. Unknown request fields and unsupported query parameters fail validation. Nullable request fields may be omitted or null; response fields use explicit null. Collections apply policy before filtering, pagination and aggregation. ${options.description || ''}`, 'x-permission-policy':options.policy || groupPolicies[group], 'x-implementation-status':group === 'health' ? 'foundation' : 'unavailable', 'x-capability': options.capability || `${group.replace(/-([a-z])/g,(_,c)=>c.toUpperCase())}.${options.action || (method==='get'?'read':method==='put'?'manage':'create')}`, parameters, responses };
 if (['health'].includes(group) || (group === 'auth' && ['csrf','login','reset-password','recovery-guidance','logout'].includes(path.split('/').at(-1)))) op.security = [];
 if (input && mutation) {
   let request = typeof input === 'string' && schemas[input] ? ref(input) : define(`${operationId}Input`, input);
   op.requestBody = {required:true,content:{'application/json':{schema:request,example:example(request)}}};
   op['x-body-limit-bytes'] = 1048576;
 }
 paths[path] ??= {};
 if (paths[path][method]) throw new Error(`Duplicate operation ${method} ${path}`);
 paths[path][method] = op;
 return op;
}
for (const [name,path,ops,, , filters,sort] of resources) {
 if (ops.includes('L')) operation('get',path,null,name,{list:true,query:`cursor limit q sort direction projectId status from to ${filters}`,sort});
 if (ops.includes('C')) operation('post',path,`${name}Input`,name,{create:true});
 const detail = `${path}/{${name === 'Site' ? 'siteId' : name === 'Task' ? 'taskId' : 'id'}}`;
 if (ops.includes('R')) operation('get',detail,null,name);
 if (ops.includes('U')) operation('put',detail,`${name}Update`,name);
}
// Explicit auth/admin methods: no generic public staff registration.
operation('get','/auth/csrf',null,'Csrf');
operation('post','/auth/login','email:Email password:Text','Identity',{description:'Invalid/inactive/unknown accounts return equal-shaped 401 INVALID_CREDENTIALS. Rotate session and CSRF cookies; no domain inputs accepted.'});
operation('get','/auth/me',null,'Identity');
operation('post','/auth/logout','', 'Identity',{status:204});
operation('post','/auth/change-password','currentPassword:Text newPassword:Password','PasswordChanged');
operation('post','/auth/reset-password','token newPassword:Password','PasswordChanged',{description:'Invalid, expired, reused and revoked tokens uniformly return 422 INVALID_RESET_TOKEN.'});
operation('get','/auth/recovery-guidance',null,'RecoveryGuidance');
operation('get','/users',null,'Staff',{list:true,query:'cursor limit q sort direction status role',sort:'displayName email updatedAt'});
operation('get','/users/{id}',null,'Staff');
operation('post','/users','StaffInput','StaffCreated',{create:true});
operation('put','/users/{id}','StaffUpdate','Staff');
operation('put','/users/{id}/projects','version:Version projectIds:UUID[]','Staff');
for (const action of ['deactivate','reactivate']) operation('post',`/users/{id}/${action}`,'ReasonInput','Staff',{action});
operation('post','/users/{id}/password-reset','version:Version purpose:setup|recovery identityVerified:Boolean reason','PasswordResetIssued');
define('Roles','roles:RoleDefinition[]'); operation('get','/roles',null,'Roles');
operation('get','/health/live',null,'HealthLive'); operation('get','/health/ready',null,'HealthReady');
operation('get','/operator/status',null,'OperatorStatus');
function action(path,input,output,options={}) { return operation(options.method || 'post',path,input,output,{...options,action:path.split('/').at(-1).replace(/-([a-z])/g,(_,c)=>c.toUpperCase())}); }
function actions(base,names,input,output) { for (const name of names.split(' ')) action(`${base}/${name}`,input,output); }
action('/projects/{id}/transition','version:Version status:draft|active|on-hold|completed|archived reason','Project');
action('/projects/{id}/archive','ReasonInput','Project'); action('/projects/{id}/sites/{siteId}/archive','ReasonInput','Site');
action('/projects/{id}/plan','version:Version milestones:PlanMilestoneInput[] reason','Plan',{method:'put',capability:'plan.manage'});
action('/projects/{id}/tasks/{taskId}/progress','version:Version completionPercent:Percent status:PlanStatus notes','Task',{capability:'tasks.progress'});
operation('get','/projects/{id}/summary',null,'ProjectSummary');
actions('/site-logs/{id}','submit approve','VersionInput','SiteLog'); action('/site-logs/{id}/return','ReasonInput','SiteLog');
actions('/inspections/{id}','submit complete','VersionInput','Inspection');
action('/inspections/{id}/items/{itemId}/issue','version:Version title description ownerId:UUID priority:Priority dueDate:Date','Issue');
action('/issues/{id}/start','VersionInput','Issue'); action('/issues/{id}/resolve','version:Version resolutionNotes documentIds:UUID[]','Issue'); action('/issues/{id}/reopen','ReasonInput','Issue');
for (const [path,name] of [['materials','Material'],['suppliers','Supplier'],['contractors','Contractor'],['customers','Customer'],['documents','Document']]) action(`/${path}/{id}/archive`,'ReasonInput',name);
operation('get','/inventory/balances',null,'InventoryBalance',{list:true,query:'projectId siteId materialId lowStock cursor limit'});
action('/inventory/minimums','SiteMinimum','SiteMinimum',{method:'put'});
actions('/purchase-orders/{id}','submit approve','VersionInput','PurchaseOrder'); actions('/purchase-orders/{id}','return cancel','ReasonInput','PurchaseOrder');
action('/purchase-orders/{id}/amendments','version:Version reason deliveryDate:Date lines:PurchaseAmendmentLine[]','PurchaseAmendment');
actions('/purchase-orders/{id}/amendments/{amendmentId}','approve reject','ReasonInput','PurchaseOrder');
action('/goods-receipts/{id}/post','VersionInput','GoodsReceipt'); action('/goods-receipts/{id}/reverse','ReasonInput','GoodsReceipt');
action('/stock-movements/issue','projectId:UUID siteId:UUID materialId:UUID quantity:Quantity unit activityType:site-log|task|work-order activityId:UUID reason','StockTransaction');
action('/stock-movements/return','projectId:UUID originalMovementId:UUID quantity:Quantity unit reason','StockTransaction');
action('/stock-movements/transfer','projectId:UUID fromSiteId:UUID toSiteId:UUID materialId:UUID quantity:Quantity unit reason','StockTransaction');
action('/stock-movements/adjustments','AdjustmentInput','Adjustment');
actions('/stock-movements/adjustments/{id}','approve reject','ReasonInput','Adjustment');
operation('get','/stock-movements/adjustments',null,'Adjustment',{list:true,query:'cursor limit q sort direction projectId status from to',sort:'updatedAt'});
actions('/work-orders/{id}','submit approve','VersionInput','WorkOrder'); actions('/work-orders/{id}','return cancel','ReasonInput','WorkOrder');
action('/work-orders/{id}/completion','version:Version milestones:CompletionMilestone[] reason','WorkOrder');
action('/work-orders/{id}/amendments','version:Version reason milestones:WorkAmendmentMilestone[] endDate:Date','WorkAmendment');
actions('/work-orders/{id}/amendments/{amendmentId}','approve reject','ReasonInput','WorkOrder');
action('/budgets/{id}/revisions','version:Version reason lines:BudgetLineInput[]','Budget'); action('/budgets/{id}/approve','VersionInput','Budget');
actions('/expenses/{id}','submit approve','VersionInput','Expense'); action('/expenses/{id}/reject','ReasonInput','Expense');
action('/invoices/{id}/duplicate-review','version:Version reason relatedInvoiceIds:UUID[] documentIds:UUID[]','Invoice');
for (const [path,name] of [['invoices','Invoice'],['payments','Payment']]) {action(`/${path}/{id}/post`,'VersionInput',name);action(`/${path}/{id}/reverse`,'ReasonInput',name);}
action('/payments/{id}/allocations','version:Version allocations:AllocationInput[]','Payment'); action('/payments/{id}/allocations/{allocationId}/reverse','ReasonInput','Payment');
action('/properties/{id}/price','version:Version askingPriceMinor:Money reason','Property'); action('/properties/{id}/availability','version:Version availability:available|unavailable reason','Property');
action('/properties/{id}/images','version:Version images:Image[]','Property',{method:'put'});
action('/leads/{id}/stage','version:Version stage:LeadStage notes nextFollowUpDate?:Date','Lead');
action('/leads/{id}/convert','version:Version customerId?:UUID customer?:ConversionCustomer booking:ConversionBooking','LeadConversion');
action('/site-visits/{id}/complete','version:Version actualAt:Timestamp outcome notes','SiteVisit'); action('/site-visits/{id}/cancel','ReasonInput','SiteVisit');
action('/reservations/{id}/cancel','ReasonInput','ReservationCancellation'); actions('/reservations/{id}/cancellation','approve reject','ReasonInput','Reservation');
action('/bookings/{id}/activate','VersionInput','Booking');
operation('get','/bookings/{id}/installments',null,'Installment',{list:true,query:'cursor limit',sort:'dueDate'});
action('/bookings/{id}/installments','version:Version installments:InstallmentInput[] reason','Booking',{method:'put'});
action('/bookings/{id}/installments/{installmentId}/invoice','version:Version number taxMinor:Money documentIds:UUID[]','Invoice');
action('/bookings/{id}/cancellation-terms','version:Version bands:CancellationBand[] reason','CancellationTerms');
actions('/bookings/{id}/cancellation-terms/{termsId}','approve reject','ReasonInput','Booking');
action('/bookings/{id}/cancellation','ReasonInput','Booking'); actions('/bookings/{id}/cancellation','approve reject','ReasonInput','Booking');
action('/bookings/{id}/refund-proposals','version:Version amountMinor:Money reason','RefundProposal');
actions('/bookings/{id}/refund-proposals/{refundId}','approve reject','ReasonInput','RefundProposal');
action('/bookings/{id}/refunds','version:Version refundProposalId:UUID amountMinor:Money originalPaymentIds:UUID[] date:Date method:PaymentMethod reference documentIds:UUID[]','Payment');
action('/bookings/{id}/sale-milestone','version:Version agreementDocumentId:UUID milestoneDate:Date requiredPaymentMinor:Money notes','Booking');
action('/bookings/{id}/payment-verification','version:Version purpose:sale|handover allocationIds:UUID[] documentIds:UUID[] notes','Booking');
action('/bookings/{id}/confirm-sale','VersionInput','Booking',{capability:'bookings.confirmSale'});
action('/bookings/{id}/handover-checklist','version:Version items:HandoverItemInput[] reason','Booking',{method:'put'});
action('/bookings/{id}/handover','version:Version handoverDate:Date notes','Booking');
action('/reconciliations/{id}/lines/{lineId}/review','version:Version resolution reason sourceCostId?:UUID documentIds:UUID[]','Reconciliation');
action('/reconciliations/{id}/approve','version:Version role:project-manager|finance-officer reason','Reconciliation');
action('/privacy-requests/{id}/review','version:Version decision:approve|reject reason corrections?:Corrections','PrivacyRequest');
action('/privacy-requests/{id}/execute','VersionInput','PrivacyRequest');
action('/retention-holds','recordType recordId:UUID reason reviewAt?:Timestamp','RetentionHold');
action('/retention-holds/{id}/release','ReasonInput','RetentionHold');
action('/retention-reviews/{id}/execute','version:Version recordIds:UUID[] reason','RetentionReview');
define('UploadFileMetadata','clientFileId originalName');
define('UploadMetadata','projectId:UUID entityType entityId:UUID files:UploadFileMetadata[]');
schemas.UploadMetadata.properties.files.minItems = 1; schemas.UploadMetadata.properties.files.maxItems = 10;
define('UploadResult','documents:Document[]');
const upload = operation('post','/documents/uploads',null,'UploadResult',{create:true,description:'Whole-batch commit after content validation. PDF/JPEG/PNG/WebP only. Maximum 20,000,000 bytes per file, 10 files, 200,000,000 bytes total plus bounded multipart overhead. Parts are named by metadata.files[].clientFileId; reject extra/missing/duplicate parts.'});
upload.requestBody = {required:true,content:{'multipart/form-data':{schema:{type:'object',properties:{metadata:ref('UploadMetadata')},required:['metadata'],additionalProperties:{type:'string',format:'binary',maxLength:20000000},maxProperties:11},encoding:{metadata:{contentType:'application/json'}}}}};
for (const name of ['download','preview']) {
 const op = operation('get',`/documents/{id}/${name}`,null,'Document');
 op.responses['200'] = { description: name === 'download' ? 'Authorized verified bytes, safe attachment filename, private/no-store, nosniff.' : 'Re-encoded image or sandboxed PDF; restrictive CSP; incompatible content returns 415.',headers:{'Content-Disposition':{schema:{type:'string'}},'Cache-Control':{schema:{type:'string',const:'private, no-store'}},'X-Content-Type-Options':{schema:{type:'string',const:'nosniff'}}},content:Object.fromEntries(['application/pdf','image/jpeg','image/png','image/webp'].map(mime=>[mime,{schema:{type:'string',format:'binary'}}]))};
}
operation('get','/settings',null,'Settings'); action('/settings','SettingsInput','Settings',{method:'put'});
operation('get','/projects/{id}/overrides',null,'Overrides'); action('/projects/{id}/overrides','OverridesInput','Overrides',{method:'put'});
action('/policy-acknowledgments','PolicyAcknowledgmentInput','PolicyAcknowledgment',{create:true});
operation('get','/dashboard',null,'Dashboard',{query:'projectId'});
const reports = operation('get','/reports/{report}',null,'Report',{query:'projectId from to view cursor limit sort direction format',description:'Report/view allowlist and row schema are in x-report-views. Sort accepts only exposed column keys. Maximum CSV export 50,000 rows; larger exports return 422 EXPORT_TOO_LARGE. Escape formula-leading user text including leading whitespace/control characters; retain typed numeric representations.'});
reports['x-report-views'] = reportViews;
reports.parameters.find(p=>p.name==='format').schema = en('json|csv');
reports.parameters.find(p=>p.name==='view').schema = en([...new Set(Object.values(reportViews).flatMap(v=>Object.keys(v)))].join('|'));
reports.parameters.find(p=>p.name==='sort').schema = en([...new Set(Object.values(reportViews).flatMap(v=>Object.values(v)).flatMap(n=>Object.keys(schemas[n].properties)))].join('|'));
reports.responses['200'].content['text/csv'] = {schema:{type:'string'},example:'generatedAt,2026-01-01T00:00:00.000Z\r\n'};
action('/reports/{report}/snapshots','projectId:UUID from?:Date to?:Date view? documentIds:UUID[]','ReportSnapshot');

// Nullable input keys may be omitted; outputs always include explicit nulls.
// Resource create/update schemas are separate from response schemas, so no
// read-only field or allowedActions can be mass-assigned through a mutation.
const requestSchemas = new Set();
function collectRequest(schema) {
 if (schema.$ref) { const n=schema.$ref.split('/').at(-1); if(requestSchemas.has(n))return;requestSchemas.add(n);collectRequest(schemas[n]);return; }
 for (const child of Object.values(schema.properties || {})) collectRequest(child);
 for (const child of schema.anyOf || []) collectRequest(child);
 if(schema.items)collectRequest(schema.items);
}
for(const item of Object.values(paths)) for(const op of Object.values(item)) for(const media of Object.values(op.requestBody?.content || {})) collectRequest(media.schema);
// Do not mutate shared response components: optionality is modeled in request
// variants generated recursively, preserving nullable required output fields.
const cloned = new Set();
function requestVariant(schema) {
 if(schema.$ref) {
  const name=schema.$ref.split('/').at(-1), target=schemas[name];
  if(!target.properties && !target.items)return schema;
  const variant=`${name}Request`;
  if(!cloned.has(variant)){cloned.add(variant);schemas[variant]=requestVariant(structuredClone(target));}
  return ref(variant);
 }
 if(schema.properties){for(const [k,v] of Object.entries(schema.properties))schema.properties[k]=requestVariant(v);schema.required=(schema.required||[]).filter(k=>!schema.properties[k].anyOf?.some(v=>v.type==='null'));}
 if(schema.items)schema.items=requestVariant(schema.items);
 if(schema.anyOf)schema.anyOf=schema.anyOf.map(requestVariant);
 return schema;
}
for(const item of Object.values(paths)) for(const op of Object.values(item)) for(const media of Object.values(op.requestBody?.content || {})) media.schema=requestVariant(media.schema);

export const document = {
 openapi:'3.1.0',info:{title:'BuildEstate API',version:'1.0.0',description:'Versioned design contract, not an implementation-completeness claim. All business operations are unavailable foundation slots until their owning feature is implemented and contract-tested. /api/v1 is the compatibility boundary. Exact strings for money/quantity, strict input fields, explicit response nulls, server-only policy and atomic effects. See docs/api/README.md.'},
 servers:[{url:'/api/v1'}],security:[{sessionCookie:[]}],
 'x-contract-version':'1.0.0','x-json-body-limit-bytes':1048576,
 'x-policy-matrix':groupPolicies,
 'x-transaction-contract':'Current access revision, version/state, policy, related scopes, balance/claim checks, writes, audit and idempotency result commit atomically. Creator cannot approve own submission, including Administrator. No automatic write retry.',
 paths,components:{securitySchemes:{sessionCookie:{type:'apiKey',in:'cookie',name:'buildestate_session',description:'Opaque HttpOnly host-only SameSite=Lax cookie, Secure for HTTPS. Never bearer tokens in browser storage. Idle 30 minutes, absolute 12 hours.'}},schemas},
};
