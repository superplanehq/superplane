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
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/registry"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pbBlueprints "github.com/superplanehq/superplane/pkg/protos/blueprints"
	pbSup "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbIntegrations "github.com/superplanehq/superplane/pkg/protos/integrations"
	pbMe "github.com/superplanehq/superplane/pkg/protos/me"
	pbOrg "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbPrimitives "github.com/superplanehq/superplane/pkg/protos/primitives"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecret "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	pbWorkflows "github.com/superplanehq/superplane/pkg/protos/workflows"
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
	registry              *registry.Registry
	jwt                   *jwt.Signer
	oidcVerifier          *crypto.OIDCVerifier
	authService           authorization.Authorization
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

func NewServer(
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	jwtSigner *jwt.Signer,
	oidcVerifier *crypto.OIDCVerifier,
	basePath string,
	appEnv string,
	authorizationService authorization.Authorization,
	middlewares ...mux.MiddlewareFunc,
) (*Server, error) {

	// Initialize OAuth providers from environment variables
	authHandler := authentication.NewHandler(jwtSigner, encryptor, authorizationService, appEnv)
	providers := getOAuthProviders()
	authHandler.InitializeProviders(providers)

	server := &Server{
		BasePath:              basePath,
		wsHub:                 ws.NewHub(),
		authHandler:           authHandler,
		isDev:                 appEnv == "development",
		timeoutHandlerTimeout: 15 * time.Second,
		encryptor:             encryptor,
		jwt:                   jwtSigner,
		oidcVerifier:          oidcVerifier,
		registry:              registry,
		authService:           authorizationService,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all connections - you may want to restrict this in production
				// TODO: implement origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
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

	err = pbMe.RegisterMeHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbPrimitives.RegisterPrimitivesHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbBlueprints.RegisterBlueprintsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbWorkflows.RegisterWorkflowsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	// Public health check
	s.Router.HandleFunc("/api/v1/canvases/is-alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	// Protect the gRPC gateway routes with organization authentication
	orgAuthMiddleware := middleware.OrganizationAuthMiddleware(s.jwt)
	protectedGRPCHandler := orgAuthMiddleware(s.grpcGatewayHandler(grpcGatewayMux))

	s.Router.PathPrefix("/api/v1/users").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/groups").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/roles").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/canvases").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/organizations").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/integrations").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/secrets").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/me").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/primitives").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/blueprints").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/workflows").Handler(protectedGRPCHandler)

	return nil
}

func headersMatcher(key string) (string, bool) {
	switch key {
	case "X-User-Id", "X-Organization-Id":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func (s *Server) grpcGatewayHandler(grpcGatewayMux *runtime.ServeMux) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not found in context", http.StatusUnauthorized)
			return
		}

		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.Header.Set("x-User-id", user.ID.String())
		r2.Header.Set("x-Organization-id", user.OrganizationID.String())
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

	// WebSocket endpoint - protected by organization scoped authentication
	s.Router.Handle(
		"/ws/{canvasId}",
		middleware.OrganizationAuthMiddleware(s.jwt).
			Middleware(http.HandlerFunc(s.handleWebSocket)),
	)

	//
	// In development mode, we proxy to the Vite dev server.
	//
	if s.isDev {
		log.Info("Running in development mode - proxying to Vite dev server for web app")
		s.setupDevProxy(webBasePath)
		return
	}

	log.Info("Running in production mode - serving static web assets")

	handler := middleware.AccountAuthMiddleware(s.jwt).
		Middleware(
			web.NewAssetHandler(http.FS(assets.EmbeddedAssets), webBasePath),
		)

	s.Router.PathPrefix(webBasePath).Handler(handler)

	s.Router.HandleFunc(webBasePath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == webBasePath {
			http.Redirect(w, r, webBasePath+"/", http.StatusMovedPermanently)
			return
		}
		handler.ServeHTTP(w, r)
	})
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
	publicRoute.HandleFunc("/health", s.HealthCheck).Methods("GET")

	//
	// Webhook endpoints for integrations (they have their own authentication).
	//
	// Any verification that happens here must be quick
	// so we always respond with a 200 OK to the event origin.
	// All the event processing happen on the workers.
	//
	publicRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}/{integrationName}", s.HandleIntegrationWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	//
	// Webhook endpoint for custom event sources that do not use integration.
	//
	publicRoute.
		HandleFunc(s.BasePath+"/sources/{sourceID}", s.HandleCustomWebhook).
		Headers("Content-Type", "application/json").
		Methods("POST")

	//
	// Endpoint for receiving execution outputs from execution resources.
	//
	publicRoute.
		HandleFunc(s.BasePath+"/outputs", s.HandleExecutionOutputs).
		Headers("Content-Type", "application/json").
		Methods("POST")

	// Account-based endpoints (use account session, not organization context)
	accountRoute := r.NewRoute().Subrouter()
	accountRoute.Use(middleware.AccountAuthMiddleware(s.jwt))
	accountRoute.HandleFunc("/account", s.getAccount).Methods("GET")
	accountRoute.HandleFunc("/organizations", s.listAccountOrganizations).Methods("GET")
	accountRoute.HandleFunc("/organizations", s.createOrganization).Methods("POST")

	// Apply additional middlewares
	for _, middleware := range additionalMiddlewares {
		publicRoute.Use(middleware)
	}

	s.Router = r
}

type OrganizationCreationRequest struct {
	Name string `json:"name"`
}

func (s *Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var req OrganizationCreationRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	//
	// TODO: the organization creation should be in a transaction
	// Create the organization and set up roles for it.
	//
	organization, err := models.CreateOrganization(req.Name, "")
	if err != nil {
		log.Errorf("Error creating organization: %v", err)
		http.Error(w, "Failed to create organization", http.StatusInternalServerError)
		return
	}

	err = s.authService.SetupOrganizationRoles(organization.ID.String())
	if err != nil {
		log.Errorf("Error setting up organization roles for %s: %v", organization.Name, err)
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to set up organization roles", http.StatusInternalServerError)
		return
	}

	//
	// Create the owner user for it
	//
	user, err := models.CreateUser(organization.ID, account.ID, account.Email, account.Name)
	if err != nil {
		log.Errorf("Error creating user for new organization: %v", err)
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to create user account", http.StatusInternalServerError)
		return
	}

	err = s.authService.CreateOrganizationOwner(user.ID.String(), organization.ID.String())
	if err != nil {
		log.Errorf("Error creating organization owner for %s: %v", organization.Name, err)
		models.HardDeleteOrganization(organization.ID.String())
		http.Error(w, "Failed to create organization owner", http.StatusInternalServerError)
		return
	}

	response := map[string]any{}
	response["id"] = organization.ID.String()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type AccountResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	providers, err := account.GetAccountProviders()
	if err != nil {
		log.Errorf("Error getting account providers for %s: %v", account.Email, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	accountResponse := AccountResponse{
		ID:        account.ID.String(),
		Name:      account.Name,
		Email:     account.Email,
		AvatarURL: getAvatarURL(providers),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accountResponse)
}

func (s *Server) listAccountOrganizations(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	type Organization struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	organizations, err := models.FindOrganizationsForAccount(account.Email)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := []Organization{}
	for _, organization := range organizations {
		response = append(response, Organization{
			ID:          organization.ID.String(),
			Name:        organization.Name,
			Description: organization.Description,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

func (s *Server) authenticateExecution(w http.ResponseWriter, r *http.Request, req *ExecutionOutputRequest) *models.StageExecution {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	headerParts := strings.Split(authHeader, "Bearer ")
	if len(headerParts) != 2 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	token := headerParts[1]
	ID, err := uuid.Parse(req.ExecutionID)
	if err != nil {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return nil
	}

	execution, err := models.FindUnscopedExecutionByID(ID)
	if err != nil {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return nil
	}

	//
	// Try to authenticate using the token issued by SuperPlane itself.
	// It the integration does not support OIDC tokens, this is the method of authentication used.
	//
	err = s.jwt.Validate(token, req.ExecutionID)
	if err == nil {
		return execution
	}

	//
	// If authenticating with the token issued by SuperPlane itself fails,
	// try to authenticate expecting an OIDC ID token issued by the integration.
	//
	integrationResource, err := execution.IntegrationResource(req.ExternalID)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	verifier, err := s.registry.GetOIDCVerifier(integrationResource.IntegrationType)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	err = verifier.Verify(r.Context(), s.oidcVerifier, token, integrations.VerifyTokenOptions{
		IntegrationURL: integrationResource.IntegrationURL,
		ParentResource: integrationResource.ParentExternalID,
		ChildResource:  integrationResource.ExecutionExternalID,
	})

	if err != nil {
		log.Warnf("Invalid token for execution %s: %v", req.ExecutionID, err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	return execution
}

type ExecutionOutputRequest struct {
	ExecutionID string         `json:"execution_id"`
	ExternalID  string         `json:"external_id"`
	Outputs     map[string]any `json:"outputs"`
}

func (s *Server) HandleExecutionOutputs(w http.ResponseWriter, r *http.Request) {
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

	var req ExecutionOutputRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	execution := s.authenticateExecution(w, r, &req)
	if execution == nil {
		return
	}

	stage, err := models.FindStageByID(execution.CanvasID.String(), execution.StageID.String())
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

func (s *Server) HandleCustomWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceIDFromRequest := vars["sourceID"]
	sourceID, err := uuid.Parse(sourceIDFromRequest)
	if err != nil {
		http.Error(w, "source not found", http.StatusNotFound)
		return
	}

	signature := r.Header.Get("X-Signature-256")
	if signature == "" {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	source, err := models.FindEventSource(sourceID)
	if err != nil {
		http.Error(w, "source not found", http.StatusNotFound)
		return
	}

	// Only webhook event sources can receive webhook events
	if source.Type != models.EventSourceTypeWebhook {
		http.Error(w, "webhook events not supported for this event source type", http.StatusBadRequest)
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

	key, err := s.encryptor.Decrypt(r.Context(), source.Key, []byte(source.ID.String()))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := crypto.VerifySignature(key, body, signature); err != nil {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	headers, err := parseHeaders(&r.Header)
	if err != nil {
		http.Error(w, "Error parsing headers", http.StatusInternalServerError)
		return
	}

	event, err := models.CreateEvent(source.ID, source.CanvasID, source.Name, models.SourceTypeEventSource, "custom", body, headers)
	if err != nil {
		http.Error(w, "Error receiving event", http.StatusInternalServerError)
		return
	}

	err = messages.NewEventCreatedMessage(source.CanvasID.String(), event).Publish()
	if err != nil {
		log.Errorf("failed to publish event created message: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleIntegrationWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceIDFromRequest := vars["sourceID"]
	sourceID, err := uuid.Parse(sourceIDFromRequest)
	if err != nil {
		http.Error(w, "source not found", http.StatusNotFound)
		return
	}

	integrationName := vars["integrationName"]
	eventHandler, err := s.registry.GetEventHandler(integrationName)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	source, err := models.FindEventSource(sourceID)
	if err != nil {
		http.Error(w, "source not found", http.StatusNotFound)
		return
	}

	if source.Type != integrationName {
		http.Error(w, "events not supported for this event source type", http.StatusBadRequest)
		return
	}

	integration, err := source.FindIntegration()
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	if integration.Type != integrationName {
		http.Error(w, "integration type mismatch", http.StatusNotFound)
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

	//
	// Use integration event handler to convert request into an event.
	//
	event, err := eventHandler.Handle(body, r.Header)
	if err != nil {
		if err == integrations.ErrInvalidSignature {
			http.Error(w, "Invalid signature", http.StatusForbidden)
			return
		}

		log.Errorf("Error handling event for %s: %v", integrationName, err)
		http.Error(w, "error handling webhook", http.StatusInternalServerError)
		return
	}

	key, err := s.encryptor.Decrypt(r.Context(), source.Key, []byte(source.ID.String()))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := crypto.VerifySignature(key, body, event.Signature()); err != nil {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	//
	// Store event in database.
	//
	headers, err := parseHeaders(&r.Header)
	if err != nil {
		http.Error(w, "Error parsing headers", http.StatusInternalServerError)
		return
	}

	newEvent, err := models.CreateEvent(source.ID, source.CanvasID, source.Name, models.SourceTypeEventSource, event.Type(), body, headers)
	if err != nil {
		http.Error(w, "error creating event", http.StatusInternalServerError)
		return
	}

	err = messages.NewEventCreatedMessage(source.CanvasID.String(), newEvent).Publish()
	if err != nil {
		log.Errorf("failed to publish event created message: %v", err)
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

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	canvasID := vars["canvasId"]
	_, err := models.FindCanvasByID(canvasID, user.OrganizationID)
	if err != nil {
		http.Error(w, "canvas not found", http.StatusNotFound)
		return
	}

	accessibleCanvases, err := s.authService.GetAccessibleCanvasesForUser(user.ID.String())
	if err != nil {
		log.Errorf("Error getting accessible canvases for user %s: %v", user.ID.String(), err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	if !slices.Contains(accessibleCanvases, canvasID) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		log.Errorf("Failed to upgrade to WebSocket: %v", err)
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

	s.Router.PathPrefix(webBasePath).Handler(middleware.AccountAuthMiddleware(s.jwt).Middleware(proxyHandler))
}

func getAvatarURL(providers []models.AccountProvider) string {
	if len(providers) == 0 {
		return ""
	}

	return providers[0].AvatarURL
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
