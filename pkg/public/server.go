package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	pbBlueprints "github.com/superplanehq/superplane/pkg/protos/blueprints"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbComponents "github.com/superplanehq/superplane/pkg/protos/components"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbIntegrations "github.com/superplanehq/superplane/pkg/protos/integrations"
	pbMe "github.com/superplanehq/superplane/pkg/protos/me"
	pbOrg "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecret "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbServiceAccounts "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	pbTriggers "github.com/superplanehq/superplane/pkg/protos/triggers"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	pbWidgets "github.com/superplanehq/superplane/pkg/protos/widgets"
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
	oidcProvider          oidc.Provider
	authService           authorization.Authorization
	timeoutHandlerTimeout time.Duration
	upgrader              *websocket.Upgrader
	Router                *mux.Router
	BasePath              string
	BaseURL               string
	WebhooksBaseURL       string
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
	oidcProvider oidc.Provider,
	basePath string,
	baseURL string,
	webhooksBaseURL string,
	appEnv string,
	templateDir string,
	authorizationService authorization.Authorization,
	blockSignup bool,
	middlewares ...mux.MiddlewareFunc,
) (*Server, error) {

	// Initialize OAuth providers from environment variables
	passwordLoginEnabled := os.Getenv("ENABLE_PASSWORD_LOGIN") == "yes"
	authHandler := authentication.NewHandler(jwtSigner, encryptor, authorizationService, appEnv, templateDir, blockSignup, passwordLoginEnabled)
	providers := getOAuthProviders()
	authHandler.InitializeProviders(providers)

	server := &Server{
		BaseURL:               baseURL,
		WebhooksBaseURL:       webhooksBaseURL,
		BasePath:              basePath,
		wsHub:                 ws.NewHub(),
		authHandler:           authHandler,
		isDev:                 appEnv == "development",
		timeoutHandlerTimeout: 15 * time.Second,
		encryptor:             encryptor,
		jwt:                   jwtSigner,
		oidcProvider:          oidcProvider,
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

	// Google
	if googleKey := os.Getenv("GOOGLE_CLIENT_ID"); googleKey != "" {
		if googleSecret := os.Getenv("GOOGLE_CLIENT_SECRET"); googleSecret != "" {
			providers["google"] = authentication.ProviderConfig{
				Key:         googleKey,
				Secret:      googleSecret,
				CallbackURL: fmt.Sprintf("%s/auth/google/callback", baseURL),
			}
		}
	}
	return providers
}

func (s *Server) RegisterGRPCGateway(grpcServerAddr string) error {
	ctx := context.Background()

	grpcGatewayMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(headersMatcher),
		runtime.SetQueryParameterParser(&grpc.QueryParser{}),
	)

	opts := []grpcLib.DialOption{grpcLib.WithTransportCredentials(insecure.NewCredentials())}

	err := pbUsers.RegisterUsersHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
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

	err = pbComponents.RegisterComponentsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbTriggers.RegisterTriggersHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbWidgets.RegisterWidgetsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbBlueprints.RegisterBlueprintsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbCanvases.RegisterCanvasesHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
	if err != nil {
		return err
	}

	err = pbServiceAccounts.RegisterServiceAccountsHandlerFromEndpoint(ctx, grpcGatewayMux, grpcServerAddr, opts)
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

	accountAuthMiddleware := middleware.AccountAuthMiddleware(s.jwt)
	protectedAccountGRPCHandler := accountAuthMiddleware(s.grpcGatewayAccountHandler(grpcGatewayMux))

	s.Router.PathPrefix("/api/v1/users").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/groups").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/roles").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/canvases").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/organizations").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/invite-links").Handler(protectedAccountGRPCHandler)
	s.Router.PathPrefix("/api/v1/integrations").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/secrets").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/me").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/components").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/triggers").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/widgets").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/blueprints").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/service-accounts").Handler(protectedGRPCHandler)
	s.Router.PathPrefix("/api/v1/workflows").Handler(protectedGRPCHandler)

	return nil
}

func headersMatcher(key string) (string, bool) {
	switch key {
	case "X-User-Id", "X-Organization-Id", "X-Account-Id":
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

func (s *Server) grpcGatewayAccountHandler(grpcGatewayMux *runtime.ServeMux) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account, ok := middleware.GetAccountFromContext(r.Context())
		if !ok {
			http.Error(w, "Account not found in context", http.StatusUnauthorized)
			return
		}

		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.Header.Set("x-account-id", account.ID.String())
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
		"/ws/{workflowId}",
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

func (s *Server) InitRouter(additionalMiddlewares ...mux.MiddlewareFunc) {
	r := mux.NewRouter().StrictSlash(true)
	r.Use(otelmux.Middleware(
		"superplane-public-api",
		otelmux.WithTracerProvider(nooptrace.NewTracerProvider()),
	))
	r.Use(middleware.LoggingMiddleware(log.StandardLogger()))

	// Register authentication routes (no auth required)
	s.authHandler.RegisterRoutes(r)

	//
	// Public routes (no authentication required)
	//
	publicRoute := r.Methods(http.MethodGet, http.MethodPost).Subrouter()

	// Health check
	publicRoute.HandleFunc("/health", s.HealthCheck).Methods("GET")
	publicRoute.HandleFunc("/api/v1/setup-owner", s.setupOwner).Methods("POST")

	// OIDC discovery endpoints
	publicRoute.HandleFunc("/.well-known/openid-configuration", s.handleOIDCConfiguration).Methods("GET")
	publicRoute.HandleFunc("/.well-known/jwks.json", s.handleOIDCJWKS).Methods("GET")

	// Test endpoints
	publicRoute.HandleFunc("/server1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	publicRoute.HandleFunc("/server2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}).Methods("GET")

	publicRoute.HandleFunc("/server3", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	//
	// Webhook endpoints for triggers
	//
	publicRoute.
		HandleFunc(s.BasePath+"/webhooks/{webhookID}", s.HandleWebhook).
		Methods("POST")

	//
	// HTTP endpoints for app installations
	// Match all paths under /integrations/{integrationID}/ including subpaths
	//
	r.PathPrefix(s.BasePath+"/integrations/{integrationID}").HandlerFunc(s.HandleIntegrationRequest).
		Methods("GET", "POST")

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

type oidcDiscoveryResponse struct {
	Issuer                           string   `json:"issuer"`
	JWKSURI                          string   `json:"jwks_uri"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
}

type jwksResponse struct {
	Keys []oidc.PublicJWK `json:"keys"`
}

func (s *Server) handleOIDCConfiguration(w http.ResponseWriter, _ *http.Request) {
	baseURL := strings.TrimRight(s.WebhooksBaseURL, "/")
	response := oidcDiscoveryResponse{
		Issuer:                           baseURL,
		JWKSURI:                          baseURL + "/.well-known/jwks.json",
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		SubjectTypesSupported:            []string{"public"},
		ResponseTypesSupported:           []string{"id_token"},
	}
	respondJSON(w, response)
}

func (s *Server) handleOIDCJWKS(w http.ResponseWriter, _ *http.Request) {
	response := jwksResponse{
		Keys: s.oidcProvider.PublicJWKs(),
	}
	respondJSON(w, response)
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) HandleIntegrationRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	integrationIDFromRequest := vars["integrationID"]
	integrationID, err := uuid.Parse(integrationIDFromRequest)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	integrationInstance, err := models.FindUnscopedIntegration(integrationID)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	integration, err := s.registry.GetIntegration(integrationInstance.AppName)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	integration.HandleRequest(core.HTTPRequestContext{
		Logger:          logging.ForIntegration(*integrationInstance),
		Request:         r,
		Response:        w,
		BaseURL:         s.BaseURL,
		WebhooksBaseURL: s.WebhooksBaseURL,
		OrganizationID:  integrationInstance.OrganizationID.String(),
		HTTP:            s.registry.HTTPContext(),
		Integration: contexts.NewIntegrationContext(
			database.Conn(),
			nil,
			integrationInstance,
			s.encryptor,
			s.registry,
		),
	})

	err = database.Conn().Save(&integrationInstance).Error
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}
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
	// Create the organization
	//
	tx := database.Conn().Begin()
	log.Infof("Creating organization %s for account %s", req.Name, account.Email)
	organization, err := models.CreateOrganizationInTransaction(tx, req.Name, "")

	if err != nil {
		tx.Rollback()

		if err.Error() == "name already used" {
			http.Error(w, "Organization name already in use", http.StatusConflict)
			return
		}

		log.Errorf("Error creating organization %s: %v", req.Name, err)
		http.Error(w, "Failed to create organization", http.StatusInternalServerError)
		return
	}

	//
	// Create the owner user for it
	//
	log.Infof("Creating user for new organization %s (%s)", organization.Name, organization.ID)
	user, err := models.CreateUserInTransaction(tx, organization.ID, account.ID, account.Email, account.Name)
	if err != nil {
		tx.Rollback()
		log.Errorf("Error creating user for new organization %s (%s): %v", organization.Name, organization.ID, err)
		http.Error(w, "Failed to create user account", http.StatusInternalServerError)
		return
	}

	//
	// Finally, set up RBAC for the new organization.
	//
	log.Infof("Setting up RBAC policies for new organization %s (%s)", organization.Name, organization.ID)
	err = s.authService.SetupOrganization(tx, organization.ID.String(), user.ID.String())
	if err != nil {
		tx.Rollback()
		log.Errorf("Error setting up RBAC policies for %s (%s): %v", organization.Name, organization.ID, err)
		http.Error(w, "Failed to set up organization roles", http.StatusInternalServerError)
		return
	}

	err = tx.Commit().Error
	if err != nil {
		log.Errorf("Error committing transaction for organization %s (%s) creation: %v", organization.Name, organization.ID, err)
		http.Error(w, "Failed to create organization", http.StatusInternalServerError)
		return
	}

	log.Infof("Organization %s (%s) created successfully", organization.Name, organization.ID)

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
		CanvasCount int64  `json:"canvasCount"`
		MemberCount int64  `json:"memberCount"`
	}

	organizations, err := models.FindOrganizationsForAccount(account.Email)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	orgIDs := make([]string, 0, len(organizations))
	for _, organization := range organizations {
		orgIDs = append(orgIDs, organization.ID.String())
	}

	canvasCounts, err := models.CountCanvasesByOrganizationIDs(orgIDs)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	memberCounts, err := models.CountActiveUsersByOrganizationIDs(orgIDs)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := []Organization{}
	for _, organization := range organizations {
		orgID := organization.ID.String()
		response = append(response, Organization{
			ID:          organization.ID.String(),
			Name:        organization.Name,
			Description: organization.Description,
			CanvasCount: canvasCounts[orgID],
			MemberCount: memberCounts[orgID],
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

func (s *Server) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookIDFromRequest := vars["webhookID"]
	webhookID, err := uuid.Parse(webhookIDFromRequest)
	if err != nil {
		http.Error(w, "webhook not found", http.StatusNotFound)
		return
	}

	_, err = models.FindWebhook(webhookID)
	if err != nil {
		http.Error(w, "webhook not found", http.StatusNotFound)
		return
	}

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

	nodes, err := models.FindWebhookNodes(webhookID)
	if err != nil {
		http.Error(w, "webhook not found", http.StatusNotFound)
		return
	}

	for _, node := range nodes {
		code, err := s.executeWebhookNode(r.Context(), body, r.Header, node)
		if err != nil {
			http.Error(w, fmt.Sprintf("error handling webhook: %v", err), code)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) executeWebhookNode(ctx context.Context, body []byte, headers http.Header, node models.CanvasNode) (int, error) {
	if node.Type == models.NodeTypeTrigger {
		return s.executeTriggerNode(ctx, body, headers, node)
	}

	return s.executeComponentNode(ctx, body, headers, node)
}

func (s *Server) executeTriggerNode(ctx context.Context, body []byte, headers http.Header, node models.CanvasNode) (int, error) {
	ref := node.Ref.Data()
	trigger, err := s.registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("trigger not found: %w", err)
	}

	logger := logging.ForNode(node)
	tx := database.Conn()
	var integrationCtx core.IntegrationContext
	if node.AppInstallationID != nil {
		integration, integrationErr := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if integrationErr != nil {
			return http.StatusInternalServerError, integrationErr
		}

		logger = logging.WithIntegration(logger, *integration)
		integrationCtx = contexts.NewIntegrationContext(tx, &node, integration, s.encryptor, s.registry)
	}

	return trigger.HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		WorkflowID:    node.WorkflowID.String(),
		NodeID:        node.NodeID,
		Configuration: node.Configuration.Data(),
		Metadata:      contexts.NewNodeMetadataContext(tx, &node),
		Logger:        logger,
		HTTP:          s.registry.HTTPContext(),
		Webhook:       contexts.NewNodeWebhookContext(ctx, tx, s.encryptor, &node, s.BaseURL+s.BasePath),
		Events:        contexts.NewEventContext(tx, &node),
		Integration:   integrationCtx,
	})
}

func (s *Server) executeComponentNode(ctx context.Context, body []byte, headers http.Header, node models.CanvasNode) (int, error) {
	ref := node.Ref.Data()
	component, err := s.registry.GetComponent(ref.Component.Name)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("component not found: %w", err)
	}

	logger := logging.ForNode(node)
	tx := database.Conn()
	var integrationCtx core.IntegrationContext
	if node.AppInstallationID != nil {
		integration, integrationErr := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if integrationErr != nil {
			return http.StatusInternalServerError, integrationErr
		}

		logger = logging.WithIntegration(logger, *integration)
		integrationCtx = contexts.NewIntegrationContext(tx, &node, integration, s.encryptor, s.registry)
	}

	return component.HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		WorkflowID:    node.WorkflowID.String(),
		NodeID:        node.NodeID,
		Configuration: node.Configuration.Data(),
		Metadata:      contexts.NewNodeMetadataContext(tx, &node),
		Logger:        logger,
		HTTP:          s.registry.HTTPContext(),
		Webhook:       contexts.NewNodeWebhookContext(ctx, tx, s.encryptor, &node, s.BaseURL+s.BasePath),
		Events:        contexts.NewEventContext(tx, &node),
		Integration:   integrationCtx,
		FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
			execution, err := models.FirstNodeExecutionByKVInTransaction(tx, node.WorkflowID, node.NodeID, key, value)
			if err != nil {
				return nil, err
			}

			return &core.ExecutionContext{
				ID:             execution.ID,
				WorkflowID:     execution.WorkflowID.String(),
				NodeID:         execution.NodeID,
				BaseURL:        s.BaseURL,
				Configuration:  execution.Configuration.Data(),
				HTTP:           s.registry.HTTPContext(),
				Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
				NodeMetadata:   contexts.NewNodeMetadataContext(tx, &node),
				ExecutionState: contexts.NewExecutionStateContext(tx, execution),
				Requests:       contexts.NewExecutionRequestContext(tx, execution),
				Logger:         logging.ForExecution(execution, nil),
				Notifications:  contexts.NewNotificationContext(tx, uuid.Nil, execution.WorkflowID),
				CanvasMemory:   contexts.NewCanvasMemoryContext(tx, execution.WorkflowID),
			}, nil
		},
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Infof("New WebSocket connection from %s", r.RemoteAddr)

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	workflowID := vars["workflowId"]

	parsedWorkflowID, err := uuid.Parse(workflowID)
	if err != nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	_, err = models.FindCanvas(user.OrganizationID, parsedWorkflowID)
	if err != nil {
		http.Error(w, "canvas not found", http.StatusNotFound)
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

	client := s.wsHub.NewClient(ws, workflowID)

	<-client.Done
}

// setupDevProxy configures a simple reverse proxy to the Vite development server
func (s *Server) setupDevProxy(webBasePath string) {
	viteHost := os.Getenv("VITE_DEV_HOST")
	if viteHost == "" {
		viteHost = "localhost"
	}

	vitePort := os.Getenv("VITE_DEV_PORT")
	if vitePort == "" {
		vitePort = "5173"
	}

	target, err := url.Parse(fmt.Sprintf("http://%s:%s", viteHost, vitePort))
	if err != nil {
		log.Fatalf("Error parsing Vite dev server URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ModifyResponse = func(res *http.Response) error {
		contentType := res.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/html") {
			return nil
		}

		originalBody, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		_ = res.Body.Close()

		rendered, err := web.RenderIndexTemplate(originalBody)
		if err != nil {
			return err
		}

		res.Body = io.NopCloser(bytes.NewReader(rendered))
		res.ContentLength = int64(len(rendered))
		res.Header.Set("Content-Length", strconv.Itoa(len(rendered)))

		return nil
	}

	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
	}

	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			return
		}

		proxy.ServeHTTP(w, r)
	})

	s.Router.PathPrefix(webBasePath).Handler(proxyHandler)
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
