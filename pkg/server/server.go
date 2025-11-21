package server

import (
	"context"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	grpc "github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/public"
	registry "github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers"

	// Import components and triggers to register them via init()
	_ "github.com/superplanehq/superplane/pkg/components/approval"
	_ "github.com/superplanehq/superplane/pkg/components/filter"
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/components/if"
	_ "github.com/superplanehq/superplane/pkg/components/merge"
	_ "github.com/superplanehq/superplane/pkg/components/noop"
	_ "github.com/superplanehq/superplane/pkg/components/semaphore"
	_ "github.com/superplanehq/superplane/pkg/components/timegate"
	_ "github.com/superplanehq/superplane/pkg/components/wait"
	_ "github.com/superplanehq/superplane/pkg/triggers/github"
	_ "github.com/superplanehq/superplane/pkg/triggers/schedule"
	_ "github.com/superplanehq/superplane/pkg/triggers/semaphore"
	_ "github.com/superplanehq/superplane/pkg/triggers/start"
)

func startWorkers(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, registry *registry.Registry, baseURL string, authService authorization.Authorization) {
	log.Println("Starting Workers")

	rabbitMQURL, err := config.RabbitMQURL()
	if err != nil {
		panic(err)
	}

	if os.Getenv("START_CONSUMERS") == "yes" {
		log.Println("Starting Invitation Email Consumer")
		resendAPIKey := os.Getenv("RESEND_API_KEY")
		fromName := os.Getenv("EMAIL_FROM_NAME")
		fromEmail := os.Getenv("EMAIL_FROM_ADDRESS")
		templateDir := os.Getenv("TEMPLATE_DIR")

		if resendAPIKey != "" && fromName != "" && fromEmail != "" && templateDir != "" {
			emailService := services.NewResendEmailService(resendAPIKey, fromName, fromEmail, templateDir)
			invitationEmailConsumer := workers.NewInvitationEmailConsumer(rabbitMQURL, emailService, baseURL)
			go invitationEmailConsumer.Start()
		} else {
			log.Warn("Invitation Email Consumer not started - missing required environment variables (RESEND_API_KEY, EMAIL_FROM_NAME, EMAIL_FROM_ADDRESS, TEMPLATE_DIR)")
		}
	}

	if os.Getenv("START_WORKFLOW_EVENT_ROUTER") == "yes" {
		log.Println("Starting Execution Router")

		w := workers.NewWorkflowEventRouter()
		go w.Start(context.Background())
	}

	if os.Getenv("START_WORKFLOW_NODE_EXECUTOR") == "yes" {
		log.Println("Starting Pending Node Execution Worker")

		w := workers.NewWorkflowNodeExecutor(registry)
		go w.Start(context.Background())
	}

	if os.Getenv("START_NODE_REQUEST_WORKER") == "yes" {
		log.Println("Starting Node Request Worker")

		w := workers.NewNodeRequestWorker(registry)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WORKFLOW_NODE_QUEUE_WORKER") == "yes" {
		log.Println("Starting Workflow Node Queue Worker")

		w := workers.NewWorkflowNodeQueueWorker(registry)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WEBHOOK_PROVISIONER") == "yes" {
		log.Println("Starting Webhook Provisioner")

		w := workers.NewWebhookProvisioner(baseURL, encryptor, registry)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WEBHOOK_CLEANUP_WORKER") == "yes" {
		log.Println("Starting Webhook Cleanup Worker")

		w := workers.NewWebhookCleanupWorker(registry)
		go w.Start(context.Background())
	}

	if os.Getenv("START_WORKFLOW_CLEANUP_WORKER") == "yes" {
		log.Println("Starting Workflow Cleanup Worker")

		w := workers.NewWorkflowCleanupWorker()
		go w.Start(context.Background())
	}
}

func startInternalAPI(encryptor crypto.Encryptor, authService authorization.Authorization, registry *registry.Registry) {
	log.Println("Starting Internal API")
	grpc.RunServer(encryptor, authService, registry, lookupInternalAPIPort())
}

func startPublicAPI(encryptor crypto.Encryptor, registry *registry.Registry, jwtSigner *jwt.Signer, oidcVerifier *crypto.OIDCVerifier, authService authorization.Authorization) {
	log.Println("Starting Public API with integrated Web Server")

	basePath := os.Getenv("PUBLIC_API_BASE_PATH")
	if basePath == "" {
		panic("PUBLIC_API_BASE_PATH must be set")
	}

	appEnv := os.Getenv("APP_ENV")
	templateDir := os.Getenv("TEMPLATE_DIR")
	blockSignup := os.Getenv("BLOCK_SIGNUP") == "yes"

	server, err := public.NewServer(encryptor, registry, jwtSigner, oidcVerifier, basePath, appEnv, templateDir, authService, blockSignup)
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

	// Sync missing default roles on startup
	log.Info("Syncing default permissions for all organizations and canvases...")
	if err := authService.CheckAndSyncMissingPermissions(); err != nil {
		log.Warnf("Failed to sync missing permissions on startup: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET must be set")
	}

	jwtSigner := jwt.NewSigner(jwtSecret)
	oidcVerifier := crypto.NewOIDCVerifier()
	registry := registry.NewRegistry(encryptorInstance)

	if os.Getenv("START_PUBLIC_API") == "yes" {
		go startPublicAPI(encryptorInstance, registry, jwtSigner, oidcVerifier, authService)
	}

	if os.Getenv("START_INTERNAL_API") == "yes" {
		go startInternalAPI(encryptorInstance, authService, registry)
	}

	startWorkers(jwtSigner, encryptorInstance, registry, baseURL, authService)

	log.Println("Superplane is UP.")

	select {}
}
