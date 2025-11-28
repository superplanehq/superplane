package web

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AssetHandler serves static files from the assets filesystem
// and handles SPA routing by serving index.html for non-asset routes
type AssetHandler struct {
	assets       http.FileSystem
	basePath     string
	indexContent []byte
	indexModTime time.Time
}

// NewAssetHandler creates a new AssetHandler with the given file system
func NewAssetHandler(assets http.FileSystem, basePath string) http.Handler {
	indexContent, indexModTime := loadIndexHTML(assets)

	return &AssetHandler{
		assets:       assets,
		basePath:     basePath,
		indexContent: indexContent,
		indexModTime: indexModTime,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle /app/assets/* paths
	if h.isAssetPath(r.URL.Path) {
		h.serveAsset(w, r)
		return
	}

	// For all other paths, serve index.html for SPA routing
	h.serveIndex(w, r)
}

// isAssetPath checks if the request is for an asset file
func (h *AssetHandler) isAssetPath(path string) bool {
	if strings.HasPrefix(path, h.basePath+"/assets") {
		return true
	}

	// Also serve common root-level assets produced by Vite's public/ directory
	// e.g., /favicon.ico, /robots.txt, /manifest.webmanifest
	root := strings.TrimPrefix(path, h.basePath)
	switch root {
	case "/favicon.ico", "/robots.txt", "/manifest.webmanifest":
		return true
	}
	return false
}

// serveAsset serves static files from the assets directory
func (h *AssetHandler) serveAsset(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, h.basePath)

	f, err := h.assets.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	if fi, _ := f.Stat(); fi != nil && !fi.IsDir() {
		if mimeType := mime.TypeByExtension(filepath.Ext(path)); mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
	} else {
		http.NotFound(w, r)
	}
}

// serveIndex serves the index.html file for SPA routing
func (h *AssetHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if h.indexContent == nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	reader := bytes.NewReader(h.indexContent)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeContent(w, r, "index.html", h.indexModTime, reader)
}

func loadIndexHTML(assets http.FileSystem) ([]byte, time.Time) {
	indexFile, err := assets.Open("index.html")
	if err != nil {
		return nil, time.Time{}
	}
	defer indexFile.Close()

	data, err := io.ReadAll(indexFile)
	if err != nil {
		return nil, time.Time{}
	}

	if fi, err := indexFile.Stat(); err == nil {
		return injectSentryConfig(data), fi.ModTime()
	}

	return injectSentryConfig(data), time.Now()
}

func injectSentryConfig(indexHTML []byte) []byte {
	dsn := os.Getenv("SENTRY_DSN")
	env := os.Getenv("SENTRY_ENVIRONMENT")

	if dsn == "" && env == "" {
		return indexHTML
	}

	var scriptBuilder strings.Builder
	scriptBuilder.WriteString("<script>")

	if dsn != "" {
		scriptBuilder.WriteString("window.SUPERPLANE_SENTRY_DSN=")
		scriptBuilder.WriteString(strconv.Quote(dsn))
		scriptBuilder.WriteString(";")
	}

	if env != "" {
		scriptBuilder.WriteString("window.SUPERPLANE_SENTRY_ENVIRONMENT=")
		scriptBuilder.WriteString(strconv.Quote(env))
		scriptBuilder.WriteString(";")
	}

	scriptBuilder.WriteString("</script>")
	scriptTag := scriptBuilder.String()

	headClose := []byte("</head>")
	if idx := bytes.Index(indexHTML, headClose); idx != -1 {
		var buf bytes.Buffer
		buf.Write(indexHTML[:idx])
		buf.WriteString(scriptTag)
		buf.Write(indexHTML[idx:])
		return buf.Bytes()
	}

	return indexHTML
}
