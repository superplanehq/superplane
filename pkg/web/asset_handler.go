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
    // Load index.html once. http.FileSystem expects paths to start with '/'.
    indexFile, _ := assets.Open("/index.html")
    if indexFile == nil {
        // Fallback for embedded assets where files live under dist/
        indexFile, _ = assets.Open("/dist/index.html")
    }

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
    // Ensure leading slash for http.FileSystem
    if !strings.HasPrefix(path, "/") {
        path = "/" + path
    }

    f, err := h.assets.Open(path)
    if err != nil {
        // Fallback for embedded assets where files live under dist/
        // Avoid double slashes
        if strings.HasPrefix(path, "/") {
            f, err = h.assets.Open("/dist" + path)
        } else {
            f, err = h.assets.Open("/dist/" + path)
        }
    }
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
    f := h.indexFile
    // If not preloaded, try to open dynamically (supports both layouts)
    if f == nil {
        if file, err := h.assets.Open("/index.html"); err == nil {
            f = file
            defer f.Close()
        } else if file, err2 := h.assets.Open("/dist/index.html"); err2 == nil {
            f = file
            defer f.Close()
        }
    }
    if f == nil {
        http.Error(w, "index.html not found", http.StatusInternalServerError)
        return
    }

    // Reset and serve index.html
    f.Seek(0, 0)
    if fi, _ := f.Stat(); fi != nil {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
        http.ServeContent(w, r, "index.html", fi.ModTime(), f)
    }
}
