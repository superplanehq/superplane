package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectInsecureForwardedHTTP(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := RedirectInsecureForwardedHTTP(next)

	t.Run("redirects forwarded HTTP login", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "http://superplane.example/login?next=/", nil)
		request.Host = "superplane.example"
		request.Header.Set("X-Forwarded-Proto", "http")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusPermanentRedirect, response.Code)
		assert.Equal(t, "https://superplane.example/login?next=/", response.Header().Get("Location"))
	})

	t.Run("keeps HTTPS and direct probes", func(t *testing.T) {
		for _, header := range []string{"https", ""} {
			request := httptest.NewRequest(http.MethodGet, "http://superplane.example/login", nil)
			request.Host = "superplane.example"
			if header != "" {
				request.Header.Set("X-Forwarded-Proto", header)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, http.StatusOK, response.Code, header)
		}
	})

	t.Run("leaves health and ACME challenge on HTTP", func(t *testing.T) {
		for _, path := range []string{"/health", "/.well-known/acme-challenge/token"} {
			request := httptest.NewRequest(http.MethodGet, "http://superplane.example"+path, nil)
			request.Host = "superplane.example"
			request.Header.Set("X-Forwarded-Proto", "http")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, http.StatusOK, response.Code, path)
		}
	})
}
