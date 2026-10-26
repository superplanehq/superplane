package polar

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func APIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Polar-Version", r.Header.Get("Polar-Version"))
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func UseServer(t *testing.T, server *httptest.Server) {
	t.Helper()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
}
