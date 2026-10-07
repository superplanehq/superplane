package web

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const DefaultAssetCDNOrigin = "https://assets.superplane.com"

var (
	releaseSHA  = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	releaseFile = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

type releaseAssetProxy struct {
	origin *url.URL
	client *http.Client
}

func AssetCDNOrigin() string {
	origin := strings.TrimSpace(os.Getenv("ASSET_CDN_ORIGIN"))
	if origin == "" {
		return DefaultAssetCDNOrigin
	}
	return origin
}

func NewReleaseAssetProxy(origin string) (http.Handler, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return nil, fmt.Errorf("asset CDN origin %q is invalid", origin)
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return &releaseAssetProxy{
		origin: parsed,
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (p *releaseAssetProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isReleaseAssetPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	upstream := *p.origin
	upstream.Path = r.URL.Path
	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstream.String(), nil)
	if err != nil {
		log.WithError(err).Warn("build release asset request")
		http.Error(w, "asset unavailable", http.StatusBadGateway)
		return
	}

	resp, err := p.client.Do(req)
	if err != nil {
		log.WithError(err).Warn("fetch release asset")
		http.Error(w, "asset unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		http.Error(w, "asset unavailable", http.StatusBadGateway)
		return
	}

	copyReleaseAssetHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, resp.Body)
}

func isReleaseAssetPath(path string) bool {
	if strings.Contains(path, "..") || strings.Contains(path, "\\") {
		return false
	}
	rest, ok := strings.CutPrefix(path, "/releases/")
	if !ok {
		return false
	}
	sha, file, ok := strings.Cut(rest, "/assets/")
	if !ok || file == "" || strings.Contains(file, "/") {
		return false
	}
	return releaseSHA.MatchString(sha) && releaseFile.MatchString(file)
}

func copyReleaseAssetHeaders(w http.ResponseWriter, resp *http.Response) {
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	}
	if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	} else if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	if etag := resp.Header.Get("ETag"); etag != "" {
		w.Header().Set("ETag", etag)
	}
}
