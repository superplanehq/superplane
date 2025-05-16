package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/encryptor"
	grpc "github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/public"
	"github.com/superplanehq/superplane/pkg/web"
	"github.com/superplanehq/superplane/pkg/workers"
)

func startWorkers(jwtSigner *jwt.Signer, encryptor encryptor.Encryptor) {
	log.Println("Starting Workers")

	rabbitMQURL, err := config.RabbitMQURL()
	if err != nil {
		panic(err)
	}

	if os.Getenv("START_PENDING_EVENTS_WORKER") == "yes" {
		log.Println("Starting Pending Events Worker")
		w := workers.PendingEventsWorker{}
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

		w := workers.NewExecutionPoller(encryptor)
		go w.Start()
	}

	if os.Getenv("START_PENDING_EXECUTIONS_WORKER") == "yes" {
		log.Println("Starting Pending Stage Events Worker")

		w := workers.PendingExecutionsWorker{JwtSigner: jwtSigner, Encryptor: encryptor}
		go w.Start()
	}
}

func startInternalAPI(encryptor encryptor.Encryptor) {
	log.Println("Starting Internal API")
	grpc.RunServer(encryptor, 50051)
}

func startPublicAPI(encryptor encryptor.Encryptor, jwtSigner *jwt.Signer) {
	log.Println("Starting Public API")

	basePath := os.Getenv("PUBLIC_API_BASE_PATH")
	if basePath == "" {
		panic("PUBLIC_API_BASE_PATH must be set")
	}

	server, err := public.NewServer(encryptor, jwtSigner, basePath)
	if err != nil {
		log.Panicf("Error creating public API server: %v", err)
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

func startWebServer() {
	log.Println("Starting Web Server")

	basePath := os.Getenv("WEB_BASE_PATH")
	if basePath == "" {
		panic("WEB_BASE_PATH must be set")
	}

	server, err := web.NewServer(basePath)
	if err != nil {
		log.Panicf("Error creating web server: %v", err)
	}

	err = server.Serve("0.0.0.0", 4000)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFormatter(&log.TextFormatter{TimestampFormat: time.StampMilli})

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		panic("ENCRYPTION_KEY can't be empty")
	}

	encryptor := encryptor.NewAESGCMEncryptor([]byte(encryptionKey))

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		panic("JWT_SECRET must be set")
	}

	jwtSigner := jwt.NewSigner(jwtSecret)

	if os.Getenv("START_PUBLIC_API") == "yes" {
		go startPublicAPI(encryptor, jwtSigner)
	}

	if os.Getenv("START_INTERNAL_API") == "yes" {
		go startInternalAPI(encryptor)
	}

	if os.Getenv("START_WEB_SERVER") == "yes" {
		go startWebServer()
	}

	startWorkers(jwtSigner, encryptor)

	log.Println("Superplane is UP.")

	select {}
}
