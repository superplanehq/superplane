package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReleaseAssetProxyServesHistoricalWorker(t *testing.T) {
	const workerPath = "/releases/1ac57e49e2cf11975bc1ece6eab40c3681b6dbf5/assets/json.worker-DIO6m__1.js"
	const workerBody = "export const worker = true;"

	var sawCookie bool
	var sawPath string
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		if r.Header.Get("Cookie") != "" {
			sawCookie = true
		}
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Header().Set("ETag", `"worker"`)
		w.Header().Set("Set-Cookie", "secret=1")
		_, _ = io.WriteString(w, workerBody)
	}))
	t.Cleanup(cdn.Close)

	handler := newTestReleaseAssetProxy(t, cdn.URL)
	req := httptest.NewRequest(http.MethodGet, workerPath, nil)
	req.Header.Set("Cookie", "session=secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if sawPath != workerPath {
		t.Fatalf("upstream path = %q", sawPath)
	}
	if sawCookie {
		t.Fatal("forwarded the page cookie to the asset host")
	}
	if recorder.Body.String() != workerBody {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if recorder.Header().Get("Set-Cookie") != "" {
		t.Fatal("copied Set-Cookie from the asset host")
	}
	if recorder.Header().Get("Content-Type") != "text/javascript" {
		t.Fatalf("content type = %q", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("ETag") != `"worker"` {
		t.Fatalf("etag = %q", recorder.Header().Get("ETag"))
	}
}

func TestReleaseAssetProxyCachesOnlySuccessfulAssets(t *testing.T) {
	const workerPath = "/releases/abc1234/assets/json.worker.js"
	cases := []struct {
		name          string
		status        int
		upstreamCache string
		wantCache     string
	}{
		{
			name:      "success without cache control",
			status:    http.StatusOK,
			wantCache: "public, max-age=31536000",
		},
		{
			name:          "success keeps upstream cache control",
			status:        http.StatusOK,
			upstreamCache: "public, max-age=60",
			wantCache:     "public, max-age=60",
		},
		{
			name:      "not found is not cached",
			status:    http.StatusNotFound,
			wantCache: "no-store",
		},
		{
			name:      "unavailable is not cached",
			status:    http.StatusServiceUnavailable,
			wantCache: "no-store",
		},
		{
			name:          "error keeps upstream cache control",
			status:        http.StatusNotFound,
			upstreamCache: "max-age=0",
			wantCache:     "max-age=0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.upstreamCache != "" {
					w.Header().Set("Cache-Control", tc.upstreamCache)
				}
				w.WriteHeader(tc.status)
			}))
			t.Cleanup(cdn.Close)

			handler := newTestReleaseAssetProxy(t, cdn.URL)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, workerPath, nil))

			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.status)
			}
			if got := recorder.Header().Get("Cache-Control"); got != tc.wantCache {
				t.Fatalf("cache control = %q, want %q", got, tc.wantCache)
			}
		})
	}
}

func TestAssetCDNOriginUsesDefaultOrConfiguredHost(t *testing.T) {
	t.Setenv("ASSET_CDN_ORIGIN", "   ")
	if got := AssetCDNOrigin(); got != DefaultAssetCDNOrigin {
		t.Fatalf("origin = %q, want %q", got, DefaultAssetCDNOrigin)
	}

	t.Setenv("ASSET_CDN_ORIGIN", "  https://cdn.example  ")
	if got := AssetCDNOrigin(); got != "https://cdn.example" {
		t.Fatalf("origin = %q", got)
	}
}

func TestReleaseAssetProxyRejectsOtherMethods(t *testing.T) {
	var upstreamHits int
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(cdn.Close)

	handler := newTestReleaseAssetProxy(t, cdn.URL)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/releases/abc1234/assets/json.worker.js", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("allow = %q", recorder.Header().Get("Allow"))
	}
	if upstreamHits != 0 {
		t.Fatalf("upstream hits = %d", upstreamHits)
	}
}

func TestReleaseAssetProxyHeadOmitsBody(t *testing.T) {
	const workerPath = "/releases/abc1234/assets/json.worker.js"
	var sawMethod string
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = io.WriteString(w, "export const worker = true;")
	}))
	t.Cleanup(cdn.Close)

	handler := newTestReleaseAssetProxy(t, cdn.URL)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, workerPath, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if sawMethod != http.MethodHead {
		t.Fatalf("upstream method = %q", sawMethod)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "text/javascript" {
		t.Fatalf("content type = %q", recorder.Header().Get("Content-Type"))
	}
}

func TestReleaseAssetProxyReportsUpstreamFailure(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cdn.Close()

	handler := newTestReleaseAssetProxy(t, cdn.URL)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/releases/abc1234/assets/json.worker.js", nil))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.TrimSpace(recorder.Body.String()) != "asset unavailable" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestReleaseAssetProxyRejectsPathsOutsideReleaseAssets(t *testing.T) {
	var upstreamHits int
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(cdn.Close)

	handler := newTestReleaseAssetProxy(t, cdn.URL)
	paths := []string{
		"/releases/not-a-sha/assets/json.worker.js",
		"/releases/abc1234/private/json.worker.js",
		"/releases/abc1234/assets/../secret.js",
		"/releases/abc1234/assets/nested/json.worker.js",
		"/assets/json.worker.js",
	}
	for _, path := range paths {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d", path, recorder.Code)
		}
	}
	if upstreamHits != 0 {
		t.Fatalf("upstream hits = %d", upstreamHits)
	}
}

func TestReleaseAssetProxyDoesNotFollowRedirects(t *testing.T) {
	var internalHits int
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internalHits++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(internal.Close)

	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL+"/secret", http.StatusFound)
	}))
	t.Cleanup(cdn.Close)

	handler := newTestReleaseAssetProxy(t, cdn.URL)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/releases/abc1234/assets/json.worker.js", nil),
	)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Header().Get("Location") != "" {
		t.Fatalf("location = %q", recorder.Header().Get("Location"))
	}
	if internalHits != 0 {
		t.Fatalf("redirect target hits = %d", internalHits)
	}
}

func newTestReleaseAssetProxy(t *testing.T, origin string) http.Handler {
	t.Helper()
	handler, err := NewReleaseAssetProxy(origin)
	if err != nil {
		t.Fatalf("NewReleaseAssetProxy: %v", err)
	}
	return handler
}

func TestNewReleaseAssetProxyRejectsUnsafeOrigin(t *testing.T) {
	origins := []string{
		"",
		"assets.superplane.com",
		"file:///tmp/assets",
		"https://user:pass@assets.superplane.com",
	}
	for _, origin := range origins {
		if _, err := NewReleaseAssetProxy(origin); err == nil {
			t.Fatalf("origin %q was accepted", origin)
		}
	}
}
