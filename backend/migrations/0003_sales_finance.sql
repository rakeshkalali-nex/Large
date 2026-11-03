CREATE TABLE properties (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), site_id TEXT NOT NULL, unit_code TEXT NOT NULL, block TEXT NOT NULL, floor TEXT, property_type TEXT NOT NULL, area INTEGER NOT NULL CHECK(area > 0), area_unit TEXT NOT NULL,
 asking_price_minor INTEGER NOT NULL CHECK(asking_price_minor >= 0), description TEXT NOT NULL, construction_status TEXT NOT NULL CHECK(construction_status IN ('planned','in-progress','completed','on-hold')),
 availability TEXT NOT NULL DEFAULT 'available' CHECK(availability IN ('available','reserved','booked','sold','unavailable')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(site_id,project_id) REFERENCES sites(id,project_id), UNIQUE(project_id,unit_code), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX property_availability ON properties(project_id,availability,asking_price_minor,id);
CREATE TABLE property_price_history (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, property_id TEXT NOT NULL, old_price_minor INTEGER NOT NULL CHECK(old_price_minor >= 0), new_price_minor INTEGER NOT NULL CHECK(new_price_minor >= 0), reason TEXT NOT NULL CHECK(length(trim(reason)) > 0), created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL, FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id)
) STRICT;
CREATE TABLE leads (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), contact_name TEXT NOT NULL, contact_email TEXT, contact_phone TEXT, normalized_email TEXT, normalized_phone TEXT,
 source TEXT NOT NULL, sales_executive_id TEXT NOT NULL REFERENCES users(id), min_budget_minor INTEGER CHECK(min_budget_minor >= 0), max_budget_minor INTEGER CHECK(max_budget_minor >= 0), preferred_property_id TEXT,
 notes TEXT NOT NULL, next_follow_up_date TEXT, notice_acknowledgment_id TEXT REFERENCES policy_acknowledgments(id), consent_id TEXT REFERENCES policy_acknowledgments(id),
 stage TEXT NOT NULL DEFAULT 'new' CHECK(stage IN ('new','contacted','qualified','visit','booked','lost')), last_activity_at TEXT NOT NULL,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(preferred_property_id,project_id) REFERENCES properties(id,project_id), CHECK(min_budget_minor IS NULL OR max_budget_minor IS NULL OR min_budget_minor <= max_budget_minor), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX leads_follow_up ON leads(project_id,sales_executive_id,stage,next_follow_up_date,id);
CREATE INDEX leads_email ON leads(project_id,normalized_email);
CREATE INDEX leads_phone ON leads(project_id,normalized_phone);
CREATE TABLE lead_activities (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, lead_id TEXT NOT NULL, kind TEXT NOT NULL CHECK(kind IN ('note','call','follow-up','stage-change','conversion')),
 notes TEXT NOT NULL, occurred_at TEXT NOT NULL, next_follow_up_date TEXT, from_stage TEXT, to_stage TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), FOREIGN KEY(lead_id,project_id) REFERENCES leads(id,project_id)
) STRICT;
CREATE TABLE site_visits (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, lead_id TEXT NOT NULL, property_id TEXT, sales_executive_id TEXT NOT NULL REFERENCES users(id), scheduled_at TEXT NOT NULL, notes TEXT NOT NULL, actual_at TEXT, outcome TEXT,
 status TEXT NOT NULL DEFAULT 'scheduled' CHECK(status IN ('scheduled','completed','cancelled')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(lead_id,project_id) REFERENCES leads(id,project_id), FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id), CHECK(status <> 'completed' OR (actual_at IS NOT NULL AND outcome IS NOT NULL)), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE customers (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), name TEXT NOT NULL, source_lead_id TEXT, notice_acknowledgment_id TEXT REFERENCES policy_acknowledgments(id), consent_id TEXT REFERENCES policy_acknowledgments(id),
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(source_lead_id,project_id) REFERENCES leads(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE customer_contacts (id TEXT PRIMARY KEY, customer_id TEXT NOT NULL REFERENCES customers(id), name TEXT NOT NULL, email TEXT, phone TEXT, normalized_email TEXT, normalized_phone TEXT) STRICT;
CREATE INDEX customer_contact_email ON customer_contacts(normalized_email,customer_id);
CREATE INDEX customer_contact_phone ON customer_contacts(normalized_phone,customer_id);
CREATE TABLE reservations (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), property_id TEXT NOT NULL, customer_id TEXT NOT NULL, expires_at TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','pending-review','converted','expired','cancelled')), cancellation_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id), FOREIGN KEY(customer_id,project_id) REFERENCES customers(id,project_id), UNIQUE(id,project_id), UNIQUE(id,property_id,project_id)
) STRICT;
CREATE INDEX reservation_expiry ON reservations(status,expires_at,id);
CREATE TABLE bookings (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), property_id TEXT NOT NULL, customer_id TEXT NOT NULL, sales_executive_id TEXT NOT NULL REFERENCES users(id), reservation_id TEXT UNIQUE, source_lead_id TEXT,
 agreed_price_minor INTEGER NOT NULL CHECK(agreed_price_minor > 0), deposit_required_minor INTEGER NOT NULL CHECK(deposit_required_minor >= 0), agreement_date TEXT NOT NULL, cancellation_terms_id TEXT, handover_required_minor INTEGER NOT NULL CHECK(handover_required_minor >= 0),
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','active','cancellation-requested','cancelled')), sale_status TEXT NOT NULL DEFAULT 'unconfirmed' CHECK(sale_status IN ('unconfirmed','confirmed')),
 confirmed_by TEXT REFERENCES users(id), confirmed_at TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id), FOREIGN KEY(customer_id,project_id) REFERENCES customers(id,project_id), FOREIGN KEY(reservation_id,property_id,project_id) REFERENCES reservations(id,property_id,project_id), FOREIGN KEY(source_lead_id,project_id) REFERENCES leads(id,project_id), FOREIGN KEY(cancellation_terms_id,id) REFERENCES cancellation_terms(id,booking_id),
 CHECK(confirmed_by IS NULL OR confirmed_by <> created_by), CHECK(sale_status <> 'confirmed' OR (confirmed_by IS NOT NULL AND confirmed_at IS NOT NULL)), UNIQUE(id,project_id), UNIQUE(id,property_id,project_id)
) STRICT;
CREATE TABLE unit_claims (
 property_id TEXT PRIMARY KEY, project_id TEXT NOT NULL, reservation_id TEXT UNIQUE, booking_id TEXT UNIQUE, acquired_at TEXT NOT NULL,
 FOREIGN KEY(property_id,project_id) REFERENCES properties(id,project_id), FOREIGN KEY(reservation_id,property_id,project_id) REFERENCES reservations(id,property_id,project_id), FOREIGN KEY(booking_id,property_id,project_id) REFERENCES bookings(id,property_id,project_id),
 CHECK((reservation_id IS NOT NULL)+(booking_id IS NOT NULL)=1)
) STRICT;
CREATE TABLE installments (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL, label TEXT NOT NULL, due_date TEXT NOT NULL, amount_minor INTEGER NOT NULL CHECK(amount_minor > 0),
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), UNIQUE(id,booking_id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX installments_due ON installments(project_id,due_date,id);
CREATE TABLE cancellation_terms (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision > 0), reason TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'submitted' CHECK(status IN ('submitted','approved','rejected','superseded')), proposed_by TEXT NOT NULL REFERENCES users(id), decided_by TEXT REFERENCES users(id), decided_at TEXT, decision_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), UNIQUE(booking_id,revision), UNIQUE(id,booking_id), UNIQUE(id,project_id), CHECK(decided_by IS NULL OR decided_by <> proposed_by)
) STRICT;
CREATE TABLE cancellation_bands (
 id TEXT PRIMARY KEY, terms_id TEXT NOT NULL REFERENCES cancellation_terms(id), from_date TEXT NOT NULL, to_date TEXT,
 deduction_type TEXT NOT NULL CHECK(deduction_type IN ('fixed','percentage')), fixed_minor INTEGER CHECK(fixed_minor >= 0), percentage_bps INTEGER CHECK(percentage_bps BETWEEN 0 AND 10000),
 CHECK(to_date IS NULL OR from_date < to_date), CHECK((deduction_type='fixed' AND fixed_minor IS NOT NULL AND percentage_bps IS NULL) OR (deduction_type='percentage' AND percentage_bps IS NOT NULL AND fixed_minor IS NULL)), UNIQUE(id,terms_id)
) STRICT;
CREATE TABLE cancellations (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL, terms_id TEXT NOT NULL, band_id TEXT NOT NULL, request_date TEXT NOT NULL, reason TEXT NOT NULL CHECK(length(trim(reason)) > 0),
 status TEXT NOT NULL DEFAULT 'submitted' CHECK(status IN ('submitted','approved','rejected')), agreed_price_minor INTEGER NOT NULL CHECK(agreed_price_minor > 0), collections_minor INTEGER NOT NULL CHECK(collections_minor >= 0), deduction_minor INTEGER NOT NULL CHECK(deduction_minor >= 0), refunds_paid_minor INTEGER NOT NULL CHECK(refunds_paid_minor >= 0), refundable_cap_minor INTEGER NOT NULL CHECK(refundable_cap_minor >= 0),
 decided_by TEXT REFERENCES users(id), decided_at TEXT, decision_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), FOREIGN KEY(terms_id,booking_id) REFERENCES cancellation_terms(id,booking_id), FOREIGN KEY(band_id,terms_id) REFERENCES cancellation_bands(id,terms_id), CHECK(decided_by IS NULL OR decided_by <> created_by), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE invoices (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), number TEXT NOT NULL UNIQUE, type TEXT NOT NULL CHECK(type IN ('receivable','payable')),
 counterparty_type TEXT NOT NULL CHECK(counterparty_type IN ('customer','supplier','contractor')), customer_id TEXT, supplier_id TEXT REFERENCES suppliers(id), contractor_id TEXT REFERENCES contractors(id), booking_id TEXT, installment_id TEXT UNIQUE,
 supplier_reference TEXT, due_date TEXT NOT NULL, subtotal_minor INTEGER NOT NULL CHECK(subtotal_minor >= 0), tax_minor INTEGER NOT NULL CHECK(tax_minor >= 0), total_minor INTEGER NOT NULL CHECK(total_minor >= 0),
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','posted','reversed')), posted_at TEXT, reversal_id TEXT REFERENCES financial_reversals(id),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(customer_id,project_id) REFERENCES customers(id,project_id), FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), FOREIGN KEY(installment_id,booking_id,project_id) REFERENCES installments(id,booking_id,project_id),
 CHECK((counterparty_type='customer' AND customer_id IS NOT NULL AND supplier_id IS NULL AND contractor_id IS NULL) OR (counterparty_type='supplier' AND supplier_id IS NOT NULL AND customer_id IS NULL AND contractor_id IS NULL) OR (counterparty_type='contractor' AND contractor_id IS NOT NULL AND supplier_id IS NULL AND customer_id IS NULL)),
 CHECK(total_minor >= subtotal_minor AND total_minor-subtotal_minor=tax_minor), CHECK(installment_id IS NULL OR booking_id IS NOT NULL), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX invoice_due ON invoices(project_id,type,status,due_date,id);
CREATE INDEX supplier_invoice_reference ON invoices(supplier_id,supplier_reference);
CREATE TABLE invoice_lines (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, invoice_id TEXT NOT NULL, description TEXT NOT NULL, quantity INTEGER NOT NULL CHECK(quantity > 0), unit_price INTEGER NOT NULL CHECK(unit_price >= 0), amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0), category_id TEXT REFERENCES cost_categories(id), source_cost_id TEXT, purchase_order_line_id TEXT, work_order_milestone_id TEXT,
 FOREIGN KEY(invoice_id,project_id) REFERENCES invoices(id,project_id), FOREIGN KEY(source_cost_id,project_id) REFERENCES cost_sources(id,project_id), FOREIGN KEY(purchase_order_line_id,project_id) REFERENCES purchase_order_lines(id,project_id), FOREIGN KEY(work_order_milestone_id,project_id) REFERENCES work_order_milestones(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE invoice_sources (
 invoice_line_id TEXT NOT NULL, project_id TEXT NOT NULL, receipt_line_id TEXT NOT NULL,
 FOREIGN KEY(invoice_line_id,project_id) REFERENCES invoice_lines(id,project_id), FOREIGN KEY(receipt_line_id,project_id) REFERENCES goods_receipt_lines(id,project_id), PRIMARY KEY(invoice_line_id,receipt_line_id)
) STRICT;
CREATE TABLE duplicate_reviews (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, invoice_id TEXT NOT NULL, invoice_version INTEGER NOT NULL CHECK(invoice_version > 0), reason TEXT NOT NULL CHECK(length(trim(reason)) > 0), reviewer_id TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL,
 FOREIGN KEY(invoice_id,project_id) REFERENCES invoices(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE duplicate_review_invoices (review_id TEXT NOT NULL REFERENCES duplicate_reviews(id), related_invoice_id TEXT NOT NULL REFERENCES invoices(id), PRIMARY KEY(review_id,related_invoice_id)) STRICT;
CREATE TABLE payments (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), direction TEXT NOT NULL CHECK(direction IN ('incoming','outgoing')),
 counterparty_type TEXT NOT NULL CHECK(counterparty_type IN ('customer','supplier','contractor')), customer_id TEXT, supplier_id TEXT REFERENCES suppliers(id), contractor_id TEXT REFERENCES contractors(id), booking_id TEXT,
 amount_minor INTEGER NOT NULL CHECK(amount_minor > 0), method TEXT NOT NULL CHECK(method IN ('cash','cheque','bank-transfer','other')), reference TEXT NOT NULL, date TEXT NOT NULL, notes TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','posted','reversed')), posted_at TEXT, refund_proposal_id TEXT, reversal_id TEXT REFERENCES financial_reversals(id),
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(customer_id,project_id) REFERENCES customers(id,project_id), FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), FOREIGN KEY(refund_proposal_id,project_id) REFERENCES refund_proposals(id,project_id),
 CHECK((counterparty_type='customer' AND customer_id IS NOT NULL AND supplier_id IS NULL AND contractor_id IS NULL) OR (counterparty_type='supplier' AND supplier_id IS NOT NULL AND customer_id IS NULL AND contractor_id IS NULL) OR (counterparty_type='contractor' AND contractor_id IS NOT NULL AND supplier_id IS NULL AND customer_id IS NULL)), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX payments_date ON payments(project_id,direction,date,id);
CREATE TABLE allocations (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, payment_id TEXT NOT NULL, invoice_id TEXT NOT NULL, amount_minor INTEGER NOT NULL CHECK(amount_minor > 0), created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL,
 FOREIGN KEY(payment_id,project_id) REFERENCES payments(id,project_id), FOREIGN KEY(invoice_id,project_id) REFERENCES invoices(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE INDEX allocation_payment ON allocations(payment_id,id);
CREATE INDEX allocation_invoice ON allocations(invoice_id,id);
CREATE TABLE financial_reversals (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), invoice_id TEXT UNIQUE, payment_id TEXT UNIQUE, allocation_id TEXT UNIQUE,
 reason TEXT NOT NULL CHECK(length(trim(reason)) > 0), created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL,
 FOREIGN KEY(invoice_id,project_id) REFERENCES invoices(id,project_id), FOREIGN KEY(payment_id,project_id) REFERENCES payments(id,project_id), FOREIGN KEY(allocation_id,project_id) REFERENCES allocations(id,project_id),
 CHECK((invoice_id IS NOT NULL)+(payment_id IS NOT NULL)+(allocation_id IS NOT NULL)=1)
) STRICT;
CREATE TABLE refund_proposals (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL, cancellation_id TEXT NOT NULL, amount_minor INTEGER NOT NULL CHECK(amount_minor > 0), remaining_refundable_minor INTEGER NOT NULL CHECK(remaining_refundable_minor >= 0), collections_minor INTEGER NOT NULL CHECK(collections_minor >= 0), deduction_minor INTEGER NOT NULL CHECK(deduction_minor >= 0), paid_refunds_minor INTEGER NOT NULL CHECK(paid_refunds_minor >= 0), reason TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'submitted' CHECK(status IN ('submitted','approved','rejected')), decided_by TEXT REFERENCES users(id), decided_at TEXT, decision_reason TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), FOREIGN KEY(cancellation_id,project_id) REFERENCES cancellations(id,project_id), CHECK(amount_minor <= remaining_refundable_minor), CHECK(decided_by IS NULL OR decided_by <> created_by), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE payment_refund_sources (
 refund_payment_id TEXT NOT NULL, original_payment_id TEXT NOT NULL, project_id TEXT NOT NULL, amount_minor INTEGER NOT NULL CHECK(amount_minor > 0),
 FOREIGN KEY(refund_payment_id,project_id) REFERENCES payments(id,project_id), FOREIGN KEY(original_payment_id,project_id) REFERENCES payments(id,project_id), CHECK(refund_payment_id <> original_payment_id), PRIMARY KEY(refund_payment_id,original_payment_id)
) STRICT;
CREATE TABLE sale_milestones (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL UNIQUE, agreement_document_id TEXT NOT NULL, milestone_date TEXT NOT NULL, required_payment_minor INTEGER NOT NULL CHECK(required_payment_minor >= 0), notes TEXT NOT NULL,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), FOREIGN KEY(agreement_document_id,project_id) REFERENCES documents(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE payment_verifications (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL, purpose TEXT NOT NULL CHECK(purpose IN ('sale','handover')), booking_version INTEGER NOT NULL CHECK(booking_version > 0), required_minor INTEGER NOT NULL CHECK(required_minor >= 0), verified_minor INTEGER NOT NULL CHECK(verified_minor >= 0), verifier_id TEXT NOT NULL REFERENCES users(id), verified_at TEXT NOT NULL, notes TEXT NOT NULL,
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE payment_verification_allocations (
 verification_id TEXT NOT NULL, project_id TEXT NOT NULL, allocation_id TEXT NOT NULL, payment_version INTEGER NOT NULL CHECK(payment_version > 0), invoice_version INTEGER NOT NULL CHECK(invoice_version > 0), amount_minor INTEGER NOT NULL CHECK(amount_minor > 0),
 FOREIGN KEY(verification_id,project_id) REFERENCES payment_verifications(id,project_id), FOREIGN KEY(allocation_id,project_id) REFERENCES allocations(id,project_id), PRIMARY KEY(verification_id,allocation_id)
) STRICT;
CREATE TABLE handover_items (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL, label TEXT NOT NULL, required INTEGER NOT NULL DEFAULT 1 CHECK(required IN (0,1)), completed INTEGER NOT NULL DEFAULT 0 CHECK(completed IN (0,1)), completed_by TEXT REFERENCES users(id), completed_at TEXT,
 created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0),
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), CHECK(completed=0 OR (completed_by IS NOT NULL AND completed_at IS NOT NULL)), UNIQUE(id,project_id)
) STRICT;
CREATE TABLE handovers (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, booking_id TEXT NOT NULL UNIQUE, payment_verification_id TEXT NOT NULL, handover_date TEXT NOT NULL, notes TEXT NOT NULL, created_by TEXT NOT NULL REFERENCES users(id), created_at TEXT NOT NULL,
 FOREIGN KEY(booking_id,project_id) REFERENCES bookings(id,project_id), FOREIGN KEY(payment_verification_id,project_id) REFERENCES payment_verifications(id,project_id), UNIQUE(id,project_id)
) STRICT;
