package web

import (
	"fmt"
	"net/http"
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

	Router := r.Methods(http.MethodGet).Subrouter()
	//
	// serve static files from pkg/web/assets/dist
	//
	Router.PathPrefix("/").Handler(http.StripPrefix(s.BasePath, http.FileServer(http.FS(assets.EmbeddedAssets))))
	Router.Use(additionalMiddlewares...)

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
