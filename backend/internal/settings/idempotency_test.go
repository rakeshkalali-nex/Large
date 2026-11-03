package settings

import (
	"buildestate/backend/internal/platform/transactions"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEndpointDurableReplayCreateUpdateMismatchAndRevocation(t *testing.T) {
	s, ctx := settingsFixture(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, s.D)
	call := func(method, path, key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	body := `{"subjectType":"user","subjectId":"admin","kind":"access","reason":"Requested access"}`
	key := transactions.NewID()
	a := call("POST", "/api/v1/privacy-requests", key, body)
	b := call("POST", "/api/v1/privacy-requests", key, body)
	if a.Code != 200 || b.Code != 200 || a.Body.String() != b.Body.String() {
		t.Fatalf("create replay %d %s / %d %s", a.Code, a.Body, b.Code, b.Body)
	}
	var n int
	s.D.DB.QueryRow(`SELECT count(*) FROM privacy_requests`).Scan(&n)
	if n != 1 {
		t.Fatal("duplicate create")
	}
	mismatch := call("POST", "/api/v1/privacy-requests", key, strings.Replace(body, "Requested access", "Different reason", 1))
	if mismatch.Code != 409 {
		t.Fatalf("mismatch: %d %s", mismatch.Code, mismatch.Body)
	}
	v := validSettings()
	raw, _ := json.Marshal(v)
	var input map[string]any
	json.Unmarshal(raw, &input)
	delete(input, "currencyLocked")
	delete(input["operatingPolicy"].(map[string]any), "confirmedAt")
	raw, _ = json.Marshal(input)
	updateKey := transactions.NewID()
	a = call("PUT", "/api/v1/settings", updateKey, string(raw))
	b = call("PUT", "/api/v1/settings", updateKey, string(raw))
	if a.Code != 200 || b.Code != 200 || a.Body.String() != b.Body.String() {
		t.Fatalf("update replay %d %s / %d %s", a.Code, a.Body, b.Code, b.Body)
	}
	s.D.DB.QueryRow(`SELECT count(*) FROM settings_history`).Scan(&n)
	if n != 1 {
		t.Fatal("duplicate history")
	}
	if _, e := s.D.DB.Exec(`UPDATE users SET access_revision=access_revision+1 WHERE id='admin'`); e != nil {
		t.Fatal(e)
	}
	if w := call("POST", "/api/v1/privacy-requests", key, body); w.Code != 401 {
		t.Fatalf("revoked replay: %d", w.Code)
	}
}
func TestEndpointIdempotencyAuditRollback(t *testing.T) {
	s, ctx := settingsFixture(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, s.D)
	if _, e := s.D.DB.Exec(`CREATE TRIGGER reject_audit BEFORE INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'fault'); END`); e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("POST", "/api/v1/privacy-requests", strings.NewReader(`{"subjectType":"user","subjectId":"admin","kind":"access","reason":"Requested access"}`)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", transactions.NewID())
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 500 {
		t.Fatal(w.Code)
	}
	for _, table := range []string{"idempotency_records", "privacy_requests"} {
		var n int
		s.D.DB.QueryRow("SELECT count(*) FROM " + table).Scan(&n)
		if n != 0 {
			t.Fatal("partial commit", table)
		}
	}
}
