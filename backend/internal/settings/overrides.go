package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/money"
	"buildestate/backend/internal/platform/transactions"
)

type StockMinimum struct {
	SiteID          *string `json:"siteId"`
	MaterialID      string  `json:"materialId"`
	MinimumQuantity string  `json:"minimumQuantity"`
}
type ProjectOverrides struct {
	ProjectID              string           `json:"projectId"`
	Version                int64            `json:"version"`
	ReservationHours       *int             `json:"reservationHours"`
	StockMinimums          []StockMinimum   `json:"stockMinimums"`
	InspectionChecklistIDs []string         `json:"inspectionChecklistIds"`
	ReferenceValues        []ReferenceValue `json:"referenceValues"`
}

func decodeOwnedSettings(w http.ResponseWriter, r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return invalid()
	}
	if d.Decode(new(any)) != io.EOF {
		return invalid()
	}
	return nil
}
func (s Service) registerOverrides(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects/{id}/overrides", s.handle("projects.manage", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		tx, err := s.D.DB.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		defer tx.Rollback()
		err = (policy.Service{Clock: s.D.Clock}).AuthorizeTx(r.Context(), tx, "projects.manage", id)
		var out ProjectOverrides
		if err == nil {
			out, err = readProjectOverrides(r.Context(), tx, id)
		}
		respond(w, r, out, err)
	}))
	mux.HandleFunc("PUT /api/v1/projects/{id}/overrides", s.handle("projects.manage", func(w http.ResponseWriter, r *http.Request) {
		var in ProjectOverrides
		err := decodeOwnedSettings(w, r, &in)
		if err == nil && in.ProjectID != "" && in.ProjectID != r.PathValue("id") {
			err = invalid()
		}
		in.ProjectID = r.PathValue("id")
		if err == nil {
			err = s.saveProjectOverrides(r.Context(), &in)
		}
		respond(w, r, in, err)
	}))
}
func readProjectOverrides(ctx context.Context, tx contracts.Transaction, id string) (ProjectOverrides, error) {
	out := ProjectOverrides{ProjectID: id, StockMinimums: []StockMinimum{}, InspectionChecklistIDs: []string{}, ReferenceValues: []ReferenceValue{}}
	err := tx.QueryRowContext(ctx, `SELECT version,reservation_hours FROM project_overrides WHERE project_id=?`, id).Scan(&out.Version, &out.ReservationHours)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT site_id,material_id,minimum_quantity FROM material_site_minimums WHERE project_id=? ORDER BY coalesce(site_id,''),material_id`, id)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var m StockMinimum
		var q int64
		if err = rows.Scan(&m.SiteID, &m.MaterialID, &q); err != nil {
			rows.Close()
			return out, err
		}
		m.MinimumQuantity = formatOverrideQuantity(q)
		out.StockMinimums = append(out.StockMinimums, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT checklist_id FROM project_checklists WHERE project_id=? ORDER BY checklist_id`, id)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v string
		if err = rows.Scan(&v); err != nil {
			rows.Close()
			return out, err
		}
		out.InspectionChecklistIDs = append(out.InspectionChecklistIDs, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT key,value,label FROM reference_values WHERE project_id=? AND status='active' ORDER BY key,value`, id)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v ReferenceValue
		if err = rows.Scan(&v.Key, &v.Value, &v.Label); err != nil {
			rows.Close()
			return out, err
		}
		out.ReferenceValues = append(out.ReferenceValues, v)
	}
	err = rows.Err()
	rows.Close()
	return out, err
}
func minimumKey(m StockMinimum) string {
	site := ""
	if m.SiteID != nil {
		site = *m.SiteID
	}
	return site + "\x00" + m.MaterialID
}
func formatOverrideQuantity(q int64) string {
	whole := strconv.FormatInt(q/1000, 10)
	if q%1000 == 0 {
		return whole
	}
	fraction := strconv.FormatInt(1000+q%1000, 10)[1:]
	return whole + "." + strings.TrimRight(fraction, "0")
}
func permittedOverrideReference(key string) bool {
	return strings.HasPrefix(key, "operational.") && len(key) > len("operational.") || key == "property-type" || key == "unit" || key == "cost-category" || key == "weather"
}
func (s Service) saveProjectOverrides(ctx context.Context, in *ProjectOverrides) error {
	if in.ProjectID == "" || in.Version < 0 || (in.ReservationHours != nil && (*in.ReservationHours < 1 || *in.ReservationHours > 8760)) || len(in.StockMinimums) > 1000 || len(in.ReferenceValues) > 1000 || len(in.InspectionChecklistIDs) > 1000 {
		return invalid()
	}
	if in.StockMinimums == nil {
		in.StockMinimums = []StockMinimum{}
	}
	seen := map[string]bool{}
	for i, m := range in.StockMinimums {
		q, e := money.ParseQuantity(m.MinimumQuantity)
		k := minimumKey(m)
		if e != nil || q < 0 || m.MaterialID == "" || (m.SiteID != nil && *m.SiteID == "") || seen[k] {
			return invalid()
		}
		seen[k] = true
		in.StockMinimums[i].MinimumQuantity = formatOverrideQuantity(q)
	}
	sort.Slice(in.StockMinimums, func(i, j int) bool { return minimumKey(in.StockMinimums[i]) < minimumKey(in.StockMinimums[j]) })
	seen = map[string]bool{}
	for _, v := range in.ReferenceValues {
		if !permittedOverrideReference(v.Key) {
			return invalid()
		}
		k := v.Key + "\x00" + v.Value
		if strings.TrimSpace(v.Key) == "" || strings.TrimSpace(v.Value) == "" || len(v.Key) > 100 || len(v.Value) > 1000 || len(v.Label) > 1000 || seen[k] {
			return invalid()
		}
		seen[k] = true
	}
	seen = map[string]bool{}
	for _, v := range in.InspectionChecklistIDs {
		if v == "" || seen[v] {
			return invalid()
		}
		seen[v] = true
	}
	err := s.write(ctx, "projects.manage", in.ProjectID, "project.update", "project", in.ProjectID, func(ctx context.Context, tx contracts.Transaction) error {
		old, err := readProjectOverrides(ctx, tx, in.ProjectID)
		if err != nil {
			return err
		}
		if old.Version != in.Version {
			return &transactions.Error{Status: 409, Code: "STALE_VERSION", Message: "Reload the current record before saving."}
		}
		stockChanged := !reflect.DeepEqual(old.StockMinimums, in.StockMinimums)
		if stockChanged {
			if err = (policy.Service{Clock: s.D.Clock}).AuthorizeTx(ctx, tx, "inventory.minimums", in.ProjectID); err != nil {
				return err
			}
		}
		actor := policy.PrincipalFrom(ctx).UserID
		stamp := s.now().UTC().Format(time.RFC3339Nano)
		for _, m := range in.StockMinimums {
			var ok int
			err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM materials WHERE id=? AND status='active') AND (? IS NULL OR EXISTS(SELECT 1 FROM sites WHERE id=? AND project_id=? AND status='active'))`, m.MaterialID, m.SiteID, m.SiteID, in.ProjectID).Scan(&ok)
			if err != nil {
				return err
			}
			if ok != 1 {
				return invalid()
			}
		}
		for _, id := range in.InspectionChecklistIDs {
			var ok int
			err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM inspection_checklists WHERE id=? AND status='active')`, id).Scan(&ok)
			if err != nil {
				return err
			}
			if ok != 1 {
				return invalid()
			}
		}
		if old.Version == 0 {
			_, err = tx.ExecContext(ctx, `INSERT INTO project_overrides(id,project_id,reservation_hours,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?)`, transactions.NewID(), in.ProjectID, in.ReservationHours, actor, stamp, stamp)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE project_overrides SET reservation_hours=?,version=version+1,updated_at=? WHERE project_id=?`, in.ReservationHours, stamp, in.ProjectID)
		}
		if err != nil {
			return err
		}
		if stockChanged {
			if _, err = tx.ExecContext(ctx, `DELETE FROM material_site_minimums WHERE project_id=?`, in.ProjectID); err != nil {
				return err
			}
			for _, m := range in.StockMinimums {
				q, _ := money.ParseQuantity(m.MinimumQuantity)
				if _, err = tx.ExecContext(ctx, `INSERT INTO material_site_minimums(id,project_id,site_id,material_id,minimum_quantity,created_by) VALUES(?,?,?,?,?,?)`, transactions.NewID(), in.ProjectID, m.SiteID, m.MaterialID, q, actor); err != nil {
					return err
				}
			}
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM project_checklists WHERE project_id=?`, in.ProjectID); err != nil {
			return err
		}
		for _, id := range in.InspectionChecklistIDs {
			if _, err = tx.ExecContext(ctx, `INSERT INTO project_checklists(project_id,checklist_id) VALUES(?,?)`, in.ProjectID, id); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE reference_values SET status='archived',version=version+1,updated_at=? WHERE project_id=? AND status='active'`, stamp, in.ProjectID); err != nil {
			return err
		}
		for _, v := range in.ReferenceValues {
			if _, err = tx.ExecContext(ctx, `INSERT INTO reference_values(id,project_id,key,value,label,status,created_by) VALUES(?,?,?,?,?,'active',?) ON CONFLICT(project_id,key,value) DO UPDATE SET label=excluded.label,status='active',version=reference_values.version+1,updated_at=?`, transactions.NewID(), in.ProjectID, v.Key, v.Value, v.Label, actor, stamp); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		if replay, ok := mutationReplay(ctx).(ProjectOverrides); ok {
			*in = replay
		} else {
			in.Version++
		}
	}
	return err
}
