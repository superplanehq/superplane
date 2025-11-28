package web

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// AssetHandler serves static files from the assets filesystem
// and handles SPA routing by serving index.html for non-asset routes.
type AssetHandler struct {
	assets       http.FileSystem
	basePath     string
	indexContent []byte
	indexModTime time.Time
}

// NewAssetHandler creates a new AssetHandler with the given file system.
func NewAssetHandler(assets http.FileSystem, basePath string) http.Handler {
	indexContent, indexModTime := loadIndexContent(assets)

	return &AssetHandler{
		assets:       assets,
		basePath:     basePath,
		indexContent: indexContent,
		indexModTime: indexModTime,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle /assets/* paths
	if h.isAssetPath(r.URL.Path) {
		h.serveAsset(w, r)
		return
	}

	// For all other paths, serve index.html for SPA routing
	h.serveIndex(w, r)
}

// isAssetPath checks if the request is for an asset file.
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

// serveAsset serves static files from the assets directory.
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

// serveIndex serves the index.html file for SPA routing.
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

// loadIndexContent loads and renders index.html once at startup,
// applying the shared template rendering logic and caching the result.
func loadIndexContent(assets http.FileSystem) ([]byte, time.Time) {
	indexFile, err := assets.Open("index.html")
	if err != nil {
		log.Fatalf("failed to open index.html from assets: %v", err)
	}
	defer indexFile.Close()

	data, err := io.ReadAll(indexFile)
	if err != nil {
		log.Fatalf("failed to read index.html from assets: %v", err)
	}

	rendered, err := RenderIndexTemplate(data)
	if err != nil {
		log.Fatalf("failed to render index.html template: %v", err)
	}

	if fi, err := indexFile.Stat(); err == nil {
		return rendered, fi.ModTime()
	}

	return rendered, time.Now()
}
