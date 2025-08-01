package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	grpc "github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/public"
	registry "github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers"
)

func startWorkers(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, registry *registry.Registry, baseURL string) {
	log.Println("Starting Workers")

	rabbitMQURL, err := config.RabbitMQURL()
	if err != nil {
		panic(err)
	}

	if os.Getenv("START_PENDING_EVENTS_WORKER") == "yes" {
		log.Println("Starting Pending Events Worker")
		w := workers.NewPendingEventsWorker(encryptor, registry)
		go w.Start()
	}

	if os.Getenv("START_PENDING_STAGE_EVENTS_WORKER") == "yes" {
		log.Println("Starting Pending Stage Events Worker")
		w, err := workers.NewPendingStageEventsWorker(time.Now)
		if err != nil {
			panic(err)
		}

		go w.Start()
	}

	if os.Getenv("START_TIME_WINDOW_WORKER") == "yes" {
		log.Println("Starting Time Window Worker")
		w, err := workers.NewTimeWindowWorker(time.Now)
		if err != nil {
			panic(err)
		}

		go w.Start()
	}

	if os.Getenv("START_STAGE_EVENT_APPROVED_CONSUMER") == "yes" {
		log.Println("Starting Stage Event Approved Consumer")
		w := workers.NewStageEventApprovedConsumer(rabbitMQURL)
		go w.Start()
	}

	if os.Getenv("START_EXECUTIONS_POLLER") == "yes" {
		log.Println("Starting Executions Poller")
		log.Println("Starting Execution Resources Poller")

		executionPoller := workers.NewExecutionPoller(encryptor)
		go executionPoller.Start()

		resourcePoller := workers.NewExecutionResourcePoller(encryptor, registry)
		go resourcePoller.Start()
	}

	if os.Getenv("START_PENDING_EXECUTIONS_WORKER") == "yes" {
		log.Println("Starting Pending Stage Events Worker")

		w := workers.PendingExecutionsWorker{
			JwtSigner:   jwtSigner,
			Encryptor:   encryptor,
			SpecBuilder: executors.SpecBuilder{},
			Registry:    registry,
		}

		go w.Start()
	}

	if os.Getenv("START_PENDING_FIELD_SETS_WORKER") == "yes" {
		log.Println("Starting Pending Field Sets Worker")

		w, err := workers.NewPendingFieldSetsWorker(time.Now)
		if err != nil {
			panic(err)
		}

		go w.Start()
	}

	if os.Getenv("START_PENDING_EVENT_SOURCES_WORKER") == "yes" {
		log.Println("Starting Pending Event Sources Worker")

		w, err := workers.NewPendingEventSourcesWorker(encryptor, registry, baseURL)
		if err != nil {
			panic(err)
		}

		go w.Start()
	}
}

func startInternalAPI(encryptor crypto.Encryptor, authService authorization.Authorization, registry *registry.Registry) {
	log.Println("Starting Internal API")
	grpc.RunServer(encryptor, authService, registry, 50051)
}

func startPublicAPI(encryptor crypto.Encryptor, registry *registry.Registry, jwtSigner *jwt.Signer, oidcVerifier *crypto.OIDCVerifier) {
	log.Println("Starting Public API with integrated Web Server")

	basePath := os.Getenv("PUBLIC_API_BASE_PATH")
	if basePath == "" {
		panic("PUBLIC_API_BASE_PATH must be set")
	}

	appEnv := os.Getenv("APP_ENV")

	server, err := public.NewServer(encryptor, registry, jwtSigner, oidcVerifier, basePath, appEnv)
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

	// Register web routes only if START_WEB_SERVER is set to "yes"
	if os.Getenv("START_WEB_SERVER") == "yes" {
		webBasePath := os.Getenv("WEB_BASE_PATH")
		if webBasePath == "" {
			log.Warn("WEB_BASE_PATH is not set, defaulting to /app")
			webBasePath = "/app"
		}
		log.Printf("Registering web routes in public API server with base path: %s", webBasePath)
		server.RegisterWebRoutes(webBasePath)
	} else {
		log.Println("Web server routes not registered (START_WEB_SERVER != yes)")
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

	err = server.Serve("0.0.0.0", 8000)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.StampMilli,
	})

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
		go startPublicAPI(encryptorInstance, registry, jwtSigner, oidcVerifier)
	}

	if os.Getenv("START_INTERNAL_API") == "yes" {
		go startInternalAPI(encryptorInstance, authService, registry)
	}

	startWorkers(jwtSigner, encryptorInstance, registry, baseURL)

	log.Println("Superplane is UP.")

	select {}
}
