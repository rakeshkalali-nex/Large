package settings

import (
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/posting"
	"buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/platform/transport"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
)

func (s Service) registerSettings(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings", s.handle("settings.read", func(w http.ResponseWriter, r *http.Request) {
		v, e := readSettings(r.Context(), s.D.DB)
		respond(w, r, v, e)
	}))
	mux.HandleFunc("PUT /api/v1/settings", s.handle("settings.manage", func(w http.ResponseWriter, r *http.Request) {
		var v Settings
		var raw json.RawMessage
		if !transport.DecodeJSON(w, r, &raw) {
			return
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			respond(w, r, nil, invalid())
			return
		}
		if _, ok := fields["currencyLocked"]; ok {
			respond(w, r, nil, invalid())
			return
		}
		var policyFields map[string]json.RawMessage
		if json.Unmarshal(fields["operatingPolicy"], &policyFields) != nil {
			respond(w, r, nil, invalid())
			return
		}
		if _, ok := policyFields["confirmedAt"]; ok {
			respond(w, r, nil, invalid())
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&v) != nil {
			respond(w, r, nil, invalid())
			return
		}
		out, e := s.saveSettings(r.Context(), v)
		respond(w, r, out, e)
	}))
	mux.HandleFunc("GET /api/v1/settings/history", s.handle("settings.read", func(w http.ResponseWriter, r *http.Request) {
		page, e := stewardPaging(r.URL.RawQuery)
		if e != nil {
			respond(w, r, nil, e)
			return
		}
		rows, e := s.D.DB.QueryContext(r.Context(), `SELECT version,snapshot_json,actor_id,occurred_at FROM settings_history ORDER BY version DESC LIMIT ? OFFSET ?`, page.Limit, page.Offset)
		if e != nil {
			respond(w, r, nil, e)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var version int64
			var raw, actor, at string
			if e = rows.Scan(&version, &raw, &actor, &at); e != nil {
				break
			}
			var snapshot Settings
			if e = json.Unmarshal([]byte(raw), &snapshot); e != nil {
				break
			}
			out = append(out, map[string]any{"id": fmt.Sprint(version), "action": "settings.update", "version": version, "settings": snapshot, "actorId": actor, "occurredAt": at})
		}
		if e == nil {
			e = rows.Err()
		}
		respond(w, r, out, e)
	}))
}
func readSettings(ctx context.Context, q contracts.Transaction) (Settings, error) {
	v := Settings{Timezone: "UTC", ReservationHours: 48, ReferenceValues: []ReferenceValue{}, PropertyTypes: []string{}, InspectionChecklists: []Checklist{}, OperatingPolicy: OperatingPolicy{Countries: []string{}}}
	e := q.QueryRowContext(ctx, `SELECT version,company_name,wordmark,logo_document_id,currency_code,currency_minor_digits,currency_locked,timezone,timezone_confirmed,recovery_instructions,recovery_contact_label,reservation_hours,notice_version,retention_schedule_version,operational_policy_confirmed_at FROM company_settings WHERE id='company'`).Scan(&v.Version, &v.CompanyName, &v.Branding.Wordmark, &v.Branding.LogoDocumentID, &v.CurrencyCode, &v.CurrencyMinorDigits, &v.CurrencyLocked, &v.Timezone, &v.TimezoneConfirmed, &v.RecoveryInstructions, &v.RecoveryContactLabel, &v.ReservationHours, &v.OperatingPolicy.NoticeVersion, &v.OperatingPolicy.RetentionScheduleVersion, &v.OperatingPolicy.ConfirmedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return v, nil
	}
	if e != nil {
		return v, e
	}
	rows, e := q.QueryContext(ctx, `SELECT country_code FROM operating_countries WHERE settings_id='company' ORDER BY country_code`)
	if e != nil {
		return v, e
	}
	for rows.Next() {
		var c string
		if e = rows.Scan(&c); e != nil {
			break
		}
		v.OperatingPolicy.Countries = append(v.OperatingPolicy.Countries, c)
	}
	rows.Close()
	if e != nil {
		return v, e
	}
	rows, e = q.QueryContext(ctx, `SELECT key,value,label FROM reference_values WHERE project_id IS NULL AND status='active' ORDER BY key,value`)
	if e != nil {
		return v, e
	}
	for rows.Next() {
		var x ReferenceValue
		if e = rows.Scan(&x.Key, &x.Value, &x.Label); e != nil {
			break
		}
		v.ReferenceValues = append(v.ReferenceValues, x)
	}
	rows.Close()
	if e != nil {
		return v, e
	}
	rows, e = q.QueryContext(ctx, `SELECT name FROM property_types WHERE status='active' ORDER BY name`)
	if e != nil {
		return v, e
	}
	for rows.Next() {
		var x string
		if e = rows.Scan(&x); e != nil {
			break
		}
		v.PropertyTypes = append(v.PropertyTypes, x)
	}
	rows.Close()
	if e != nil {
		return v, e
	}
	rows, e = q.QueryContext(ctx, `SELECT id,name FROM inspection_checklists WHERE status='active' ORDER BY name,id`)
	if e != nil {
		return v, e
	}
	for rows.Next() {
		x := Checklist{Items: []string{}}
		if e = rows.Scan(&x.ID, &x.Name); e != nil {
			break
		}
		v.InspectionChecklists = append(v.InspectionChecklists, x)
	}
	rows.Close()
	if e != nil {
		return v, e
	}
	for i := range v.InspectionChecklists {
		rows, e = q.QueryContext(ctx, `SELECT label FROM inspection_checklist_items WHERE checklist_id=? ORDER BY sort_order`, v.InspectionChecklists[i].ID)
		if e != nil {
			return v, e
		}
		for rows.Next() {
			var x string
			if e = rows.Scan(&x); e != nil {
				break
			}
			v.InspectionChecklists[i].Items = append(v.InspectionChecklists[i].Items, x)
		}
		rows.Close()
		if e != nil {
			return v, e
		}
	}
	return v, nil
}
func text(v string, max int) bool { return strings.TrimSpace(v) != "" && len(v) <= max }
func validateSettings(v Settings) error {
	if !text(v.CompanyName, 200) || !text(v.Branding.Wordmark, 100) || !text(v.RecoveryInstructions, 2000) || !text(v.RecoveryContactLabel, 200) || v.ReservationHours < 1 || v.ReservationHours > 8760 || v.Timezone == "Local" || v.Timezone == "" {
		return invalid()
	}
	if _, e := time.LoadLocation(v.Timezone); e != nil {
		return invalid()
	}
	if (v.CurrencyCode == nil) != (v.CurrencyMinorDigits == nil) {
		return invalid()
	}
	if v.CurrencyCode != nil && (!posting.ValidCurrency(*v.CurrencyCode) || *v.CurrencyMinorDigits < 0 || *v.CurrencyMinorDigits > 4) {
		return invalid()
	}
	if len(v.ReferenceValues) > 200 || len(v.PropertyTypes) > 100 || len(v.InspectionChecklists) > 100 || len(v.OperatingPolicy.Countries) > 100 {
		return invalid()
	}
	seen := map[string]bool{}
	for _, c := range v.OperatingPolicy.Countries {
		if len(c) != 2 || c[0] < 'A' || c[0] > 'Z' || c[1] < 'A' || c[1] > 'Z' || seen[c] {
			return invalid()
		}
		seen[c] = true
	}
	seen = map[string]bool{}
	consentRule := false
	for _, x := range v.ReferenceValues {
		if x.Key == "privacy.consent-required" {
			if consentRule || (x.Value != "true" && x.Value != "false") {
				return invalid()
			}
			consentRule = true
		}
		key := x.Key + "\x00" + x.Value
		if !text(x.Key, 100) || !text(x.Value, 200) || !text(x.Label, 200) || seen[key] {
			return invalid()
		}
		seen[key] = true
	}
	seen = map[string]bool{}
	for _, x := range v.PropertyTypes {
		if !text(x, 100) || seen[x] {
			return invalid()
		}
		seen[x] = true
	}
	seen = map[string]bool{}
	for _, x := range v.InspectionChecklists {
		if !text(x.Name, 200) || len(x.Items) == 0 || len(x.Items) > 100 || x.ID != "" && seen[x.ID] {
			return invalid()
		}
		seen[x.ID] = true
		for _, item := range x.Items {
			if !text(item, 500) {
				return invalid()
			}
		}
	}
	if v.ConfirmOperationalPolicy && (len(v.OperatingPolicy.Countries) == 0 || v.OperatingPolicy.NoticeVersion == nil || !text(*v.OperatingPolicy.NoticeVersion, 200) || v.OperatingPolicy.RetentionScheduleVersion == nil || !text(*v.OperatingPolicy.RetentionScheduleVersion, 200)) {
		return invalid()
	}
	return nil
}
func (s Service) saveSettings(ctx context.Context, v Settings) (Settings, error) {
	if e := validateSettings(v); e != nil {
		return Settings{}, e
	}
	var out Settings
	e := s.write(ctx, "settings.manage", "", "settings.update", "settings", "company", func(ctx context.Context, tx contracts.Transaction) error {
		old, e := readSettings(ctx, tx)
		if e != nil {
			return e
		}
		if old.Version != v.Version {
			return transactions.Conflict("STALE_VERSION")
		}
		if old.CurrencyLocked && (v.CurrencyCode == nil || v.CurrencyMinorDigits == nil || old.CurrencyCode == nil || *v.CurrencyCode != *old.CurrencyCode || *v.CurrencyMinorDigits != *old.CurrencyMinorDigits) {
			return transactions.Conflict("CURRENCY_LOCKED")
		}
		v.CurrencyLocked = old.CurrencyLocked
		v.OperatingPolicy.ConfirmedAt = nil
		if samePolicy(old, v) {
			v.OperatingPolicy.ConfirmedAt = old.OperatingPolicy.ConfirmedAt
		}
		if v.ConfirmOperationalPolicy {
			at := transactions.Timestamp(s.now())
			v.OperatingPolicy.ConfirmedAt = &at
		}
		actor, _ := s.actor(ctx, tx)
		at := transactions.Timestamp(s.now())
		v.Version++
		if v.Branding.LogoDocumentID != nil {
			var count int
			if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM documents WHERE id=? AND status='active' AND mime_type IN ('image/jpeg','image/png','image/webp') AND classification='operational'`, *v.Branding.LogoDocumentID).Scan(&count); e != nil {
				return e
			}
			if count != 1 {
				return invalid()
			}
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO company_settings(id,company_name,wordmark,logo_document_id,currency_code,currency_minor_digits,timezone,timezone_confirmed,recovery_instructions,recovery_contact_label,reservation_hours,notice_version,retention_schedule_version,operational_policy_confirmed_at,created_by,updated_at,version) VALUES('company',?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET company_name=excluded.company_name,wordmark=excluded.wordmark,logo_document_id=excluded.logo_document_id,currency_code=excluded.currency_code,currency_minor_digits=excluded.currency_minor_digits,timezone=excluded.timezone,timezone_confirmed=excluded.timezone_confirmed,recovery_instructions=excluded.recovery_instructions,recovery_contact_label=excluded.recovery_contact_label,reservation_hours=excluded.reservation_hours,notice_version=excluded.notice_version,retention_schedule_version=excluded.retention_schedule_version,operational_policy_confirmed_at=excluded.operational_policy_confirmed_at,updated_at=excluded.updated_at,version=excluded.version`, v.CompanyName, v.Branding.Wordmark, v.Branding.LogoDocumentID, v.CurrencyCode, v.CurrencyMinorDigits, v.Timezone, v.TimezoneConfirmed, v.RecoveryInstructions, v.RecoveryContactLabel, v.ReservationHours, v.OperatingPolicy.NoticeVersion, v.OperatingPolicy.RetentionScheduleVersion, v.OperatingPolicy.ConfirmedAt, actor, at, v.Version)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `DELETE FROM operating_countries WHERE settings_id='company'`); e != nil {
			return e
		}
		for _, c := range v.OperatingPolicy.Countries {
			if _, e = tx.ExecContext(ctx, `INSERT INTO operating_countries VALUES('company',?)`, c); e != nil {
				return e
			}
		}
		if e = saveReferences(ctx, tx, &v, old, actor, at); e != nil {
			return e
		}
		v.ConfirmOperationalPolicy = false
		raw, e := json.Marshal(v)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO settings_history VALUES(?,?,?,?)`, v.Version, string(raw), actor, at); e != nil {
			return e
		}
		out = v
		return nil
	})
	if e == nil {
		if replay, ok := mutationReplay(ctx).(Settings); ok {
			out = replay
		}
	}
	return out, e
}

func samePolicy(a, b Settings) bool {
	normalize := func(v Settings) string {
		v.OperatingPolicy.ConfirmedAt = nil
		v.OperatingPolicy.Countries = append([]string{}, v.OperatingPolicy.Countries...)
		sort.Strings(v.OperatingPolicy.Countries)
		consent := "true"
		for _, r := range v.ReferenceValues {
			if r.Key == "privacy.consent-required" {
				consent = r.Value
			}
		}
		raw, _ := json.Marshal(v.OperatingPolicy)
		return string(raw) + consent
	}
	return normalize(a) == normalize(b)
}
func saveReferences(ctx context.Context, tx contracts.Transaction, v *Settings, old Settings, actor, at string) error {
	if _, e := tx.ExecContext(ctx, `UPDATE reference_values SET status='archived' WHERE project_id IS NULL`); e != nil {
		return e
	}
	for _, x := range v.ReferenceValues {
		_, e := tx.ExecContext(ctx, `INSERT INTO reference_values(id,key,value,label,status,created_by) VALUES(?,?,?,?,'active',?) ON CONFLICT(key,value) WHERE project_id IS NULL DO UPDATE SET label=excluded.label,status='active',version=reference_values.version+1,updated_at=?`, transactions.NewID(), x.Key, x.Value, x.Label, actor, at)
		if e != nil {
			return e
		}
	}
	if _, e := tx.ExecContext(ctx, `UPDATE property_types SET status='archived'`); e != nil {
		return e
	}
	for _, x := range v.PropertyTypes {
		_, e := tx.ExecContext(ctx, `INSERT INTO property_types(id,code,name,status,created_by) VALUES(?,?,?,'active',?) ON CONFLICT(code) DO UPDATE SET status='active',name=excluded.name,version=property_types.version+1,updated_at=?`, transactions.NewID(), x, x, actor, at)
		if e != nil {
			return e
		}
	}
	if _, e := tx.ExecContext(ctx, `UPDATE inspection_checklists SET status='archived'`); e != nil {
		return e
	}
	for i := range v.InspectionChecklists {
		x := &v.InspectionChecklists[i]
		found := false
		for _, previous := range old.InspectionChecklists {
			if previous.ID == x.ID {
				a, _ := json.Marshal(previous)
				b, _ := json.Marshal(x)
				if string(a) != string(b) {
					return transactions.Conflict("CHECKLIST_REVISION_REQUIRED")
				}
				found = true
			}
		}
		if found {
			if _, e := tx.ExecContext(ctx, `UPDATE inspection_checklists SET status='active' WHERE id=?`, x.ID); e != nil {
				return e
			}
			continue
		}
		if x.ID != "" {
			return invalid()
		}
		x.ID = transactions.NewID()
		if _, e := tx.ExecContext(ctx, `INSERT INTO inspection_checklists(id,name,revision,status,created_by) VALUES(?,?,1,'active',?)`, x.ID, x.Name, actor); e != nil {
			return e
		}
		for n, label := range x.Items {
			if _, e := tx.ExecContext(ctx, `INSERT INTO inspection_checklist_items(id,checklist_id,label,required,sort_order) VALUES(?,?,?,1,?)`, transactions.NewID(), x.ID, label, n); e != nil {
				return e
			}
		}
	}
	return nil
}
