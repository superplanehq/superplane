package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPprofHandlerUsesDedicatedMux(t *testing.T) {
	previousDefaultMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() {
		http.DefaultServeMux = previousDefaultMux
	})

	http.HandleFunc("/default-mux-only", func(http.ResponseWriter, *http.Request) {})
	handler := pprofHandler()

	profileRequest := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	profileResponse := httptest.NewRecorder()
	handler.ServeHTTP(profileResponse, profileRequest)

	defaultMuxRequest := httptest.NewRequest(http.MethodGet, "/default-mux-only", nil)
	defaultMuxResponse := httptest.NewRecorder()
	handler.ServeHTTP(defaultMuxResponse, defaultMuxRequest)

	assert.Equal(t, http.StatusOK, profileResponse.Code)
	assert.Equal(t, http.StatusNotFound, defaultMuxResponse.Code)
}
