// Package csrf implements signed, session-bound double-submit browser tokens.
package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"buildestate/backend/internal/contracts"
)

const (
	CookieName = "buildestate_csrf"
	HeaderName = "X-CSRF-Token"
	Lifetime   = 30 * time.Minute
)

var (
	ErrInvalid       = errors.New("invalid CSRF token or origin")
	ErrConfiguration = errors.New("CSRF requires a key of at least 32 bytes and a clock")
)

type Manager struct {
	Key    []byte
	Clock  contracts.Clock
	Secure bool
}

// Issue rotates the nonce. binding is the trusted session token digest, or empty
// for anonymous requests; it must never come from a browser-supplied identity.
func (m Manager) Issue(w http.ResponseWriter, r *http.Request, binding string) (string, error) {
	if len(m.Key) < 32 || m.Clock == nil {
		return "", ErrConfiguration
	}
	if r == nil {
		return "", ErrInvalid
	}
	expires := m.Clock.Now().UTC().Add(Lifetime).Truncate(time.Second)
	payload := make([]byte, 72, 104)
	binary.BigEndian.PutUint64(payload[:8], uint64(expires.Unix()))
	if _, err := rand.Read(payload[8:40]); err != nil {
		return "", err
	}
	bound := sha256.Sum256([]byte(binding))
	copy(payload[40:72], bound[:])
	token := base64.RawURLEncoding.EncodeToString(append(payload, m.signature(payload)...))
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: token, Path: "/", Expires: expires,
		MaxAge: int(Lifetime / time.Second), Secure: m.Secure || r.TLS != nil, HttpOnly: false, SameSite: http.SameSiteLaxMode})
	w.Header().Set("Cache-Control", "no-store")
	return token, nil
}

// Validate requires an exact Origin even for anonymous login/reset requests.
// Forwarded authority is rejected rather than used to construct the origin.
func (m Manager) Validate(r *http.Request, binding string) error {
	if len(m.Key) < 32 || m.Clock == nil {
		return ErrConfiguration
	}
	if r == nil || !m.sameOrigin(r) {
		return ErrInvalid
	}
	var cookie string
	count := 0
	for _, c := range r.Cookies() {
		if c.Name == CookieName {
			cookie = c.Value
			count++
		}
	}
	headers := r.Header.Values(HeaderName)
	if count != 1 || len(headers) != 1 || len(cookie) != 139 || len(headers[0]) != 139 {
		return ErrInvalid
	}
	if subtle.ConstantTimeCompare([]byte(cookie), []byte(headers[0])) != 1 {
		return ErrInvalid
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(cookie)
	if err != nil || len(raw) != 104 {
		return ErrInvalid
	}
	bound := sha256.Sum256([]byte(binding))
	signatureOK := hmac.Equal(raw[72:], m.signature(raw[:72]))
	bindingOK := subtle.ConstantTimeCompare(raw[40:72], bound[:]) == 1
	if !signatureOK || !bindingOK {
		return ErrInvalid
	}
	now := m.Clock.Now().Unix()
	expires := int64(binary.BigEndian.Uint64(raw[:8]))
	if expires <= now || expires > now+int64(Lifetime/time.Second) {
		return ErrInvalid
	}
	return nil
}

func (m Manager) signature(payload []byte) []byte {
	mac := hmac.New(sha256.New, m.Key)
	mac.Write([]byte("buildestate/csrf/v1\x00"))
	mac.Write(payload)
	return mac.Sum(nil)
}

func (m Manager) sameOrigin(r *http.Request) bool {
	for _, header := range []string{"Forwarded", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-Port"} {
		if len(r.Header.Values(header)) != 0 {
			return false
		}
	}
	scheme := "http"
	if m.Secure || r.TLS != nil {
		scheme = "https"
	}
	if r.Host == "" || strings.ContainsAny(r.Host, "/\\?#@ \t\r\n") {
		return false
	}
	expected := scheme + "://" + r.Host
	u, err := url.Parse(expected)
	if err != nil || u.Host != r.Host || u.Hostname() == "" {
		return false
	}
	origins := r.Header.Values("Origin")
	return len(origins) == 1 && origins[0] == expected
}
