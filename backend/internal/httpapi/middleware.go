package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"buildestate/backend/internal/platform/transport"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func Middleware(next http.Handler, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			transport.Error(w, r, 500, "INTERNAL_ERROR", "Request could not be completed.")
			return
		}
		random[6] = (random[6] & 0x0f) | 0x40
		random[8] = (random[8] & 0x3f) | 0x80
		h := hex.EncodeToString(random[:])
		id := h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
		budget := 30 * time.Second
		if strings.HasPrefix(r.URL.Path, "/api/v1/reports/") {
			budget = 35 * time.Second
		}
		if r.URL.Path == "/api/v1/documents/uploads" {
			budget = 120 * time.Second
		}
		ctx, cancel := context.WithTimeout(r.Context(), budget)
		defer cancel()
		r = r.WithContext(transport.WithRequestID(ctx, id))
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' blob: data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		rw := &responseWriter{ResponseWriter: w}
		started := time.Now()
		defer func() {
			if recover() != nil {
				if rw.status == 0 {
					transport.Error(rw, r, 500, "INTERNAL_ERROR", "Request could not be completed.")
				}
				logger.Error("request panic", "requestId", id)
			}
			// Never log URLs, queries, bodies, cookies, credentials or panic values.
			logger.Info("request", "requestId", id, "method", r.Method, "status", rw.status, "durationMs", time.Since(started).Milliseconds())
		}()
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/v1/documents/uploads" {
			if r.ContentLength > transport.MaxJSONBody {
				transport.Error(rw, r, 413, "BODY_TOO_LARGE", "Request body is too large.")
				return
			}
			r.Body = http.MaxBytesReader(rw, r.Body, transport.MaxJSONBody)
		}
		next.ServeHTTP(rw, r)
	})
}
