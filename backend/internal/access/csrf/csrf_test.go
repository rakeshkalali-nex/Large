package csrf

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

func TestIssueAndValidate(t *testing.T) {
	clock := &testClock{time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	m := Manager{Key: bytes.Repeat([]byte{1}, 32), Clock: clock}
	for _, binding := range []string{"", "session-digest"} {
		t.Run(binding, func(t *testing.T) {
			w := httptest.NewRecorder()
			token, err := m.Issue(w, httptest.NewRequest("GET", "http://example.test/api/v1/auth/csrf", nil), binding)
			if err != nil {
				t.Fatal(err)
			}
			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatal("expected one cookie")
			}
			c := cookies[0]
			if c.Name != CookieName || c.Path != "/" || c.Domain != "" || c.HttpOnly || c.Secure || c.SameSite != http.SameSiteLaxMode || c.MaxAge != 1800 {
				t.Fatalf("wrong cookie policy: %+v", c)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing no-store")
			}
			request := func() *http.Request {
				r := httptest.NewRequest("POST", "http://example.test/api/v1/auth/login", nil)
				r.Header.Set("Origin", "http://example.test")
				r.Header.Set(HeaderName, token)
				r.AddCookie(c)
				return r
			}
			if err := m.Validate(request(), binding); err != nil {
				t.Fatal(err)
			}
			if err := m.Validate(request(), binding+"wrong"); err != ErrInvalid {
				t.Fatal("binding accepted")
			}
			cases := map[string]func(*http.Request){
				"missing origin":    func(r *http.Request) { r.Header.Del("Origin") },
				"foreign origin":    func(r *http.Request) { r.Header.Set("Origin", "http://evil.test") },
				"origin path":       func(r *http.Request) { r.Header.Set("Origin", "http://example.test/") },
				"wrong port":        func(r *http.Request) { r.Header.Set("Origin", "http://example.test:80") },
				"duplicate origin":  func(r *http.Request) { r.Header.Add("Origin", "http://example.test") },
				"forwarded":         func(r *http.Request) { r.Header.Set("Forwarded", "host=example.test") },
				"forwarded host":    func(r *http.Request) { r.Header.Set("X-Forwarded-Host", "example.test") },
				"forwarded proto":   func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "http") },
				"missing cookie":    func(r *http.Request) { r.Header.Del("Cookie") },
				"duplicate cookie":  func(r *http.Request) { r.AddCookie(c) },
				"missing header":    func(r *http.Request) { r.Header.Del(HeaderName) },
				"duplicate header":  func(r *http.Request) { r.Header.Add(HeaderName, token) },
				"mismatched header": func(r *http.Request) { r.Header.Set(HeaderName, "x"+token[1:]) },
				"forged signature": func(r *http.Request) {
					raw, _ := base64.RawURLEncoding.DecodeString(token)
					raw[8] ^= 1
					forged := base64.RawURLEncoding.EncodeToString(raw)
					r.Header.Set("Cookie", CookieName+"="+forged)
					r.Header.Set(HeaderName, forged)
				},
			}
			for name, mutate := range cases {
				t.Run(name, func(t *testing.T) {
					r := request()
					mutate(r)
					if m.Validate(r, binding) != ErrInvalid {
						t.Fatal("invalid request accepted")
					}
				})
			}
			clock.now = clock.now.Add(Lifetime)
			if m.Validate(request(), binding) != ErrInvalid {
				t.Fatal("expired token accepted")
			}
			clock.now = clock.now.Add(-Lifetime)
			other, err := m.Issue(httptest.NewRecorder(), request(), binding)
			if err != nil || other == token {
				t.Fatal("nonce not rotated", err)
			}
		})
	}
}

func TestSecureAndConfiguration(t *testing.T) {
	clock := &testClock{time.Now()}
	for _, configured := range []bool{false, true} {
		m := Manager{Key: bytes.Repeat([]byte{2}, 32), Clock: clock, Secure: configured}
		r := httptest.NewRequest("GET", "https://example.test/", nil)
		if configured {
			r.TLS = nil
		} else {
			r.TLS = &tls.ConnectionState{}
		}
		w := httptest.NewRecorder()
		token, err := m.Issue(w, r, "")
		if err != nil {
			t.Fatal(err)
		}
		if !w.Result().Cookies()[0].Secure {
			t.Fatal("secure cookie required")
		}
		r.AddCookie(w.Result().Cookies()[0])
		r.Header.Set(HeaderName, token)
		r.Header.Set("Origin", "https://example.test")
		if err := m.Validate(r, ""); err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Origin", "http://example.test")
		if m.Validate(r, "") != ErrInvalid {
			t.Fatal("downgraded origin accepted")
		}
	}
	m := Manager{}
	if _, err := m.Issue(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil), ""); err != ErrConfiguration {
		t.Fatal(err)
	}
	if m.Validate(nil, "") != ErrConfiguration {
		t.Fatal("unconfigured manager accepted")
	}
}
