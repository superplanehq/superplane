package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
		wantBody   string
		wantType   string
		wantCache  string
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
		{
			name:       "onboarding script under assets is the file, not the app page",
			appEnv:     "production",
			path:       "/assets/onboarding/three.min.js",
			wantStatus: http.StatusOK,
			wantHeader: "",
			wantBody:   "window.THREE = {}",
			wantType:   "javascript",
			wantCache:  "public, max-age=3600",
		},
		{
			name:       "unprefixed onboarding script is the app page",
			appEnv:     "production",
			path:       "/onboarding/three.min.js",
			wantStatus: http.StatusOK,
			wantHeader: "",
			wantBody:   "<html><body>Hello</body></html>",
			wantType:   "text/html",
		},
		{
			name:       "web app manifest is the manifest file, not the app page",
			appEnv:     "production",
			path:       "/manifest.webmanifest",
			wantStatus: http.StatusOK,
			wantHeader: "",
			wantBody:   `{"name":"SuperPlane","display":"standalone"}`,
			wantType:   "application/manifest+json",
			wantCache:  "public, max-age=3600",
		},
		{
			name:       "pwa icon under assets is the image, not the app page",
			appEnv:     "production",
			path:       "/assets/pwa/icon-192.png",
			wantStatus: http.StatusOK,
			wantHeader: "",
			wantBody:   "png-bytes",
			wantType:   "image/png",
			wantCache:  "public, max-age=3600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.appEnv)

			handler := NewAssetHandler(http.FS(fstest.MapFS{
				"index.html":                     &fstest.MapFile{Data: []byte("<html><body>Hello</body></html>")},
				"assets/main.js":                 &fstest.MapFile{Data: []byte("console.log('ok')")},
				"assets/onboarding/three.min.js": &fstest.MapFile{Data: []byte("window.THREE = {}")},
				"manifest.webmanifest":           &fstest.MapFile{Data: []byte(`{"name":"SuperPlane","display":"standalone"}`)},
				"assets/pwa/icon-192.png":        &fstest.MapFile{Data: []byte("png-bytes")},
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
			if tt.wantBody != "" && recorder.Body.String() != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, recorder.Body.String())
			}
			if tt.wantType != "" && !strings.Contains(recorder.Header().Get("Content-Type"), tt.wantType) {
				t.Fatalf("expected content type to contain %q, got %q", tt.wantType, recorder.Header().Get("Content-Type"))
			}
			if tt.wantCache != "" && recorder.Header().Get("Cache-Control") != tt.wantCache {
				t.Fatalf("expected Cache-Control %q, got %q", tt.wantCache, recorder.Header().Get("Cache-Control"))
			}
		})
	}
}
