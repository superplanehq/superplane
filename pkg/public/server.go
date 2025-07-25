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
	"github.com/superplanehq/superplane/pkg/authentication"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pbSup "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbIntegrations "github.com/superplanehq/superplane/pkg/protos/integrations"
	pbOrg "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecret "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
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
	authHandler           *authentication.Handler
	isDev                 bool
}

// WebsocketHub returns the websocket hub for this server
func (s *Server) WebsocketHub() *ws.Hub {
	return s.wsHub
}

func NewServer(encryptor crypto.Encryptor, jwtSigner *jwt.Signer, basePath string, appEnv string, middlewares ...mux.MiddlewareFunc) (*Server, error) { // Create and initialize a new WebSocket hub
	wsHub := ws.NewHub()

	authHandler := authentication.NewHandler(jwtSigner, encryptor, appEnv)

	// Initialize OAuth providers from environment variables
	providers := getOAuthProviders()
	authHandler.InitializeProviders(providers)

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
		BasePath:    basePath,
		wsHub:       wsHub,
		authHandler: authHandler,
		isDev:       appEnv == "development",
	}

	server.timeoutHandlerTimeout = 15 * time.Second
	server.InitRouter(middlewares...)
	return server, nil
}

func getOAuthProviders() map[string]authentication.ProviderConfig {
	baseURL := getBaseURL()
	providers := make(map[string]authentication.ProviderConfig)

	// GitHub
	if githubKey := os.Getenv("GITHUB_CLIENT_ID"); githubKey != "" {
		if githubSecret := os.Getenv("GITHUB_CLIENT_SECRET"); githubSecret != "" {
			providers["github"] = authentication.ProviderConfig{
				Key:         githubKey,
				Secret:      githubSecret,
				CallbackURL: fmt.Sprintf("%s/auth/github/callback", baseURL),
			}
		}
	}

	// ...Other providers must be added here
	return providers
}

func (s *Server) RegisterGRPCGateway(grpcServerAddr string) error {
	ctx := context.Background()

	grpcGatewayMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(headersMatcher),
	)

	opts := []grpcLib.DialOption{grpcLib.WithTransportCredentials(insecure.NewCredentials())}

	err := pbSup.RegisterSuperplaneHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbUsers.RegisterUsersHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbGroups.RegisterGroupsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbRoles.RegisterRolesHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbOrg.RegisterOrganizationsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbIntegrations.RegisterIntegrationsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbSecret.RegisterSecretsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	// Public health check
	s.Router.HandleFunc("/api/v1/canvases/is-alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	// Protect the gRPC gateway routes with authentication
	protectedGRPCHandler := s.authHandler.Middleware(
		s.stripUserIDHeaderHandler(s.grpcGatewayHandler(grpcGatewayMux)),
	)

	s.Router.PathPrefix("/api/v1/users").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/groups").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/roles").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/canvases").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/organizations").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/integrations").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/secrets").Handler(protectedGRPCHandler)

	return nil
}

// stripUserIDHeaderHandler removes the X-User-Id header from the request before we set it manually
func (s *Server) stripUserIDHeaderHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-User-Id")
		r.Header.Del("x-user-id")
		next.ServeHTTP(w, r)
	})
}

func headersMatcher(key string) (string, bool) {
	switch key {
	case "X-User-Id":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func (s *Server) grpcGatewayHandler(grpcGatewayMux *runtime.ServeMux) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := authentication.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not found in context", http.StatusUnauthorized)
			return
		}

		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.Header.Set("x-user-id", user.ID.String())
		grpcGatewayMux.ServeHTTP(w, r2.WithContext(r.Context()))
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
	log.Infof("Registering web routes with base path: %s", webBasePath)

	// WebSocket endpoint - protected by authentication
	protectedWSHandler := s.authHandler.Middleware(http.HandlerFunc(s.handleWebSocket))
	s.Router.Handle("/ws/{canvasId}", protectedWSHandler)

	// Check if we're in development mode
	if s.isDev {
		log.Info("Running in development mode - proxying to Vite dev server for web app")
		s.setupDevProxy(webBasePath)
	} else {
		log.Info("Running in production mode - serving static web assets")

		handler := web.NewAssetHandler(http.FS(assets.EmbeddedAssets), webBasePath)

		// Protect the main web application with authentication
		protectedWebHandler := s.authHandler.Middleware(handler)
		s.Router.PathPrefix(webBasePath).Handler(protectedWebHandler)

		s.Router.HandleFunc(webBasePath, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == webBasePath {
				http.Redirect(w, r, webBasePath+"/", http.StatusMovedPermanently)
				return
			}
			protectedWebHandler.ServeHTTP(w, r)
		})
	}
}

func (s *Server) InitRouter(additionalMiddlewares ...mux.MiddlewareFunc) {
	r := mux.NewRouter().StrictSlash(true)
	r.Use(middleware.LoggingMiddleware(log.StandardLogger()))

	// Register authentication routes (no auth required)
	s.authHandler.RegisterRoutes(r)

	//
	// Public routes (no authentication required)
	//
	publicRoute := r.Methods(http.MethodGet, http.MethodPost).Subrouter()

	// Health check
	publicRoute.HandleFunc("/", s.HealthCheck).Methods("GET")

	//
	// Webhook endpoints (they have their own authentication)
	//
	// Any verification that happens here must be quick
	// so we always respond with a 200 OK to the event origin.
	// All the event processing happen on the workers.
	//
	//
	publicRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}/github", s.HandleGithubWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	publicRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}/semaphore", s.HandleSemaphoreWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	publicRoute.
		HandleFunc(s.BasePath+"/outputs", s.HandleExecutionOutputs).
		Headers("Content-Type", "application/json").
		Methods("POST")

	//
	// Protected routes (authentication required)
	//
	protectedRoute := r.NewRoute().Subrouter()
	protectedRoute.Use(s.authHandler.Middleware)

	// Add protected API routes here
	protectedRoute.HandleFunc("/api/v1/user/profile", s.handleUserProfile).Methods("GET")
	protectedRoute.HandleFunc("/api/v1/user/account-providers", s.handleUserAccountProviders).Methods("GET")

	// Apply additional middlewares
	for _, middleware := range additionalMiddlewares {
		publicRoute.Use(middleware)
		protectedRoute.Use(middleware)
	}

	s.Router = r
}

func (s *Server) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := authentication.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	accountProviders, err := user.GetAccountProviders()
	if err != nil {
		log.Errorf("Error getting account providers: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var email, avatarURL string
	if len(accountProviders) > 0 {
		email = accountProviders[0].Email
		avatarURL = accountProviders[0].AvatarURL

		// Fallback to user name if no email from provider
		if email == "" && user.Name != "" {
			email = user.Name
		}
	} else {
		if user.Name != "" {
			email = user.Name
		}
	}

	safeUser := UserProfileResponse{
		ID:               user.ID.String(),
		Email:            email,
		Name:             user.Name,
		AvatarURL:        avatarURL,
		CreatedAt:        user.CreatedAt,
		AccountProviders: accountProviders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safeUser)
}

func (s *Server) handleUserAccountProviders(w http.ResponseWriter, r *http.Request) {
	user, ok := authentication.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	accountProviders, err := user.GetAccountProviders()
	if err != nil {
		log.Errorf("Error getting repo host accounts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accountProviders)
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

type UserProfileResponse struct {
	ID               string                   `json:"id"`
	Email            string                   `json:"email"`
	Name             string                   `json:"name"`
	AvatarURL        string                   `json:"avatar_url"`
	CreatedAt        time.Time                `json:"created_at"`
	AccountProviders []models.AccountProvider `json:"account_providers,omitempty"`
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

	s.handleWebhook(w, r, source, signature)
}

func (s *Server) HandleSemaphoreWebhook(w http.ResponseWriter, r *http.Request) {
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

	s.handleWebhook(w, r, source, signature)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request, source *models.EventSource, signature string) {
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

	key, err := source.GetDecryptedKey(r.Context(), s.encryptor)
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

	_, ok := authentication.GetUserFromContext(r.Context())
	if !ok {
		log.Error("WebSocket connection without authenticated user")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	canvasID := vars["canvasId"]

	// TODO: implement access check once authorization is implemented

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		log.Infof("Failed to upgrade to WebSocket: %v", err)
		return
	}

	client := s.wsHub.NewClient(ws, canvasID)

	<-client.Done
}

// setupDevProxy configures a simple reverse proxy to the Vite development server
func (s *Server) setupDevProxy(webBasePath string) {
	target, err := url.Parse("http://localhost:5173")
	if err != nil {
		log.Fatalf("Error parsing Vite dev server URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalPath := req.URL.Path

		origDirector(req)

		req.Host = target.Host

		log.Infof("Proxying: %s → %s", originalPath, req.URL.Path)
	}

	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			return
		}

		proxy.ServeHTTP(w, r)
	})

	protectedProxy := s.authHandler.Middleware(proxyHandler)
	s.Router.PathPrefix(webBasePath).Handler(protectedProxy)
}

func getBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8000"
		}
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}
	return baseURL
}
