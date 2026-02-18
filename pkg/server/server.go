package server

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	grpc "github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/public"
	registry "github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/templates"
	"github.com/superplanehq/superplane/pkg/workers"

	// Import integrations, components and triggers to register them via init()
	_ "github.com/superplanehq/superplane/pkg/components/approval"
	_ "github.com/superplanehq/superplane/pkg/components/filter"
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/components/if"
	_ "github.com/superplanehq/superplane/pkg/components/merge"
	_ "github.com/superplanehq/superplane/pkg/components/noop"
	_ "github.com/superplanehq/superplane/pkg/components/ssh"
	_ "github.com/superplanehq/superplane/pkg/components/timegate"
	_ "github.com/superplanehq/superplane/pkg/components/wait"
	_ "github.com/superplanehq/superplane/pkg/integrations/aws"
	_ "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	_ "github.com/superplanehq/superplane/pkg/integrations/circleci"
	_ "github.com/superplanehq/superplane/pkg/integrations/claude"
	_ "github.com/superplanehq/superplane/pkg/integrations/cloudflare"
	_ "github.com/superplanehq/superplane/pkg/integrations/cursor"
	_ "github.com/superplanehq/superplane/pkg/integrations/dash0"
	_ "github.com/superplanehq/superplane/pkg/integrations/datadog"
	_ "github.com/superplanehq/superplane/pkg/integrations/daytona"
	_ "github.com/superplanehq/superplane/pkg/integrations/discord"
	_ "github.com/superplanehq/superplane/pkg/integrations/dockerhub"
	_ "github.com/superplanehq/superplane/pkg/integrations/github"
	_ "github.com/superplanehq/superplane/pkg/integrations/gitlab"
	_ "github.com/superplanehq/superplane/pkg/integrations/hetzner"
	_ "github.com/superplanehq/superplane/pkg/integrations/jira"
	_ "github.com/superplanehq/superplane/pkg/integrations/openai"
	_ "github.com/superplanehq/superplane/pkg/integrations/pagerduty"
	_ "github.com/superplanehq/superplane/pkg/integrations/prometheus"
	_ "github.com/superplanehq/superplane/pkg/integrations/render"
	_ "github.com/superplanehq/superplane/pkg/integrations/rootly"
	_ "github.com/superplanehq/superplane/pkg/integrations/semaphore"
	_ "github.com/superplanehq/superplane/pkg/integrations/sendgrid"
	_ "github.com/superplanehq/superplane/pkg/integrations/slack"
	_ "github.com/superplanehq/superplane/pkg/integrations/smtp"
	_ "github.com/superplanehq/superplane/pkg/triggers/schedule"
	_ "github.com/superplanehq/superplane/pkg/triggers/start"
	_ "github.com/superplanehq/superplane/pkg/triggers/webhook"
	_ "github.com/superplanehq/superplane/pkg/widgets/annotation"
)

func startWorkers(encryptor crypto.Encryptor, registry *registry.Registry, oidcProvider oidc.Provider, baseURL string, authService authorization.Authorization) {
	log.Println("Starting Workers")

	rabbitMQURL, err := config.RabbitMQURL()
	if err != nil {
		panic(err)
	}

	if os.Getenv("START_CONSUMERS") == "yes" {
		startEmailConsumers(rabbitMQURL, encryptor, baseURL, authService)
	}

	if os.Getenv("START_WORKFLOW_EVENT_ROUTER") == "yes" || os.Getenv("START_EVENT_ROUTER") == "yes" {
		log.Println("Starting Event Router")

		w := workers.NewEventRouter()
		go w.Start(context.Background())
	}

	if os.Getenv("START_WORKFLOW_NODE_EXECUTOR") == "yes" || os.Getenv("START_NODE_EXECUTOR") == "yes" {
		log.Println("Starting Node Executor")

		webhookBaseURL := getWebhookBaseURL(baseURL)
		w := workers.NewNodeExecutor(encryptor, registry, baseURL, webhookBaseURL)
		go w.Start(context.Background())
	}

	if os.Getenv("START_NODE_REQUEST_WORKER") == "yes" {
		log.Println("Starting Node Request Worker")

		w := workers.NewNodeRequestWorker(encryptor, registry)
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

		w := workers.NewNodeQueueWorker(registry)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WEBHOOK_PROVISIONER") == "yes" {
		log.Println("Starting Webhook Provisioner")

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

		w := workers.NewCanvasCleanupWorker()
		go w.Start(context.Background())
	}
}

func startEmailConsumers(rabbitMQURL string, encryptor crypto.Encryptor, baseURL string, authService authorization.Authorization) {
	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		log.Warn("Email Consumers not started - missing required environment variable (TEMPLATE_DIR)")
		return
	}

	if os.Getenv("OWNER_SETUP_ENABLED") == "yes" {
		log.Println("Starting SMTP Email Consumers (self-hosted)")
		settingsProvider := &services.DatabaseEmailSettingsProvider{Encryptor: encryptor}
		emailService := services.NewSMTPEmailService(settingsProvider, templateDir)
		startEmailConsumersWithService(rabbitMQURL, emailService, baseURL, authService)
		return
	}

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	fromName := os.Getenv("EMAIL_FROM_NAME")
	fromEmail := os.Getenv("EMAIL_FROM_ADDRESS")
	if resendAPIKey == "" || fromName == "" || fromEmail == "" {
		log.Warn("Email Consumers not started - missing required environment variables (RESEND_API_KEY, EMAIL_FROM_NAME, EMAIL_FROM_ADDRESS)")
		return
	}

	emailService := services.NewResendEmailService(resendAPIKey, fromName, fromEmail, templateDir)
	startEmailConsumersWithService(rabbitMQURL, emailService, baseURL, authService)
}

func startEmailConsumersWithService(rabbitMQURL string, emailService services.EmailService, baseURL string, authService authorization.Authorization) {
	log.Println("Starting Invitation Email Consumer")
	invitationEmailConsumer := workers.NewInvitationEmailConsumer(rabbitMQURL, emailService, baseURL)
	go invitationEmailConsumer.Start()

	log.Println("Starting Notification Email Consumer")
	notificationEmailConsumer := workers.NewNotificationEmailConsumer(rabbitMQURL, emailService, authService)
	go notificationEmailConsumer.Start()
}

func startInternalAPI(baseURL, webhooksBaseURL, basePath string, encryptor crypto.Encryptor, authService authorization.Authorization, registry *registry.Registry, oidcProvider oidc.Provider) {
	log.Println("Starting Internal API")
	grpc.RunServer(baseURL, webhooksBaseURL, basePath, encryptor, authService, registry, oidcProvider, lookupInternalAPIPort())
}

func startPublicAPI(baseURL, basePath string, encryptor crypto.Encryptor, registry *registry.Registry, jwtSigner *jwt.Signer, oidcProvider oidc.Provider, authService authorization.Authorization) {
	log.Println("Starting Public API with integrated Web Server")

	appEnv := os.Getenv("APP_ENV")
	templateDir := os.Getenv("TEMPLATE_DIR")
	blockSignup := os.Getenv("BLOCK_SIGNUP") == "yes"

	webhooksBaseURL := getWebhookBaseURL(baseURL)
	server, err := public.NewServer(encryptor, registry, jwtSigner, oidcProvider, basePath, baseURL, webhooksBaseURL, appEnv, templateDir, authService, blockSignup)
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

	if os.Getenv("START_GRPC_GATEWAY") == "yes" {
		log.Println("Adding gRPC Gateway to Public API")

		grpcServerAddr := os.Getenv("GRPC_SERVER_ADDR")
		if grpcServerAddr == "" {
			grpcServerAddr = "localhost:50051"
		}

		err := server.RegisterGRPCGateway(grpcServerAddr)
		if err != nil {
			log.Fatalf("Failed to register gRPC gateway: %v", err)
		}

		server.RegisterOpenAPIHandler()
	}

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

func lookupInternalAPIPort() int {
	port := 50051

	if p := os.Getenv("INTERNAL_API_PORT"); p != "" {
		if v, errConv := strconv.Atoi(p); errConv == nil && v > 0 {
			port = v
		} else {
			log.Warnf("Invalid INTERNAL_API_PORT %q, falling back to 50051", p)
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

func setupOtelMetrics() {
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
}

func Start() {
	configureLogging()
	setupOtelMetrics()

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

	jwtSigner := jwt.NewSigner(jwtSecret)
	webhooksBaseURL := getWebhookBaseURL(baseURL)
	oidcProvider, err := oidc.NewProviderFromKeyDir(webhooksBaseURL, oidcKeysPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load OIDC keys: %v", err))
	}

	registry, err := registry.NewRegistry(encryptorInstance, registry.HTTPOptions{
		BlockedHosts:     getBlockedHTTPHosts(),
		PrivateIPRanges:  getPrivateIPRanges(),
		MaxResponseBytes: DefaultMaxHTTPResponseBytes,
	})

	if err != nil {
		panic(fmt.Sprintf("failed to create registry: %v", err))
	}

	templates.Setup(registry)

	if os.Getenv("START_PUBLIC_API") == "yes" {
		go startPublicAPI(baseURL, basePath, encryptorInstance, registry, jwtSigner, oidcProvider, authService)
	}

	if os.Getenv("START_INTERNAL_API") == "yes" {
		go startInternalAPI(baseURL, webhooksBaseURL, basePath, encryptorInstance, authService, registry, oidcProvider)
	}

	startWorkers(encryptorInstance, registry, oidcProvider, baseURL, authService)

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
 * 512KB is the default maximum response size for HTTP responses.
 * This prevents component/trigger implementations from using too much memory,
 * and also from emitting large events.
 */
var DefaultMaxHTTPResponseBytes int64 = 512 * 1024

/*
 * Default blocked HTTP hosts include:
 * - Cloud metadata endpoints
 * - Kubernetes API
 * - Localhost variations
 */
var defaultBlockedHTTPHosts = []string{
	"metadata.google.internal",
	"metadata.goog",
	"metadata.azure.com",
	"169.254.169.254",
	"fd00:ec2::254",
	"kubernetes.default",
	"kubernetes.default.svc",
	"kubernetes.default.svc.cluster.local",
	"localhost",
	"127.0.0.1",
	"::1",
	"0.0.0.0",
	"::",
}

func getBlockedHTTPHosts() []string {
	blockedHosts := os.Getenv("BLOCKED_HTTP_HOSTS")
	if blockedHosts == "" {
		return defaultBlockedHTTPHosts
	}

	return strings.Split(blockedHosts, ",")
}

var defaultBlockedPrivateIPRanges = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

func getPrivateIPRanges() []string {
	blockedPrivateIPRanges := os.Getenv("BLOCKED_PRIVATE_IP_RANGES")
	if blockedPrivateIPRanges == "" {
		return defaultBlockedPrivateIPRanges
	}

	return strings.Split(blockedPrivateIPRanges, ",")
}
