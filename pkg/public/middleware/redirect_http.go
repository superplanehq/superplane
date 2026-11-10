package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// RedirectInsecureForwardedHTTP sends browser and API requests that arrived
// as HTTP through a TLS-terminating proxy to HTTPS.
//
// GCE FrontendConfig redirect applies to every request on port 80. Let's
// Encrypt HTTP-01 uses that same port, so the load balancer redirect stays
// off for cert-manager. This wrapper redirects the application instead.
// Health checks and the ACME challenge path are left on HTTP.
func RedirectInsecureForwardedHTTP(next http.Handler, baseURL string) http.Handler {
	canonicalURL, err := url.Parse(baseURL)
	if err != nil || canonicalURL.Host == "" {
		canonicalURL = nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !insecureForwardedHTTP(r) || skipHTTPRedirect(r.URL.Path) || canonicalURL == nil {
			next.ServeHTTP(w, r)
			return
		}

		target := *r.URL
		target.Scheme = "https"
		target.Host = canonicalURL.Host
		http.Redirect(w, r, target.String(), http.StatusPermanentRedirect) // #nosec G710 -- the host comes from the configured canonical base URL.
	})
}

func insecureForwardedHTTP(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "http")
}

func skipHTTPRedirect(path string) bool {
	return path == "/health" || strings.HasPrefix(path, "/.well-known/acme-challenge")
}
