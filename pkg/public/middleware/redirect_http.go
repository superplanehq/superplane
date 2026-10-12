package middleware

import (
	"net/http"
	"strings"
)

// RedirectInsecureForwardedHTTP sends browser and API requests that arrived
// as HTTP through a TLS-terminating proxy to HTTPS.
//
// GCE FrontendConfig redirect applies to every request on port 80. Let's
// Encrypt HTTP-01 uses that same port, so the load balancer redirect stays
// off for cert-manager. This wrapper redirects the application instead.
// Health checks and the ACME challenge path are left on HTTP.
func RedirectInsecureForwardedHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !insecureForwardedHTTP(r) || skipHTTPRedirect(r.URL.Path) || r.Host == "" {
			next.ServeHTTP(w, r)
			return
		}

		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
}

func insecureForwardedHTTP(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "http")
}

func skipHTTPRedirect(path string) bool {
	return path == "/health" || strings.HasPrefix(path, "/.well-known/acme-challenge")
}
