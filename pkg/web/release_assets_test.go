package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const releaseSHA = "0123456789abcdef0123456789abcdef01234567"

func TestReleaseAssetServesCurrentWorkerWithoutCDN(t *testing.T) {
	cdnHits := 0
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnHits++
		http.NotFound(w, r)
	}))
	t.Cleanup(cdn.Close)

	handler := NewAssetHandler(http.FS(releaseAssetFS(cdn.URL, "current-worker")), "")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, releaseWorkerPath("editor.worker-current.js"), nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Body.String() != "current-worker" {
		t.Fatalf("expected current worker body, got %q", recorder.Body.String())
	}
	if cdnHits != 0 {
		t.Fatalf("expected no CDN request, got %d", cdnHits)
	}
	if recorder.Header().Get("Content-Type") != "text/javascript; charset=utf-8" {
		t.Fatalf("expected javascript content type, got %q", recorder.Header().Get("Content-Type"))
	}
}

func TestReleaseAssetProxiesOldWorkerAfterDeploy(t *testing.T) {
	oldSHA := "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	var gotCookie string
	var gotPath string
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, "old-worker")
	}))
	t.Cleanup(cdn.Close)

	handler := NewAssetHandler(http.FS(releaseAssetFS(cdn.URL, "current-worker")), "")
	req := httptest.NewRequest(http.MethodGet, "/releases/"+oldSHA+"/assets/editor.worker-old.js", nil)
	req.Header.Set("Cookie", "session=secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Body.String() != "old-worker" {
		t.Fatalf("expected old worker body, got %q", recorder.Body.String())
	}
	if gotPath != "/releases/"+oldSHA+"/assets/editor.worker-old.js" {
		t.Fatalf("expected old release path, got %q", gotPath)
	}
	if gotCookie != "" {
		t.Fatalf("expected no cookie on CDN request, got %q", gotCookie)
	}

	chunk := httptest.NewRecorder()
	handler.ServeHTTP(chunk, httptest.NewRequest(http.MethodGet, "/releases/"+oldSHA+"/assets/chunk-Ab12.js", nil))
	if chunk.Code != http.StatusOK || chunk.Body.String() != "old-worker" {
		t.Fatalf("expected old worker import, got %d %q", chunk.Code, chunk.Body.String())
	}
}

func TestReleaseAssetDoesNotProxyUnrelatedPaths(t *testing.T) {
	cdnHits := 0
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnHits++
		http.Error(w, "should not proxy", http.StatusOK)
	}))
	t.Cleanup(cdn.Close)

	handler := NewAssetHandler(http.FS(releaseAssetFS(cdn.URL, "current-worker")), "/ui")
	paths := []string{
		"/ui/releases/not-a-sha/assets/editor.worker-old.js",
		"/ui/releases/" + releaseSHA + "/assets/../secret.js",
		"/ui/releases/" + releaseSHA + "/assets/index.html",
		"/ui/releases/" + releaseSHA + "/asset-base.txt",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
			req.URL.Path = path
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Body.String() == "should not proxy" || strings.Contains(recorder.Body.String(), "old-worker") {
				t.Fatalf("proxied unrelated path %s", path)
			}
		})
	}

	if cdnHits != 0 {
		t.Fatalf("expected no CDN request, got %d", cdnHits)
	}
}

func TestReleaseAssetDoesNotFollowRedirect(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example/worker.js", http.StatusFound)
	}))
	t.Cleanup(cdn.Close)

	handler := NewAssetHandler(http.FS(releaseAssetFS(cdn.URL, "current-worker")), "")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, releaseWorkerPath("editor.worker-missing.js"), nil))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "" {
		t.Fatalf("expected no redirect, got %q", location)
	}
}

func TestReleaseAssetIgnoresUnsafeAssetBase(t *testing.T) {
	cdnHits := 0
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnHits++
		_, _ = io.WriteString(w, "proxied")
	}))
	t.Cleanup(cdn.Close)

	files := fstest.MapFS{
		"index.html":     &fstest.MapFile{Data: []byte("<html></html>")},
		"asset-base.txt": &fstest.MapFile{Data: []byte("javascript:" + cdn.URL + "/releases/" + releaseSHA + "/\n")},
	}
	handler := NewAssetHandler(http.FS(files), "")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, releaseWorkerPath("editor.worker-old.js"), nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
	if cdnHits != 0 {
		t.Fatalf("expected no CDN request, got %d", cdnHits)
	}
}

func TestReleaseAssetRejectsOversizedBody(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "too-long")
	}))
	t.Cleanup(cdn.Close)

	handler := NewAssetHandler(http.FS(releaseAssetFS(cdn.URL, "current-worker")), "").(*AssetHandler)
	handler.releaseMaxBytes = 4
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, releaseWorkerPath("editor.worker-old.js"), nil))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, recorder.Code)
	}
}

func TestReleaseAssetIgnoresNonReleaseAssetBase(t *testing.T) {
	cdnHits := 0
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnHits++
		_, _ = io.WriteString(w, "proxied")
	}))
	t.Cleanup(cdn.Close)

	files := fstest.MapFS{
		"index.html":     &fstest.MapFile{Data: []byte("<html></html>")},
		"asset-base.txt": &fstest.MapFile{Data: []byte(cdn.URL + "/static/\n")},
	}
	handler := NewAssetHandler(http.FS(files), "")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, releaseWorkerPath("editor.worker-old.js"), nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
	if cdnHits != 0 {
		t.Fatalf("expected no CDN request, got %d", cdnHits)
	}
}

func releaseAssetFS(cdnOrigin string, workerBody string) fstest.MapFS {
	return fstest.MapFS{
		"index.html":                      &fstest.MapFile{Data: []byte("<html></html>")},
		"asset-base.txt":                  &fstest.MapFile{Data: []byte(cdnOrigin + "/releases/" + releaseSHA + "/\n")},
		"assets/editor.worker-current.js": &fstest.MapFile{Data: []byte(workerBody)},
	}
}

func releaseWorkerPath(name string) string {
	return "/releases/" + releaseSHA + "/assets/" + name
}
