package public

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func Test__UnknownAPIPathsDoNotFallThroughToSPA(t *testing.T) {
	router := mux.NewRouter()
	router.PathPrefix("/api/v1/canvases").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "canvases")
	})
	router.HandleFunc("/admin/api/installation/network-settings", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}).Methods(http.MethodGet)

	var spaHits atomic.Int32
	registerUnknownAPINotFound(router)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		spaHits.Add(1)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "<html>spa</html>")
	})

	t.Run("removed canvas-folders API returns 404", func(t *testing.T) {
		spaHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/canvas-folders", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.NotContains(t, rec.Body.String(), "spa")
		assert.Equal(t, int32(0), spaHits.Load())
	})

	t.Run("known canvases API still matches", func(t *testing.T) {
		spaHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "canvases", rec.Body.String())
		assert.Equal(t, int32(0), spaHits.Load())
	})

	t.Run("known admin API still matches", func(t *testing.T) {
		spaHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/admin/api/installation/network-settings", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, int32(0), spaHits.Load())
	})

	t.Run("unknown admin API returns 404", func(t *testing.T) {
		spaHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/admin/api/missing", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.NotContains(t, rec.Body.String(), "spa")
		assert.Equal(t, int32(0), spaHits.Load())
	})

	t.Run("UI path still reaches SPA", func(t *testing.T) {
		spaHits.Store(0)
		req := httptest.NewRequest(http.MethodGet, "/apps", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "spa")
		assert.Equal(t, int32(1), spaHits.Load())
	})
}
