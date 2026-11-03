-- Cross-row guards supplement, not replace, authorized BEGIN IMMEDIATE workflows.
CREATE TRIGGER claims_insert_guard BEFORE INSERT ON unit_claims BEGIN
 SELECT CASE WHEN (SELECT availability FROM properties WHERE id=NEW.property_id)='unavailable' THEN RAISE(ABORT,'unit is unavailable') END;
 SELECT CASE WHEN NEW.reservation_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM reservations WHERE id=NEW.reservation_id AND status IN ('active','pending-review')) THEN RAISE(ABORT,'reservation cannot own unit') END;
 SELECT CASE WHEN NEW.booking_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM bookings WHERE id=NEW.booking_id AND status IN ('active','cancellation-requested')) THEN RAISE(ABORT,'booking cannot own unit') END;
END;
CREATE TRIGGER claims_update_guard BEFORE UPDATE ON unit_claims BEGIN
 SELECT CASE WHEN NEW.property_id<>OLD.property_id OR NEW.project_id<>OLD.project_id THEN RAISE(ABORT,'claim identity is immutable') END;
 SELECT CASE WHEN OLD.booking_id IS NOT NULL AND NEW.booking_id IS NOT OLD.booking_id THEN RAISE(ABORT,'booking claim requires approved release') END;
 SELECT CASE WHEN NEW.booking_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM bookings b JOIN reservations r ON r.id=OLD.reservation_id WHERE b.id=NEW.booking_id AND b.reservation_id=r.id AND b.customer_id=r.customer_id AND b.status='active') AND OLD.booking_id IS NULL THEN RAISE(ABORT,'invalid reservation conversion') END;
 SELECT CASE WHEN NEW.reservation_id IS NOT NULL AND NEW.reservation_id IS NOT OLD.reservation_id THEN RAISE(ABORT,'reservation ownership cannot be replaced') END;
END;
CREATE TRIGGER claims_release_guard BEFORE DELETE ON unit_claims BEGIN
 SELECT CASE WHEN OLD.booking_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM bookings b JOIN cancellations c ON c.booking_id=b.id WHERE b.id=OLD.booking_id AND b.status='cancelled' AND c.status='approved') THEN RAISE(ABORT,'booking release requires approved cancellation') END;
 SELECT CASE WHEN OLD.reservation_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM reservations WHERE id=OLD.reservation_id AND status IN ('expired','cancelled')) THEN RAISE(ABORT,'active reservation cannot be released') END;
END;
CREATE TRIGGER claimed_property_manual_status BEFORE UPDATE OF availability ON properties
WHEN NEW.availability IN ('available','unavailable') AND EXISTS(SELECT 1 FROM unit_claims WHERE property_id=OLD.id)
BEGIN SELECT RAISE(ABORT,'claimed unit cannot be manually released'); END;
CREATE TRIGGER bands_no_overlap_insert BEFORE INSERT ON cancellation_bands
WHEN EXISTS(SELECT 1 FROM cancellation_bands b WHERE b.terms_id=NEW.terms_id AND (NEW.to_date IS NULL OR b.from_date<NEW.to_date) AND (b.to_date IS NULL OR NEW.from_date<b.to_date))
BEGIN SELECT RAISE(ABORT,'cancellation effective bands overlap'); END;
CREATE TRIGGER bands_no_overlap_update BEFORE UPDATE ON cancellation_bands
WHEN EXISTS(SELECT 1 FROM cancellation_bands b WHERE b.id<>OLD.id AND b.terms_id=NEW.terms_id AND (NEW.to_date IS NULL OR b.from_date<NEW.to_date) AND (b.to_date IS NULL OR NEW.from_date<b.to_date))
BEGIN SELECT RAISE(ABORT,'cancellation effective bands overlap'); END;
CREATE TRIGGER cancellation_terms_approved_immutable BEFORE UPDATE ON cancellation_terms
WHEN OLD.status IN ('approved','superseded') AND (NEW.booking_id IS NOT OLD.booking_id OR NEW.project_id IS NOT OLD.project_id OR NEW.revision IS NOT OLD.revision OR NEW.reason IS NOT OLD.reason OR NEW.proposed_by IS NOT OLD.proposed_by OR NEW.decided_by IS NOT OLD.decided_by OR NEW.decided_at IS NOT OLD.decided_at OR NEW.status NOT IN ('approved','superseded'))
BEGIN SELECT RAISE(ABORT,'approved cancellation terms are immutable'); END;
CREATE TRIGGER cancellation_terms_retained BEFORE DELETE ON cancellation_terms BEGIN SELECT RAISE(ABORT,'terms history is retained'); END;
CREATE TRIGGER material_unit_referenced BEFORE UPDATE OF unit ON materials
WHEN NEW.unit<>OLD.unit AND (EXISTS(SELECT 1 FROM purchase_order_lines WHERE material_id=OLD.id) OR EXISTS(SELECT 1 FROM stock_entries WHERE material_id=OLD.id))
BEGIN SELECT RAISE(ABORT,'referenced material unit is immutable'); END;
CREATE TRIGGER inspection_completion_guard BEFORE UPDATE OF status ON inspections
WHEN NEW.status='completed' AND (NOT EXISTS(SELECT 1 FROM inspection_items WHERE inspection_id=OLD.id) OR EXISTS(SELECT 1 FROM inspection_items WHERE inspection_id=OLD.id AND result='fail' AND issue_id IS NULL))
BEGIN SELECT RAISE(ABORT,'inspection requires complete corrective links'); END;
CREATE TRIGGER completed_inspection_items_update BEFORE UPDATE ON inspection_items
WHEN EXISTS(SELECT 1 FROM inspections WHERE id=OLD.inspection_id AND status='completed')
BEGIN SELECT RAISE(ABORT,'completed checklist is immutable'); END;
CREATE TRIGGER completed_inspection_items_delete BEFORE DELETE ON inspection_items
WHEN EXISTS(SELECT 1 FROM inspections WHERE id=OLD.inspection_id AND status='completed')
BEGIN SELECT RAISE(ABORT,'completed checklist is retained'); END;
CREATE TRIGGER completed_inspection_items_insert BEFORE INSERT ON inspection_items
WHEN EXISTS(SELECT 1 FROM inspections WHERE id=NEW.inspection_id AND status='completed')
BEGIN SELECT RAISE(ABORT,'completed checklist is immutable'); END;
CREATE TRIGGER invoice_posted_immutable BEFORE UPDATE ON invoices
WHEN OLD.status IN ('posted','reversed') AND (NEW.project_id IS NOT OLD.project_id OR NEW.number IS NOT OLD.number OR NEW.type IS NOT OLD.type OR NEW.counterparty_type IS NOT OLD.counterparty_type OR NEW.customer_id IS NOT OLD.customer_id OR NEW.supplier_id IS NOT OLD.supplier_id OR NEW.contractor_id IS NOT OLD.contractor_id OR NEW.booking_id IS NOT OLD.booking_id OR NEW.installment_id IS NOT OLD.installment_id OR NEW.subtotal_minor IS NOT OLD.subtotal_minor OR NEW.tax_minor IS NOT OLD.tax_minor OR NEW.total_minor IS NOT OLD.total_minor OR NEW.due_date IS NOT OLD.due_date OR NEW.status='draft' OR (OLD.status='reversed' AND NEW.status<>'reversed'))
BEGIN SELECT RAISE(ABORT,'posted invoice is immutable'); END;
CREATE TRIGGER invoice_retained BEFORE DELETE ON invoices BEGIN SELECT RAISE(ABORT,'invoice history is retained'); END;
CREATE TRIGGER payment_posted_immutable BEFORE UPDATE ON payments
WHEN OLD.status IN ('posted','reversed') AND (NEW.project_id IS NOT OLD.project_id OR NEW.direction IS NOT OLD.direction OR NEW.counterparty_type IS NOT OLD.counterparty_type OR NEW.customer_id IS NOT OLD.customer_id OR NEW.supplier_id IS NOT OLD.supplier_id OR NEW.contractor_id IS NOT OLD.contractor_id OR NEW.booking_id IS NOT OLD.booking_id OR NEW.amount_minor IS NOT OLD.amount_minor OR NEW.method IS NOT OLD.method OR NEW.reference IS NOT OLD.reference OR NEW.date IS NOT OLD.date OR NEW.refund_proposal_id IS NOT OLD.refund_proposal_id OR NEW.status='draft' OR (OLD.status='reversed' AND NEW.status<>'reversed'))
BEGIN SELECT RAISE(ABORT,'posted payment is immutable'); END;
CREATE TRIGGER payment_retained BEFORE DELETE ON payments BEGIN SELECT RAISE(ABORT,'payment history is retained'); END;
CREATE TRIGGER allocation_compatibility BEFORE INSERT ON allocations
WHEN NOT EXISTS(SELECT 1 FROM invoices i JOIN payments p ON p.id=NEW.payment_id WHERE i.id=NEW.invoice_id AND i.status='posted' AND p.status='posted' AND i.project_id=p.project_id AND i.counterparty_type=p.counterparty_type AND i.customer_id IS p.customer_id AND i.supplier_id IS p.supplier_id AND i.contractor_id IS p.contractor_id AND ((i.type='receivable' AND p.direction='incoming') OR (i.type='payable' AND p.direction='outgoing')))
BEGIN SELECT RAISE(ABORT,'allocation requires compatible posted records'); END;
CREATE TRIGGER allocations_within_balances BEFORE INSERT ON allocations BEGIN
 SELECT CASE WHEN NEW.amount_minor > (SELECT amount_minor FROM payments WHERE id=NEW.payment_id) - COALESCE((SELECT sum(a.amount_minor) FROM allocations a WHERE a.payment_id=NEW.payment_id AND NOT EXISTS(SELECT 1 FROM financial_reversals r WHERE r.allocation_id=a.id)),0) - COALESCE((SELECT sum(s.amount_minor) FROM payment_refund_sources s JOIN payments p ON p.id=s.refund_payment_id WHERE s.original_payment_id=NEW.payment_id AND p.status='posted'),0) THEN RAISE(ABORT,'payment allocation exceeds available cash') END;
 SELECT CASE WHEN NEW.amount_minor > (SELECT total_minor FROM invoices WHERE id=NEW.invoice_id) - COALESCE((SELECT sum(a.amount_minor) FROM allocations a WHERE a.invoice_id=NEW.invoice_id AND NOT EXISTS(SELECT 1 FROM financial_reversals r WHERE r.allocation_id=a.id)),0) THEN RAISE(ABORT,'allocation exceeds invoice outstanding') END;
END;
CREATE TRIGGER currency_locked_guard BEFORE UPDATE ON company_settings
WHEN OLD.currency_locked=1 AND (NEW.currency_locked<>1 OR NEW.currency_code IS NOT OLD.currency_code OR NEW.currency_minor_digits IS NOT OLD.currency_minor_digits)
BEGIN SELECT RAISE(ABORT,'posted currency is locked'); END;
CREATE TRIGGER stock_nonnegative BEFORE INSERT ON stock_entries
WHEN NEW.quantity<0 AND -NEW.quantity > COALESCE((SELECT sum(quantity) FROM stock_entries WHERE site_id=NEW.site_id AND material_id=NEW.material_id),0)
BEGIN SELECT RAISE(ABORT,'insufficient stock'); END;
CREATE TRIGGER recognition_once BEFORE INSERT ON cost_recognitions
WHEN NEW.kind='recognition' AND EXISTS(SELECT 1 FROM cost_recognitions r WHERE r.source_cost_id=NEW.source_cost_id AND r.kind='recognition' AND NOT EXISTS(SELECT 1 FROM cost_recognitions v WHERE v.reverses_id=r.id))
BEGIN SELECT RAISE(ABORT,'economic source already recognized'); END;
CREATE TRIGGER recognition_reversal_matches BEFORE INSERT ON cost_recognitions
WHEN NEW.kind='reversal' AND NOT EXISTS(SELECT 1 FROM cost_recognitions r WHERE r.id=NEW.reverses_id AND r.kind='recognition' AND r.source_cost_id=NEW.source_cost_id AND r.project_id=NEW.project_id AND r.category_id=NEW.category_id AND r.amount_minor=-NEW.amount_minor)
BEGIN SELECT RAISE(ABORT,'recognition reversal must compensate original'); END;
CREATE TRIGGER property_image_mime BEFORE INSERT ON property_images
WHEN NOT EXISTS(SELECT 1 FROM documents WHERE id=NEW.document_id AND mime_type IN ('image/jpeg','image/png','image/webp') AND status='active')
BEGIN SELECT RAISE(ABORT,'property image must be an active verified image'); END;
