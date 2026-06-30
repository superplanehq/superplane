package server

import (
	"context"
	"fmt"
	"net/http"
	// Registers pprof handlers on http.DefaultServeMux, served by startPprofServer.
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
	agenttools "github.com/superplanehq/superplane/pkg/agents/agent_tools"
	"github.com/superplanehq/superplane/pkg/agents/anthropic"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/git"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	grpc "github.com/superplanehq/superplane/pkg/grpc"
	agentsActions "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/networkpolicy"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/public"
	registry "github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/registryimports"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/pkg/workers"
	"gorm.io/gorm"
)

var _ = registryimports.Loaded

var agentProviderOverride = struct {
	sync.Mutex
	provider agents.Provider
}{}

func SetAgentProviderForTests(provider agents.Provider) func() {
	agentProviderOverride.Lock()
	previous := agentProviderOverride.provider
	agentProviderOverride.provider = provider
	agentProviderOverride.Unlock()

	return func() {
		agentProviderOverride.Lock()
		agentProviderOverride.provider = previous
		agentProviderOverride.Unlock()
	}
}

func getAgentProviderOverride() agents.Provider {
	agentProviderOverride.Lock()
	defer agentProviderOverride.Unlock()
	return agentProviderOverride.provider
}

func buildAgentService(authService authorization.Authorization) (agents.Provider, agentsActions.AgentsService) {
	if provider := getAgentProviderOverride(); provider != nil {
		log.WithField("provider", provider.Name()).Info("Managed agents enabled with provider override")
		return provider, agents.NewService(provider, authService)
	}

	cfg := config.LoadAnthropicAgentConfig()
	if !cfg.Enabled() {
		log.Info("Anthropic managed agents disabled: missing ANTHROPIC_* env vars")
		return nil, nil
	}

	if err := anthropic.SyncDefaultAgentPrompt(context.Background(), anthropic.Config{
		APIKey:        cfg.APIKey,
		AgentID:       cfg.AgentID,
		EnvironmentID: cfg.EnvironmentID,
	}); err != nil {
		log.WithError(err).Warn("failed to sync Anthropic managed agent prompt; continuing with provider prompt")
	} else {
		log.Info("Anthropic managed agent prompt synced")
	}

	fileResources, err := anthropic.LoadDefaultSessionResources(context.Background(), anthropic.Config{
		APIKey:        cfg.APIKey,
		AgentID:       cfg.AgentID,
		EnvironmentID: cfg.EnvironmentID,
	})
	if err != nil {
		log.WithError(err).Warn("failed to load Anthropic session resources; continuing without mounted references")
		fileResources = nil
	}

	provider, err := anthropic.New(anthropic.Config{
		APIKey:        cfg.APIKey,
		AgentID:       cfg.AgentID,
		EnvironmentID: cfg.EnvironmentID,
		Resources:     fileResources,
	})
	if err != nil {
		log.WithError(err).Warn("failed to initialise Anthropic managed agents provider")
		return nil, nil
	}

	service := agents.NewService(provider, authService)
	log.Info("Anthropic managed agents enabled")
	return provider, service
}

func startWorkers(
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	oidcProvider oidc.Provider,
	gitProvider gitprovider.Provider,
	baseURL string,
	authService authorization.Authorization,
	agentProvider agents.Provider,
) {
	log.Println("Starting Workers")

	rabbitMQURL, err := config.RabbitMQURL()
	if err != nil {
		panic(err)
	}

	if os.Getenv("START_CONSUMERS") == "yes" {
		startEmailConsumers(rabbitMQURL, encryptor, baseURL)
	}

	if os.Getenv("START_WORKFLOW_EVENT_ROUTER") == "yes" || os.Getenv("START_EVENT_ROUTER") == "yes" {
		log.Println("Starting Event Router")

		w := workers.NewEventRouter(rabbitMQURL)
		go w.Start(context.Background())
	}

	if os.Getenv("START_RUN_FINALIZER") == "yes" {
		log.Println("Starting Run Finalizer")

		w := workers.NewRunFinalizer(rabbitMQURL)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WORKFLOW_NODE_EXECUTOR") == "yes" || os.Getenv("START_NODE_EXECUTOR") == "yes" {
		log.Println("Starting Node Executor")

		webhookBaseURL := getWebhookBaseURL(baseURL)
		w := workers.NewNodeExecutor(encryptor, registry, gitProvider, oidcProvider, baseURL, webhookBaseURL, rabbitMQURL, authService)
		go w.Start(context.Background())
	}

	if os.Getenv("START_NODE_REQUEST_WORKER") == "yes" {
		log.Println("Starting Node Request Worker")

		webhookBaseURL := getWebhookBaseURL(baseURL)
		w := workers.NewNodeRequestWorker(encryptor, registry, gitProvider, webhookBaseURL, authService)
		go w.Start(context.Background())
	}

	if os.Getenv("START_APP_INSTALLATION_REQUEST_WORKER") == "yes" || os.Getenv("START_INTEGRATION_REQUEST_WORKER") == "yes" {
		log.Println("Starting Integration Request Worker")

		webhooksBaseURL := getWebhookBaseURL(baseURL)
		w := workers.NewIntegrationRequestWorker(encryptor, registry, oidcProvider, baseURL, webhooksBaseURL)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WORKFLOW_NODE_QUEUE_WORKER") == "yes" || os.Getenv("START_NODE_QUEUE_WORKER") == "yes" {
		log.Println("Starting Node Queue Worker")

		w := workers.NewNodeQueueWorker(registry, gitProvider, rabbitMQURL)
		go w.Start(context.Background())
	}

	// Start Webhook Provisioner when internal API runs so integration webhooks (e.g. GCP On VM Created) get provisioned.
	// Can be disabled by setting START_WEBHOOK_PROVISIONER=no.
	if os.Getenv("START_WEBHOOK_PROVISIONER") != "no" {
		if os.Getenv("START_WEBHOOK_PROVISIONER") == "yes" {
			log.Println("Starting Webhook Provisioner")
		}
		webhookBaseURL := getWebhookBaseURL(baseURL)
		w := workers.NewWebhookProvisioner(webhookBaseURL, encryptor, registry)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WEBHOOK_CLEANUP_WORKER") == "yes" {
		log.Println("Starting Webhook Cleanup Worker")

		w := workers.NewWebhookCleanupWorker(encryptor, registry, baseURL)
		go w.Start(context.Background())
	}

	if os.Getenv("START_INSTALLATION_CLEANUP_WORKER") == "yes" || os.Getenv("START_INTEGRATION_CLEANUP_WORKER") == "yes" {
		log.Println("Starting Integration Cleanup Worker")

		w := workers.NewIntegrationCleanupWorker(registry, encryptor, baseURL)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WORKFLOW_CLEANUP_WORKER") == "yes" || os.Getenv("START_CANVAS_CLEANUP_WORKER") == "yes" {
		log.Println("Starting Canvas Cleanup Worker")

		w := workers.NewCanvasCleanupWorker(gitProvider, agentProvider)
		go w.Start(context.Background())
	}

	if os.Getenv("START_REPOSITORY_PROVISIONER") == "yes" {
		log.Println("Starting Repository Provisioner")
		w := workers.NewRepositoryProvisionerWorker(rabbitMQURL, gitProvider)
		go w.Start(context.Background())
	}

	var workerUsageService usage.Service
	initWorkerUsageService := func() (usage.Service, error) {
		if workerUsageService != nil {
			return workerUsageService, nil
		}

		service, err := usage.NewServiceFromEnv()
		if err != nil {
			return nil, err
		}
		workerUsageService = service
		return workerUsageService, nil
	}
	getRequiredWorkerUsageService := func() usage.Service {
		service, err := initWorkerUsageService()
		if err != nil {
			log.Fatalf("failed to initialize usage service worker dependency: %v", err)
		}
		return service
	}
	getOptionalWorkerUsageService := func() usage.Service {
		service, err := initWorkerUsageService()
		if err != nil {
			log.Printf("usage service unavailable for agent canvas tool: %v", err)
			return nil
		}
		return service
	}

	if os.Getenv("START_ORGANIZATION_CLEANUP_WORKER") == "yes" {
		log.Println("Starting Organization Cleanup Worker")

		w := workers.NewOrganizationCleanupWorker(gitProvider, agentProvider)
		go w.Start(context.Background())
	}

	if agentProvider != nil && os.Getenv("START_AGENT_STREAM_WORKER") != "no" {
		log.Println("Starting Agent Stream Worker")
		agentToolRegistry := agenttools.NewRegistry(agenttools.Dependencies{
			Encryptor:         encryptor,
			ComponentRegistry: registry,
			GitProvider:       gitProvider,
			WebhookBaseURL:    getWebhookBaseURL(baseURL),
			AuthService:       authService,
			UsageService:      getOptionalWorkerUsageService(),
		})
		w := workers.NewAgentStreamWorkerWithUsageService(
			agentProvider,
			rabbitMQURL,
			getOptionalWorkerUsageService(),
			agentToolRegistry,
		)
		go w.Start(context.Background())
	}

	if os.Getenv("START_EVENT_RETENTION_WORKER") == "yes" || os.Getenv("START_USAGE_SYNC_WORKER") == "yes" {
		usageService := getRequiredWorkerUsageService()

		if os.Getenv("START_EVENT_RETENTION_WORKER") == "yes" && usageService.Enabled() {
			log.Println("Starting Event Retention Worker")
			w := workers.NewEventRetentionWorker(usageService)
			go w.Start(context.Background())
		}

		if os.Getenv("START_USAGE_SYNC_WORKER") == "yes" && usageService.Enabled() {
			log.Println("Starting Usage Sync Worker")
			w := workers.NewUsageSyncWorker(rabbitMQURL, usageService)
			go w.Start(context.Background())
		}
	}

}

func startEmailConsumers(rabbitMQURL string, encryptor crypto.Encryptor, baseURL string) {
	emailService := services.BuildEmailService(encryptor, services.EmailServiceConfig{
		TemplateDir:       os.Getenv("TEMPLATE_DIR"),
		OwnerSetupEnabled: os.Getenv("OWNER_SETUP_ENABLED") == "yes",
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		FromName:          os.Getenv("EMAIL_FROM_NAME"),
		FromEmail:         os.Getenv("EMAIL_FROM_ADDRESS"),
	})
	if emailService == nil {
		log.Warn("Email Consumers not started - missing required environment variables")
		return
	}

	startEmailConsumersWithService(rabbitMQURL, emailService, baseURL)
}

func startEmailConsumersWithService(rabbitMQURL string, emailService services.EmailService, baseURL string) {
	log.Println("Starting Magic Code Email Consumer")
	magicCodeEmailConsumer := workers.NewMagicCodeEmailConsumer(rabbitMQURL, emailService, baseURL)
	go magicCodeEmailConsumer.Start()
}

func buildGRPCServices(
	baseURL, webhooksBaseURL string,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	registry *registry.Registry,
	oidcProvider oidc.Provider,
	gitProvider gitprovider.Provider,
	agentService agentsActions.AgentsService,
) (*grpc.Services, error) {
	usageService, err := usage.NewServiceFromEnv()
	if err != nil {
		return nil, fmt.Errorf("initialize usage service: %w", err)
	}

	return grpc.NewServices(grpc.ServicesConfig{
		BaseURL:         baseURL,
		WebhooksBaseURL: webhooksBaseURL,
		Encryptor:       encryptor,
		AuthService:     authService,
		Registry:        registry,
		OIDCProvider:    oidcProvider,
		GitProvider:     gitProvider,
		AgentService:    agentService,
		UsageService:    usageService,
	})
}

func startPublicAPI(
	baseURL, basePath string,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	jwtSigner *jwt.Signer,
	oidcProvider oidc.Provider,
	authService authorization.Authorization,
	gitProvider gitprovider.Provider,
	grpcServices *grpc.Services,
) {
	log.Println("Starting Public API with integrated Web Server")

	appEnv := os.Getenv("APP_ENV")
	templateDir := os.Getenv("TEMPLATE_DIR")
	blockSignup := os.Getenv("BLOCK_SIGNUP") == "yes"
	usageService, err := usage.NewServiceFromEnv()
	if err != nil {
		log.Panicf("failed to initialize usage service for public api: %v", err)
	}

	webhooksBaseURL := getWebhookBaseURL(baseURL)
	server, err := public.NewServer(
		encryptor,
		registry,
		jwtSigner,
		oidcProvider,
		gitProvider,
		basePath,
		baseURL,
		webhooksBaseURL,
		appEnv,
		templateDir,
		authService,
		usageService,
		blockSignup,
	)
	if err != nil {
		log.Panicf("Error creating public API server: %v", err)
	}

	// Start the EventDistributer worker if enabled
	if os.Getenv("START_EVENT_DISTRIBUTER") == "yes" {
		log.Println("Starting Event Distributer Worker")
		eventDistributer := workers.NewEventDistributer(server.WebsocketHub())
		go eventDistributer.Start()
	} else {
		log.Println("Event Distributer not started (START_EVENT_DISTRIBUTER != yes)")
	}

	log.Println("Registering gRPC gateway handlers on Public API")

	err = server.RegisterGRPCGateway(grpcServices)
	if err != nil {
		log.Fatalf("Failed to register gRPC gateway: %v", err)
	}

	server.RegisterOpenAPIHandler()

	// Register web routes only if START_WEB_SERVER is set to "yes"
	if os.Getenv("START_WEB_SERVER") == "yes" {
		webBasePath := os.Getenv("WEB_BASE_PATH")
		log.Printf("Registering web routes in public API server with base path: %s", webBasePath)
		server.RegisterWebRoutes(webBasePath)
	} else {
		log.Println("Web server routes not registered (START_WEB_SERVER != yes)")
	}

	err = server.Serve("0.0.0.0", lookupPublicAPIPort())
	if err != nil {
		log.Fatal(err)
	}
}

func lookupPublicAPIPort() int {
	port := 8000

	if p := os.Getenv("PUBLIC_API_PORT"); p != "" {
		if v, errConv := strconv.Atoi(p); errConv == nil && v > 0 {
			port = v
		} else {
			log.Warnf("Invalid PUBLIC_API_PORT %q, falling back to 8000", p)
		}
	}

	return port
}

func configureLogging() {
	appEnv := os.Getenv("APP_ENV")

	if appEnv == "development" || appEnv == "test" {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   false,
			TimestampFormat: time.Stamp,
		})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.StampMilli,
		})
	}
}

func setupOtel() {
	if os.Getenv("OTEL_ENABLED") != "yes" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := telemetry.InitMetrics(ctx); err != nil {
		log.Warnf("Failed to initialize OpenTelemetry metrics: %v", err)
	} else {
		log.Info("OpenTelemetry metrics initialized")
	}

	if err := telemetry.InitTracing(ctx); err != nil {
		log.Warnf("Failed to initialize OpenTelemetry tracing: %v", err)
	} else {
		log.Info("OpenTelemetry tracing initialized for critical API endpoints")
	}
}

func startPprofServer() {
	if os.Getenv("PPROF_ENABLED") != "yes" {
		return
	}

	port := os.Getenv("PPROF_PORT")
	if port == "" {
		port = "6060"
	}

	// Sample contention so /debug/pprof/block and /debug/pprof/mutex are useful.
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(5)

	go func() {
		log.Infof("pprof server listening on :%s", port)
		if err := http.ListenAndServe("0.0.0.0:"+port, nil); err != nil {
			log.Warnf("pprof server stopped: %v", err)
		}
	}()
}

func Start() {
	configureLogging()
	setupOtel()
	startPprofServer()

	telemetry.InitSentry()
	telemetry.StartBeacon()

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		panic("ENCRYPTION_KEY can't be empty")
	}

	log.SetLevel(log.DebugLevel)

	var encryptorInstance crypto.Encryptor
	if os.Getenv("NO_ENCRYPTION") == "yes" {
		log.Warn("NO_ENCRYPTION is set to yes, using NoOpEncryptor")
		encryptorInstance = crypto.NewNoOpEncryptor()
	} else {
		encryptorInstance = crypto.NewAESGCMEncryptor([]byte(encryptionKey))
	}

	authService, err := authorization.NewAuthService()
	if err != nil {
		log.Fatalf("failed to create auth service: %v", err)
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		panic("BASE_URL must be set")
	}

	basePath := os.Getenv("PUBLIC_API_BASE_PATH")
	if basePath == "" {
		panic("PUBLIC_API_BASE_PATH must be set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET must be set")
	}

	oidcKeysPath := os.Getenv("OIDC_KEYS_PATH")
	if oidcKeysPath == "" {
		panic("OIDC_KEYS_PATH must be set")
	}

	appEnv := os.Getenv("APP_ENV")
	jwtSigner := jwt.NewSigner(jwtSecret)
	webhooksBaseURL := getWebhookBaseURL(baseURL)
	oidcProvider, err := oidc.NewProviderFromKeyDir(webhooksBaseURL, oidcKeysPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load OIDC keys: %v", err))
	}

	log.Println("Creating Git Provider")
	gitProvider, err := git.NewProvider()
	if err != nil {
		panic(fmt.Sprintf("failed to create git provider: %v", err))
	}

	registry, err := registry.NewRegistryWithOptions(registry.RegistryOptions{
		Encryptor: encryptorInstance,
		AppEnv:    appEnv,
		HTTP: registry.HTTPOptions{
			MaxResponseBytes: DefaultMaxHTTPResponseBytes,
			PolicyResolver: func() (registry.HTTPPolicy, error) {
				policy, err := networkpolicy.ResolveHTTPPolicy()
				if err != nil {
					return registry.HTTPPolicy{}, err
				}

				return registry.HTTPPolicy{
					BlockedHosts:    policy.BlockedHosts,
					PrivateIPRanges: policy.PrivateIPRanges,
				}, nil
			},
			PolicyResolverInTransaction: func(tx *gorm.DB) (registry.HTTPPolicy, error) {
				policy, err := networkpolicy.ResolveHTTPPolicyInTransaction(tx)
				if err != nil {
					return registry.HTTPPolicy{}, err
				}

				return registry.HTTPPolicy{
					BlockedHosts:    policy.BlockedHosts,
					PrivateIPRanges: policy.PrivateIPRanges,
				}, nil
			},
			PolicyCacheTTL: 5 * time.Second,
		},
	})

	if err != nil {
		panic(fmt.Sprintf("failed to create registry: %v", err))
	}

	agentProvider, agentService := buildAgentService(authService)

	var grpcServices *grpc.Services
	if os.Getenv("START_PUBLIC_API") == "yes" {
		services, err := buildGRPCServices(
			baseURL,
			webhooksBaseURL,
			encryptorInstance,
			authService,
			registry,
			oidcProvider,
			gitProvider,
			agentService,
		)
		if err != nil {
			log.Fatalf("failed to build gRPC services: %v", err)
		}
		grpcServices = services

		go startPublicAPI(
			baseURL,
			basePath,
			encryptorInstance,
			registry,
			jwtSigner,
			oidcProvider,
			authService,
			gitProvider,
			grpcServices,
		)
	}

	startWorkers(
		encryptorInstance,
		registry,
		oidcProvider,
		gitProvider,
		baseURL,
		authService,
		agentProvider,
	)

	log.Println("SuperPlane is UP.")

	select {}
}

// getWebhookBaseURL returns the webhook base URL, using the same pattern as SyncContext.
// Use WEBHOOKS_BASE_URL if set, otherwise fall back to baseURL.
// This allows e2e tests to use a fake/mock webhook URL, and local installations to use a different
// URL for webhooks (e.g., a tunnel URL) when the base app is running on localhost.
func getWebhookBaseURL(baseURL string) string {
	webhookBaseURL := os.Getenv("WEBHOOKS_BASE_URL")
	if webhookBaseURL == "" {
		webhookBaseURL = baseURL
	}
	return webhookBaseURL
}

/*
 * 8MB is the default maximum response size for HTTP responses.
 * This prevents component/trigger implementations from using too much memory,
 * and also from emitting large events.
 */
var DefaultMaxHTTPResponseBytes int64 = 8 * 1024 * 1024
