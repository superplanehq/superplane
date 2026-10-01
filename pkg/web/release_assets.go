package web

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const releaseAssetMaxBytes int64 = 32 << 20

var releaseAssetPathPattern = regexp.MustCompile(`^/releases/([0-9a-f]{40})/assets/([A-Za-z0-9._-]{1,200}\.(?:js|mjs|wasm))$`)
var releaseAssetBasePattern = regexp.MustCompile(`^/releases/[0-9a-f]{40}/?$`)

type releaseAsset struct {
	path string
	name string
}

func newAssetCDNClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
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
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, h.assetCDNOrigin+asset.path, nil)
	if err != nil {
		http.Error(w, "release asset unavailable", http.StatusBadGateway)
		return
	}

	resp, err := h.assetCDNClient.Do(req)
	if err != nil {
		log.Warnf("release asset proxy failed: %v", err)
		http.Error(w, "release asset unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "release asset unavailable", http.StatusBadGateway)
		return
	}

	limit := h.releaseMaxBytes
	if limit <= 0 {
		limit = releaseAssetMaxBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(body)) > limit {
		http.Error(w, "release asset unavailable", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", releaseAssetContentType(asset.name))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}

	if _, err := w.Write(body); err != nil {
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
