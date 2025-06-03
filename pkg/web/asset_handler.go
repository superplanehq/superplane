package web

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// AssetHandler serves static files from the assets filesystem
// and handles SPA routing by serving index.html for non-asset routes
type AssetHandler struct {
	assets    http.FileSystem
	basePath  string
	indexFile http.File
}

// NewAssetHandler creates a new AssetHandler with the given file system
func NewAssetHandler(assets http.FileSystem, basePath string) http.Handler {
	// Load index.html once
	indexFile, _ := assets.Open("index.html")

	return &AssetHandler{
		assets:    assets,
		basePath:  basePath,
		indexFile: indexFile,
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
	return strings.HasPrefix(path, h.basePath+"/assets")
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
	if h.indexFile == nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	// Reset and serve index.html
	h.indexFile.Seek(0, 0)
	if fi, _ := h.indexFile.Stat(); fi != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeContent(w, r, "index.html", fi.ModTime(), h.indexFile)
	}
}
