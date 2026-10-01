package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
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

func TestReleaseAssetHeadDoesNotDownloadBody(t *testing.T) {
	transport := &releaseCDNTransport{}
	handler := releaseAssetHandler(transport)
	path := releaseWorkerPath("editor.worker-old.js")

	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, path, nil))
	if head.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, head.Code)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("expected empty HEAD body, got %q", head.Body.String())
	}
	if head.Header().Get("Content-Length") != "128" {
		t.Fatalf("expected content length from CDN headers, got %q", head.Header().Get("Content-Length"))
	}
	if transport.calls != 1 || transport.methods[0] != http.MethodHead || transport.reads != 0 {
		t.Fatalf("expected one unread HEAD fetch, got calls=%d methods=%v reads=%d", transport.calls, transport.methods, transport.reads)
	}

	again := httptest.NewRecorder()
	handler.ServeHTTP(again, httptest.NewRequest(http.MethodHead, path, nil))
	if again.Code != http.StatusOK || transport.calls != 1 {
		t.Fatalf("expected cached HEAD, got status %d and %d CDN calls", again.Code, transport.calls)
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, path, nil))
	if get.Code != http.StatusOK || get.Body.String() != "old-worker" {
		t.Fatalf("expected GET body after HEAD, got %d %q", get.Code, get.Body.String())
	}
	if transport.calls != 2 || transport.methods[1] != http.MethodGet {
		t.Fatalf("expected a later GET fetch, got calls=%d methods=%v", transport.calls, transport.methods)
	}
}

func TestReleaseAssetLimitsRepeatedFetches(t *testing.T) {
	transport := &releaseCDNTransport{}
	handler := releaseAssetHandler(transport).(*AssetHandler)
	now := time.Unix(1_700_000_000, 0)
	for i := 0; i < releaseFetchBurst; i++ {
		if !handler.releaseLimiter.beginFetch(now) {
			t.Fatalf("expected fetch %d to be allowed", i+1)
		}
	}
	if handler.releaseLimiter.beginFetch(now) {
		t.Fatal("expected fetch past the burst to be denied")
	}

	handler.releaseLimiter.mu.Lock()
	handler.releaseLimiter.tokens = 0
	handler.releaseLimiter.updated = time.Now().Add(time.Hour)
	handler.releaseLimiter.mu.Unlock()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, releaseWorkerPath("editor.worker-limited.js"), nil))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, recorder.Code)
	}
	if transport.calls != 0 || transport.reads != 0 {
		t.Fatalf("expected no CDN download when limited, got calls=%d reads=%d", transport.calls, transport.reads)
	}
}

func TestReleaseAssetWaitsForBusyFetchSlot(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{}, releaseFetchInflight+1)
	transport := &releaseCDNTransport{block: block, started: started}
	handler := releaseAssetHandler(transport)
	results := make(chan int, releaseFetchInflight+1)

	for i := 0; i < releaseFetchInflight+1; i++ {
		go func(i int) {
			recorder := httptest.NewRecorder()
			path := releaseWorkerPath(fmt.Sprintf("editor.worker-%d.js", i))
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			results <- recorder.Code
		}(i)
	}

	deadline := time.After(2 * time.Second)
	seen := 0
	for seen < releaseFetchInflight {
		select {
		case <-started:
			seen++
		case <-deadline:
			t.Fatalf("timed out after %d CDN calls", seen)
		}
	}

	select {
	case code := <-results:
		t.Fatalf("request finished before a fetch slot was free: %d", code)
	case <-time.After(300 * time.Millisecond):
	}
	close(block)

	waitDeadline := time.After(2 * time.Second)
	for completed := 0; completed < releaseFetchInflight+1; completed++ {
		select {
		case code := <-results:
			if code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, code)
			}
		case <-waitDeadline:
			t.Fatal("timed out waiting for queued fetch")
		}
	}
	if transport.calls != releaseFetchInflight+1 {
		t.Fatalf("expected %d CDN calls, got %d", releaseFetchInflight+1, transport.calls)
	}
}

func TestReleaseAssetKeepsSharedDownloadWhenCallerLeaves(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{}, 1)
	transport := &releaseCDNTransport{block: block, started: started}
	handler := releaseAssetHandler(transport)
	path := releaseWorkerPath("editor.worker-old.js")

	ctx, cancel := context.WithCancel(context.Background())
	firstDone := make(chan int, 1)
	go func() {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
		handler.ServeHTTP(recorder, req)
		firstDone <- recorder.Code
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shared CDN fetch")
	}

	secondDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		secondDone <- recorder
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case recorder := <-secondDone:
		t.Fatalf("second caller finished before the shared download: %d %q", recorder.Code, recorder.Body.String())
	case <-time.After(200 * time.Millisecond):
	}
	if transport.calls != 1 {
		t.Fatalf("expected one shared CDN fetch, got %d", transport.calls)
	}
	close(block)

	select {
	case recorder := <-secondDone:
		if recorder.Code != http.StatusOK || recorder.Body.String() != "old-worker" {
			t.Fatalf("expected shared worker body, got %d %q", recorder.Code, recorder.Body.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the second caller")
	}
	if transport.calls != 1 {
		t.Fatalf("expected the canceled caller not to force another fetch, got %d", transport.calls)
	}

	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the canceled caller")
	}
}

func TestReleaseAssetStopsQueuedFetchWhenCallerLeaves(t *testing.T) {
	block := make(chan struct{})
	transport := &releaseCDNTransport{block: block, started: make(chan struct{}, 16)}
	handler := releaseAssetHandler(transport)
	release := func() {
		select {
		case <-block:
		default:
			close(block)
		}
	}
	t.Cleanup(release)
	occupyReleaseFetchSlots(t, handler, transport)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, releaseWorkerPath("editor.worker-queued.js"), nil).WithContext(ctx)
		handler.ServeHTTP(recorder, req)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("queued fetch continued after the caller left")
	}
	if transport.callCount() != releaseFetchInflight {
		t.Fatalf("expected no CDN fetch for a caller that left, got %d", transport.callCount())
	}

	liveDone := make(chan int, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, releaseWorkerPath("editor.worker-live.js"), nil))
		liveDone <- recorder.Code
	}()
	select {
	case code := <-liveDone:
		t.Fatalf("live fetch finished before a slot was free: %d", code)
	case <-time.After(200 * time.Millisecond):
	}
	release()

	select {
	case code := <-liveDone:
		if code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the live fetch")
	}
	if transport.callCount() != releaseFetchInflight+1 {
		t.Fatalf("expected one live CDN fetch, got %d", transport.callCount())
	}
}

func TestReleaseAssetKeepsQueuedFetchForRemainingCaller(t *testing.T) {
	block := make(chan struct{})
	transport := &releaseCDNTransport{block: block, started: make(chan struct{}, 16)}
	handler := releaseAssetHandler(transport)
	t.Cleanup(func() {
		select {
		case <-block:
		default:
			close(block)
		}
	})
	occupyReleaseFetchSlots(t, handler, transport)

	path := releaseWorkerPath("editor.worker-queued.js")
	ctx, cancel := context.WithCancel(context.Background())
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx))
	}()

	secondDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		secondDone <- recorder
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case recorder := <-secondDone:
		t.Fatalf("remaining caller finished before a fetch slot was free: %d %q", recorder.Code, recorder.Body.String())
	case <-time.After(200 * time.Millisecond):
	}
	if transport.callCount() != releaseFetchInflight {
		t.Fatalf("expected the queued fetch to stay shared, got %d CDN calls", transport.callCount())
	}
	close(block)

	select {
	case recorder := <-secondDone:
		if recorder.Code != http.StatusOK || recorder.Body.String() != "old-worker" {
			t.Fatalf("expected queued worker body, got %d %q", recorder.Code, recorder.Body.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the remaining caller")
	}
	if transport.callCount() != releaseFetchInflight+1 {
		t.Fatalf("expected one shared CDN fetch, got %d", transport.callCount())
	}

	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the canceled caller")
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

func occupyReleaseFetchSlots(t *testing.T, handler http.Handler, transport *releaseCDNTransport) {
	t.Helper()
	for i := 0; i < releaseFetchInflight; i++ {
		go func(i int) {
			recorder := httptest.NewRecorder()
			path := releaseWorkerPath(fmt.Sprintf("editor.worker-busy-%d.js", i))
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		}(i)
	}

	deadline := time.After(2 * time.Second)
	for seen := 0; seen < releaseFetchInflight; seen++ {
		select {
		case <-transport.started:
		case <-deadline:
			t.Fatalf("timed out after %d CDN calls", seen)
		}
	}
}

func releaseAssetHandler(transport http.RoundTripper) http.Handler {
	handler := NewAssetHandler(http.FS(releaseAssetFS("https://assets.example", "current-worker")), "").(*AssetHandler)
	handler.assetCDNClient = &http.Client{Transport: transport}
	return handler
}

type releaseCDNTransport struct {
	mu      sync.Mutex
	calls   int
	reads   int
	methods []string
	block   <-chan struct{}
	started chan struct{}
}

func (t *releaseCDNTransport) callCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.calls
}

func (t *releaseCDNTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.calls++
	t.methods = append(t.methods, req.Method)
	t.mu.Unlock()
	if t.started != nil {
		t.started <- struct{}{}
	}
	if t.block != nil {
		select {
		case <-t.block:
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}

	body := io.NopCloser(strings.NewReader("old-worker"))
	length := int64(len("old-worker"))
	if req.Method == http.MethodHead {
		body = io.NopCloser(&failingReader{counter: t})
		length = 128
	}
	header := make(http.Header)
	header.Set("Content-Length", fmt.Sprintf("%d", length))
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        header,
		Body:          body,
		ContentLength: length,
		Request:       req,
	}, nil
}

type failingReader struct {
	counter *releaseCDNTransport
}

func (r *failingReader) Read([]byte) (int, error) {
	r.counter.mu.Lock()
	r.counter.reads++
	r.counter.mu.Unlock()
	return 0, io.ErrUnexpectedEOF
}
