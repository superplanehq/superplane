package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/web/assets"
)

type Server struct {
	httpServer     *http.Server
	handlerTimeout time.Duration
	Router         *mux.Router
	BasePath       string
}

func NewServer(basePath string, middlewares ...mux.MiddlewareFunc) (*Server, error) {
	server := &Server{
		handlerTimeout: 15 * time.Second,
		BasePath:       basePath,
	}
	server.InitRouter(middlewares...)
	return server, nil
}

func (s *Server) InitRouter(additionalMiddlewares ...mux.MiddlewareFunc) {
	r := mux.NewRouter().StrictSlash(true)

	// Check if we're in development mode
	if os.Getenv("ENV") == "dev" {
		log.Info("Running in development mode - proxying to Vite dev server")
		s.setupDevProxy(r)
	} else {
		log.Info("Running in production mode - serving static assets")
		Router := r.Methods(http.MethodGet).Subrouter()
		//
		// serve static files from pkg/web/assets/dist
		//
		Router.PathPrefix(s.BasePath + "/").HandlerFunc(s.ServeAssets)
		Router.Use(additionalMiddlewares...)
	}

	s.Router = r
}

func (s *Server) Serve(host string, port int) error {
	log.Infof("Starting server at %s:%d", host, port)
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler: http.TimeoutHandler(
			handlers.LoggingHandler(os.Stdout, s.Router),
			s.handlerTimeout,
			"request timed out",
		),
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Close() {
	if err := s.httpServer.Close(); err != nil {
		log.Errorf("Error closing server: %v", err)
	}
}

func (s *Server) ServeAssets(w http.ResponseWriter, r *http.Request) {
	log.Infof("Serving assets for %s but stripping prefix %s", r.URL.Path, s.BasePath)
	http.StripPrefix(s.BasePath, http.FileServer(http.FS(assets.EmbeddedAssets))).ServeHTTP(w, r)
}


// setupDevProxy configures a simple reverse proxy to the Vite development server
func (s *Server) setupDevProxy(r *mux.Router) {
	// Configure the target Vite dev server URL
	target, err := url.Parse("http://localhost:5173")
	if err != nil {
		log.Fatalf("Error parsing Vite dev server URL: %v", err)
	}

	// Create a simple proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Simple director modification
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Store original path for logging
		originalPath := req.URL.Path

		// Execute original director function
		origDirector(req)

		// Set host header to target
		req.Host = target.Host

		log.Infof("Proxying: %s → %s", originalPath, req.URL.Path)
	}

	// Create a simple handler that skips API requests
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip API requests
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			http.NotFound(w, r)
			return
		}

		// Forward everything else to Vite
		proxy.ServeHTTP(w, r)
	})

	// Mount the handler to all paths
	r.PathPrefix("/").Handler(proxyHandler)

	// Also handle base path if it's not root
	if s.BasePath != "/" {
		r.PathPrefix(s.BasePath).Handler(proxyHandler)
	}
}