package public

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__DevProxy__DoesNotForwardAdminAPIToVite(t *testing.T) {
	var viteHits atomic.Int32
	vite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viteHits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "vite")
	}))
	t.Cleanup(vite.Close)

	viteURL, err := url.Parse(vite.URL)
	require.NoError(t, err)
	t.Setenv("VITE_DEV_HOST", viteURL.Hostname())
	t.Setenv("VITE_DEV_PORT", viteURL.Port())

	router := mux.NewRouter()
	router.HandleFunc("/admin/api/installation/network-settings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}).Methods(http.MethodGet)

	server := &Server{isDev: true, Router: router}
	server.setupDevProxy("")

	t.Run("registered admin API stays on Go", func(t *testing.T) {
		viteHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/admin/api/installation/network-settings", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, int32(0), viteHits.Load())
	})

	t.Run("unknown admin API does not proxy to Vite", func(t *testing.T) {
		viteHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/admin/api/installation/llm-settings", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.NotContains(t, rec.Body.String(), "vite")
		assert.Equal(t, int32(0), viteHits.Load())
	})

	t.Run("admin UI still proxies to Vite", func(t *testing.T) {
		viteHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "vite", rec.Body.String())
		assert.Equal(t, int32(1), viteHits.Load())
	})
}
