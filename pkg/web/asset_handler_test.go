package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestAssetHandlerNoIndexHeader(t *testing.T) {
	tests := []struct {
		name       string
		appEnv     string
		path       string
		wantStatus int
		wantHeader string
	}{
		{
			name:       "index in staging includes noindex header",
			appEnv:     "staging",
			path:       "/",
			wantStatus: http.StatusOK,
			wantHeader: "noindex",
		},
		{
			name:       "asset in development includes noindex header",
			appEnv:     "development",
			path:       "/assets/main.js",
			wantStatus: http.StatusOK,
			wantHeader: "noindex",
		},
		{
			name:       "missing asset in test includes noindex header",
			appEnv:     "test",
			path:       "/assets/missing.js",
			wantStatus: http.StatusNotFound,
			wantHeader: "noindex",
		},
		{
			name:       "index in production does not include noindex header",
			appEnv:     "production",
			path:       "/",
			wantStatus: http.StatusOK,
			wantHeader: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.appEnv)

			handler := NewAssetHandler(http.FS(fstest.MapFS{
				"index.html":     &fstest.MapFile{Data: []byte("<html><body>Hello</body></html>")},
				"assets/main.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
			}), "")

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}

			gotHeader := recorder.Header().Get("X-Robots-Tag")
			if gotHeader != tt.wantHeader {
				t.Fatalf("expected X-Robots-Tag %q, got %q", tt.wantHeader, gotHeader)
			}
		})
	}
}
