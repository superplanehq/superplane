package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const releaseAssetMaxBytes int64 = 32 << 20
const releaseFetchRate = 10
const releaseFetchBurst = 40
const releaseFetchInflight = 8
const releaseFetchTimeout = 15 * time.Second
const releaseFetchWait = 30 * time.Second
const releaseCacheTTL = 15 * time.Minute
const releaseCacheEntries = 128
const releaseCacheMaxBytes = 16 << 20

var releaseAssetPathPattern = regexp.MustCompile(`^/releases/([0-9a-f]{40})/assets/([A-Za-z0-9._-]{1,200}\.(?:js|mjs|wasm))$`)
var releaseAssetBasePattern = regexp.MustCompile(`^/releases/[0-9a-f]{40}/?$`)

type releaseAsset struct {
	path string
	name string
}

type cachedReleaseAsset struct {
	status        int
	contentLength int64
	body          []byte
	hasBody       bool
	expires       time.Time
}

type releaseAssetCache struct {
	mu    sync.Mutex
	items map[string]cachedReleaseAsset
	order []string
	bytes int
}

type releaseFetchLimiter struct {
	mu      sync.Mutex
	tokens  float64
	updated time.Time
	slots   chan struct{}
}

func newReleaseAssetCache() *releaseAssetCache {
	return &releaseAssetCache{items: make(map[string]cachedReleaseAsset)}
}

func newReleaseFetchLimiter() *releaseFetchLimiter {
	return &releaseFetchLimiter{
		tokens:  releaseFetchBurst,
		updated: time.Now(),
		slots:   make(chan struct{}, releaseFetchInflight),
	}
}

func (c *releaseAssetCache) get(path, method string, now time.Time) (cachedReleaseAsset, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[path]
	if !ok || !now.Before(item.expires) {
		return cachedReleaseAsset{}, false
	}
	if method == http.MethodGet && item.status == http.StatusOK && !item.hasBody {
		return cachedReleaseAsset{}, false
	}

	return item, true
}

func (c *releaseAssetCache) store(path string, item cachedReleaseAsset) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item.hasBody && len(item.body) > releaseCacheMaxBytes {
		item.body = nil
		item.hasBody = false
	}
	c.deleteLocked(path)
	for len(c.order) > 0 && (len(c.items) >= releaseCacheEntries || c.bytes+len(item.body) > releaseCacheMaxBytes) {
		c.deleteLocked(c.order[0])
	}
	if item.hasBody && c.bytes+len(item.body) > releaseCacheMaxBytes {
		item.body = nil
		item.hasBody = false
	}

	c.items[path] = item
	c.order = append(c.order, path)
	c.bytes += len(item.body)
}

func (c *releaseAssetCache) deleteLocked(path string) {
	if item, ok := c.items[path]; ok {
		c.bytes -= len(item.body)
		delete(c.items, path)
	}

	kept := c.order[:0]
	for _, name := range c.order {
		if name != path {
			kept = append(kept, name)
		}
	}
	c.order = kept
}

func (l *releaseFetchLimiter) beginFetch(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	elapsed := now.Sub(l.updated).Seconds()
	if elapsed > 0 {
		l.tokens += elapsed * releaseFetchRate
		if l.tokens > releaseFetchBurst {
			l.tokens = releaseFetchBurst
		}
		l.updated = now
	}
	if l.tokens < 1 {
		return false
	}

	l.tokens--
	return true
}

func (l *releaseFetchLimiter) refundFetch() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.tokens < releaseFetchBurst {
		l.tokens++
	}
}

func (l *releaseFetchLimiter) waitSlot(ctx context.Context) error {
	select {
	case l.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *releaseFetchLimiter) releaseSlot() {
	<-l.slots
}

type releaseCall struct {
	ctx    context.Context
	cancel context.CancelFunc
	n      int
}

type releaseCallerSet struct {
	mu    sync.Mutex
	calls map[string]*releaseCall
}

func newReleaseCallerSet() *releaseCallerSet {
	return &releaseCallerSet{calls: make(map[string]*releaseCall)}
}

func (s *releaseCallerSet) join(key string, onEmpty func()) (context.Context, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	call := s.calls[key]
	if call == nil {
		ctx, cancel := context.WithCancel(context.Background())
		call = &releaseCall{ctx: ctx, cancel: cancel}
		s.calls[key] = call
	}
	call.n++

	return call.ctx, func() {
		s.leave(key, call, onEmpty)
	}
}

func (s *releaseCallerSet) leave(key string, call *releaseCall, onEmpty func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.calls[key] != call {
		return
	}
	call.n--
	if call.n > 0 {
		return
	}

	call.cancel()
	delete(s.calls, key)
	onEmpty()
}

func newAssetCDNClient() *http.Client {
	return &http.Client{
		Timeout: releaseFetchTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func assetCDNOrigin(assets http.FileSystem) string {
	file, err := assets.Open("asset-base.txt")
	if err != nil {
		return ""
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 512))
	if err != nil {
		return ""
	}

	return cdnOriginFromAssetBase(string(data))
}

func cdnOriginFromAssetBase(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ""
	}
	if !releaseAssetBasePattern.MatchString(parsed.Path) && !releaseAssetBasePattern.MatchString(parsed.Path+"/") {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}

func parseReleaseAsset(path string) (releaseAsset, bool) {
	match := releaseAssetPathPattern.FindStringSubmatch(path)
	if match == nil {
		return releaseAsset{}, false
	}

	return releaseAsset{path: path, name: match[2]}, true
}

func (h *AssetHandler) relativeRequestPath(path string) string {
	if h.basePath == "" {
		return path
	}

	trimmed := strings.TrimPrefix(path, h.basePath)
	if trimmed == "" {
		return "/"
	}

	return trimmed
}

func (h *AssetHandler) serveReleaseAsset(w http.ResponseWriter, r *http.Request) bool {
	asset, ok := parseReleaseAsset(h.relativeRequestPath(r.URL.Path))
	if !ok {
		return false
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return true
	}

	if h.serveExistingFile(w, r, "assets/"+asset.name, releaseAssetContentType(asset.name)) {
		return true
	}

	if h.assetCDNOrigin == "" {
		http.NotFound(w, r)
		return true
	}

	h.proxyReleaseAsset(w, r, asset)
	return true
}

func (h *AssetHandler) proxyReleaseAsset(w http.ResponseWriter, r *http.Request, asset releaseAsset) {
	if item, ok := h.releaseCache.get(asset.path, r.Method, time.Now()); ok {
		writeReleaseAsset(w, r, asset, item)
		return
	}

	key := r.Method + " " + asset.path
	shared, leave := h.joinReleaseCall(key)
	stop := stopReleaseCallWhenRequestEnds(r.Context(), leave)
	defer stop()

	value, err, _ := h.releaseGroup.Do(key, func() (any, error) {
		return h.loadReleaseAsset(shared, asset, r.Method)
	})
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		log.Warnf("release asset proxy failed: %v", err)
		http.Error(w, "release asset unavailable", http.StatusBadGateway)
		return
	}

	writeReleaseAsset(w, r, asset, value.(cachedReleaseAsset))
}

func (h *AssetHandler) joinReleaseCall(key string) (context.Context, func()) {
	return h.releaseCallers.join(key, func() {
		h.releaseGroup.Forget(key)
	})
}

func stopReleaseCallWhenRequestEnds(requestCtx context.Context, leave func()) func() {
	var once sync.Once
	leaveOnce := func() {
		once.Do(leave)
	}
	stopWatch := context.AfterFunc(requestCtx, leaveOnce)
	return func() {
		stopWatch()
		leaveOnce()
	}
}

func (h *AssetHandler) loadReleaseAsset(ctx context.Context, asset releaseAsset, method string) (cachedReleaseAsset, error) {
	if err := ctx.Err(); err != nil {
		return cachedReleaseAsset{}, err
	}
	if item, ok := h.releaseCache.get(asset.path, method, time.Now()); ok {
		return item, nil
	}
	if !h.releaseLimiter.beginFetch(time.Now()) {
		return cachedReleaseAsset{status: http.StatusTooManyRequests}, nil
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, releaseFetchWait)
	defer waitCancel()
	if err := h.releaseLimiter.waitSlot(waitCtx); err != nil {
		h.releaseLimiter.refundFetch()
		if ctx.Err() != nil {
			return cachedReleaseAsset{}, ctx.Err()
		}
		return cachedReleaseAsset{status: http.StatusTooManyRequests}, nil
	}
	defer h.releaseLimiter.releaseSlot()

	if err := ctx.Err(); err != nil {
		h.releaseLimiter.refundFetch()
		return cachedReleaseAsset{}, err
	}

	fetchCtx, fetchCancel := context.WithTimeout(ctx, releaseFetchTimeout)
	defer fetchCancel()
	return h.fetchReleaseAsset(fetchCtx, asset, method)
}

func (h *AssetHandler) fetchReleaseAsset(ctx context.Context, asset releaseAsset, method string) (cachedReleaseAsset, error) {
	req, err := http.NewRequestWithContext(ctx, method, h.assetCDNOrigin+asset.path, nil)
	if err != nil {
		return cachedReleaseAsset{}, err
	}

	resp, err := h.assetCDNClient.Do(req)
	if err != nil {
		return cachedReleaseAsset{}, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	item := cachedReleaseAsset{
		status:        releaseProxyStatus(resp.StatusCode),
		contentLength: releaseContentLength(resp),
		expires:       time.Now().Add(releaseCacheTTL),
	}
	if item.status != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			h.releaseCache.store(asset.path, item)
		}
		return item, nil
	}
	if method == http.MethodHead {
		h.releaseCache.store(asset.path, item)
		return item, nil
	}

	limit := h.releaseMaxBytes
	if limit <= 0 {
		limit = releaseAssetMaxBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return cachedReleaseAsset{}, err
	}
	if int64(len(body)) > limit {
		return cachedReleaseAsset{}, fmt.Errorf("release asset exceeds %d bytes", limit)
	}

	item.body = body
	item.hasBody = true
	item.contentLength = int64(len(body))
	h.releaseCache.store(asset.path, item)
	return item, nil
}

func releaseProxyStatus(status int) int {
	switch status {
	case http.StatusOK, http.StatusNotFound:
		return status
	default:
		return http.StatusBadGateway
	}
}

func releaseContentLength(resp *http.Response) int64 {
	if resp.ContentLength >= 0 {
		return resp.ContentLength
	}

	parsed, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil || parsed < 0 {
		return -1
	}

	return parsed
}

func writeReleaseAsset(w http.ResponseWriter, r *http.Request, asset releaseAsset, item cachedReleaseAsset) {
	if item.status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "too many requests", item.status)
		return
	}
	if item.status == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	if item.status != http.StatusOK {
		http.Error(w, "release asset unavailable", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", releaseAssetContentType(asset.name))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if item.contentLength >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(item.contentLength, 10))
	}
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead || !item.hasBody {
		return
	}
	if _, err := w.Write(item.body); err != nil {
		log.Warnf("release asset write failed: %v", err)
	}
}

func releaseAssetContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".wasm":
		return "application/wasm"
	default:
		return "application/octet-stream"
	}
}
