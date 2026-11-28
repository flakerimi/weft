package middleware

import "net/http"

// The security headers every answer carries. There is no script-src: the
// page build writes inline scripts that a policy would need their hashes for.
const (
	contentSecurityPolicy = "frame-ancestors 'none'; base-uri 'self'; object-src 'none'; form-action 'self'"
	strictTransport       = "max-age=31536000"
)

// headers sets the security headers on the answer before next runs, so a
// handler can replace one; Strict-Transport-Security only over HTTPS, as
// the app's IsHTTPS sees it.
func (m *module) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		if m.app.IsHTTPS(r) {
			h.Set("Strict-Transport-Security", strictTransport)
		}
		next.ServeHTTP(w, r)
	})
}
