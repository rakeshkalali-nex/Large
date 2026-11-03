package httpapi

import (
	"net/http"
	"os"
	"path"
	"strings"

	"buildestate/backend/internal/platform/transport"
)

func recognizedPage(p string) bool {
	if p == "/" {
		return true
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) == 1 {
		switch parts[0] {
		case "login", "reset-password", "setup-password", "change-password", "recovery", "dashboard", "projects", "site-logs", "inspections", "issues", "materials", "inventory", "purchase-orders", "goods-receipts", "stock-movements", "contractors", "work-orders", "budgets", "expenses", "invoices", "payments", "reports", "properties", "leads", "site-visits", "bookings", "documents", "users", "settings", "audit":
			return true
		}
	}
	if len(parts) == 2 {
		switch parts[0] {
		case "projects", "site-logs", "purchase-orders", "contractors", "work-orders", "invoices", "properties", "leads", "bookings":
			return parts[1] != ""
		}
	}
	return len(parts) == 3 && parts[0] == "projects" && parts[1] != "" && parts[2] == "plan"
}

func staticHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			transport.Error(w, r, 404, "NOT_FOUND", "Route not found.")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			transport.Error(w, r, 405, "METHOD_NOT_ALLOWED", "Method not allowed.")
			return
		}
		p := r.URL.Path
		if strings.Contains(p, "\\") || path.Clean(p) != p || strings.Contains(p, "\x00") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(p, "/")
		if recognizedPage(p) {
			name = "index.html"
		} else {
			for _, segment := range strings.Split(name, "/") {
				if strings.HasPrefix(segment, ".") {
					http.NotFound(w, r)
					return
				}
			}
			switch strings.ToLower(path.Ext(name)) {
			case ".js", ".css", ".woff", ".woff2", ".png", ".jpg", ".jpeg", ".svg", ".ico", ".webp":
			default:
				http.NotFound(w, r)
				return
			}
		}
		// os.Root prevents symlink escapes as well as lexical path traversal.
		root, err := os.OpenRoot(dir)
		if err != nil {
			transport.Unavailable(w, r)
			return
		}
		defer root.Close()
		f, err := root.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil || !st.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		if name == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		http.ServeContent(w, r, name, st.ModTime(), f)
	})
}
