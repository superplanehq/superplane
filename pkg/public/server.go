package public

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbSup "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/web"
	"github.com/superplanehq/superplane/pkg/web/assets"
	grpcLib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// Event payload can be up to 64k in size
	MaxEventSize = 64 * 1024

	// The size of the stage execution outputs can be up to 4k
	MaxExecutionOutputsSize = 4 * 1024
)

type Server struct {
	httpServer            *http.Server
	encryptor             crypto.Encryptor
	jwt                   *jwt.Signer
	timeoutHandlerTimeout time.Duration
	upgrader              *websocket.Upgrader
	Router                *mux.Router
	BasePath              string
	wsHub                 *ws.Hub
}

// WebsocketHub returns the websocket hub for this server
func (s *Server) WebsocketHub() *ws.Hub {
	return s.wsHub
}

func NewServer(encryptor crypto.Encryptor, jwtSigner *jwt.Signer, basePath string, middlewares ...mux.MiddlewareFunc) (*Server, error) {
	// Create and initialize a new WebSocket hub
	wsHub := ws.NewHub()

	server := &Server{
		timeoutHandlerTimeout: 15 * time.Second,
		encryptor:             encryptor,
		jwt:                   jwtSigner,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all connections - you may want to restrict this in production
				// TODO: implement origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		BasePath: basePath,
		wsHub:    wsHub,
	}

	server.timeoutHandlerTimeout = 15 * time.Second
	server.InitRouter(middlewares...)
	return server, nil
}

func (s *Server) RegisterGRPCGateway(grpcServerAddr string) error {
	ctx := context.Background()

	grpcGatewayMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(runtime.DefaultHeaderMatcher),
	)

	opts := []grpcLib.DialOption{grpcLib.WithTransportCredentials(insecure.NewCredentials())}

	err := pbSup.RegisterSuperplaneHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbAuth.RegisterAuthorizationHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = grpcGatewayMux.HandlePath("GET", "/api/v1/canvases/is-alive", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.WriteHeader(http.StatusOK)
	})

	if err != nil {
		return err
	}

	// This is necessary because it is not possible to use the current router with
	// runtime Mux. Runtime mux has no specification for the added paths and it the only
	// supported tool for grpc-gateway.
	s.Router.PathPrefix("/api/v1/canvases").Handler(s.grpcGatewayHandler(grpcGatewayMux))
	s.Router.PathPrefix("/api/v1/authorization").Handler(s.grpcGatewayHandler(grpcGatewayMux))

	return nil
}

func (s *Server) grpcGatewayHandler(grpcGatewayMux *runtime.ServeMux) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		grpcGatewayMux.ServeHTTP(w, r2)
	})
}

// RegisterOpenAPIHandler adds handlers to serve the OpenAPI specification and Swagger UI
func (s *Server) RegisterOpenAPIHandler() {
	swaggerFilesPath := os.Getenv("SWAGGER_BASE_PATH")
	if swaggerFilesPath == "" {
		log.Errorf("SWAGGER_BASE_PATH is not set")
		return
	}

	if _, err := os.Stat(swaggerFilesPath); os.IsNotExist(err) {
		log.Errorf("API documentation directory %s does not exist", swaggerFilesPath)
		return
	}

	s.Router.HandleFunc(s.BasePath+"/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, swaggerFilesPath+"/swagger-ui.html")
	})

	s.Router.HandleFunc(s.BasePath+"/docs/superplane.swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, swaggerFilesPath+"/superplane.swagger.json")
	})

	log.Infof("OpenAPI specification available at %s", swaggerFilesPath)
	log.Infof("Swagger UI available at %s", swaggerFilesPath)
	log.Infof("Raw API JSON available at %s", swaggerFilesPath+"/superplane.swagger.json")
}

func (s *Server) RegisterWebRoutes(webBasePath string) {
	// The web app routes are registered on the main router
	log.Infof("Registering web routes with base path: %s", webBasePath)

	s.Router.HandleFunc("/ws/{canvasId}", s.handleWebSocket)

	// Check if we're in development mode
	if os.Getenv("APP_ENV") == "development" {
		log.Info("Running in development mode - proxying to Vite dev server for web app")
		s.setupDevProxy(webBasePath)
	} else {
		log.Info("Running in production mode - serving static web assets")

		handler := web.NewAssetHandler(http.FS(assets.EmbeddedAssets), webBasePath)
		s.Router.PathPrefix(webBasePath).Handler(handler)

		s.Router.HandleFunc(webBasePath, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == webBasePath {
				http.Redirect(w, r, webBasePath+"/", http.StatusMovedPermanently)
				return
			}
			handler.ServeHTTP(w, r)
		})
	}
}

func (s *Server) InitRouter(additionalMiddlewares ...mux.MiddlewareFunc) {
	r := mux.NewRouter().StrictSlash(true)
	r.Use(middleware.LoggingMiddleware(log.StandardLogger()))

	//
	// Authenticated and validated routes.
	//
	authenticatedRoute := r.Methods(http.MethodPost).Subrouter()

	authenticatedRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}/github", s.HandleGithubWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	authenticatedRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}/semaphore", s.HandleSemaphoreWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	authenticatedRoute.
		HandleFunc(s.BasePath+"/outputs", s.HandleExecutionOutputs).
		Headers("Content-Type", "application/json").
		Methods("POST")

	authenticatedRoute.Use(additionalMiddlewares...)

	//
	// No authentication of any kind, just a health endpoint
	//
	unauthenticatedRoute := r.Methods(http.MethodGet).Subrouter()
	unauthenticatedRoute.HandleFunc("/", s.HealthCheck).Methods("GET")

	s.Router = r
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) Serve(host string, port int) error {
	log.Infof("Starting server at %s:%d", host, port)

	// Start the WebSocket hub
	log.Info("Starting WebSocket hub")
	s.wsHub.Run()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      s.Router,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Close() {
	if err := s.httpServer.Close(); err != nil {
		log.Errorf("Error closing server: %v", err)
	}
}

type OutputsRequest struct {
	ExecutionID string         `json:"execution_id"`
	Outputs     map[string]any `json:"outputs"`
}

func (s *Server) HandleExecutionOutputs(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	headerParts := strings.Split(authHeader, "Bearer ")
	if len(headerParts) != 2 {
		http.Error(w, "Malformed Authorization header", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxExecutionOutputsSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(
				w,
				fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxExecutionOutputsSize),
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var req OutputsRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	executionID, err := uuid.Parse(req.ExecutionID)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	token := headerParts[1]
	err = s.jwt.Validate(token, req.ExecutionID)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	execution, err := models.FindExecutionByID(executionID)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	stage, err := models.FindStageByID(execution.StageID.String())
	if err != nil {
		http.Error(w, "error finding stage", http.StatusInternalServerError)
		return
	}

	outputs, err := s.parseExecutionOutputs(stage, req.Outputs)
	if err != nil {
		http.Error(w, "Error parsing outputs", http.StatusBadRequest)
		return
	}

	err = execution.UpdateOutputs(outputs)
	if err != nil {
		http.Error(w, "Error updating outputs", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) parseExecutionOutputs(stage *models.Stage, outputs map[string]any) (map[string]any, error) {
	//
	// We ignore outputs that were sent but are not defined in the stage.
	//
	for k := range outputs {
		if !stage.HasOutputDefinition(k) {
			delete(outputs, k)
		}
	}

	return outputs, nil
}

func (s *Server) HandleGithubWebhook(w http.ResponseWriter, r *http.Request) {
	//
	// Any verification that happens here must be quick
	// so we always respond with a 200 OK to the event origin.
	// All the event processing happen on the workers.
	//

	vars := mux.Vars(r)
	sourceIDFromRequest := vars["sourceID"]
	sourceID, err := uuid.Parse(sourceIDFromRequest)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Missing X-Hub-Signature-256 header", http.StatusBadRequest)
		return
	}

	// TODO: we don't have the canvas ID here.
	// We could put it in the path, but then the path will become quite big.
	// For now, just organization/source IDs are enough for us.
	source, err := models.FindEventSource(sourceID)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	//
	// Only read up to the maximum event size we allow,
	// and only proceed if payload is below that.
	//
	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(
				w,
				fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxEventSize),
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	headers, err := parseHeaders(&r.Header)
	if err != nil {
		http.Error(w, "Error parsing headers", http.StatusBadRequest)
		return
	}

	key, err := s.encryptor.Decrypt(r.Context(), source.Key, []byte(source.Name))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	signature = strings.Replace(signature, "sha256=", "", 1)
	if err := crypto.VerifySignature(key, body, signature); err != nil {
		log.Errorf("Invalid signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	//
	// Here, we know the event is for a valid organization/source,
	// and comes from GitHub, so we just want to save it and give a response back.
	//
	if _, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, body, headers); err != nil {
		http.Error(w, "Error receiving event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleSemaphoreWebhook(w http.ResponseWriter, r *http.Request) {
	//
	// Any verification that happens here must be quick
	// so we always respond with a 200 OK to the event origin.
	// All the event processing happen on the workers.
	//

	vars := mux.Vars(r)
	sourceIDFromRequest := vars["sourceID"]
	sourceID, err := uuid.Parse(sourceIDFromRequest)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	signature := r.Header.Get("X-Semaphore-Signature-256")
	if signature == "" {
		http.Error(w, "Missing X-Semaphore-Signature-256 header", http.StatusBadRequest)
		return
	}

	source, err := models.FindEventSource(sourceID)
	if err != nil {
		http.Error(w, "source ID not found", http.StatusNotFound)
		return
	}

	//
	// Only read up to the maximum event size we allow,
	// and only proceed if payload is below that.
	//
	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(
				w,
				fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxEventSize),
				http.StatusRequestEntityTooLarge,
			)

			return
		}

		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	headers, err := parseHeaders(&r.Header)
	if err != nil {
		http.Error(w, "Error parsing headers", http.StatusBadRequest)
		return
	}

	key, err := s.encryptor.Decrypt(r.Context(), source.Key, []byte(source.Name))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	signature = strings.Replace(signature, "sha256=", "", 1)

	if err := crypto.VerifySignature(key, body, signature); err != nil {
		log.Errorf("Invalid signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	//
	// Here, we know the event is for a valid organization/source,
	// and comes from Semaphore, so we just want to save it and give a response back.
	//
	if _, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, body, headers); err != nil {
		http.Error(w, "Error receiving event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseHeaders(headers *http.Header) ([]byte, error) {
	parsedHeaders := make(map[string]string, len(*headers))
	for key, value := range *headers {
		parsedHeaders[key] = value[0]
	}

	return json.Marshal(parsedHeaders)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Infof("New WebSocket connection from %s", r.RemoteAddr)

	// Extract the canvasId from the URL path variables
	vars := mux.Vars(r)
	canvasID := vars["canvasId"]
	log.Infof("WebSocket connection for canvas ID: %s", canvasID)

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		log.Infof("Failed to upgrade to WebSocket: %v", err)
		return
	}

	client := s.wsHub.NewClient(ws, canvasID)
	log.Infof("WebSocket client registered with hub")
	// Wait for the client to disconnect
	<-client.Done
}

// setupDevProxy configures a simple reverse proxy to the Vite development server
func (s *Server) setupDevProxy(webBasePath string) {
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

	// Create a handler for web app routes
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip API requests
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			return
		}

		// Forward to Vite
		proxy.ServeHTTP(w, r)
	})

	// Mount the handler to the web app path
	s.Router.PathPrefix(webBasePath).Handler(proxyHandler)
}
