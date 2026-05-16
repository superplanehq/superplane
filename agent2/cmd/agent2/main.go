package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/superplanehq/superplane/agent2/internal/anthropic"
	agentgrpc "github.com/superplanehq/superplane/agent2/internal/grpc"
	"github.com/superplanehq/superplane/agent2/internal/store"
	"github.com/superplanehq/superplane/agent2/internal/stream"

	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	cfg := loadConfig()

	db, err := store.New(cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	client := anthropic.NewClient(cfg.AnthropicAPIKey)

	svc := agentgrpc.NewService(agentgrpc.ServiceConfig{
		Client:        client,
		Store:         db,
		AgentID:       cfg.AnthropicAgentID,
		EnvironmentID: cfg.AnthropicEnvironmentID,
	})

	// gRPC server
	grpcServer := grpc.NewServer()
	internalpb.RegisterAgentsServer(grpcServer, svc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", cfg.GRPCPort, err)
	}

	go func() {
		log.Infof("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// HTTP server for SSE streaming
	streamHandler := stream.NewHandler(stream.HandlerConfig{
		Client:    client,
		Store:     db,
		JWTSecret: cfg.JWTSecret,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/agents/chats/", streamHandler.HandleStream)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	go func() {
		log.Infof("HTTP server listening on :%d", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}

type config struct {
	AnthropicAPIKey        string
	AnthropicAgentID       string
	AnthropicEnvironmentID string
	DBURL                  string
	GRPCPort               int
	HTTPPort               int
	JWTSecret              string
}

func loadConfig() config {
	return config{
		AnthropicAPIKey:        mustEnv("ANTHROPIC_API_KEY"),
		AnthropicAgentID:       mustEnv("ANTHROPIC_AGENT_ID"),
		AnthropicEnvironmentID: getEnv("ANTHROPIC_ENVIRONMENT_ID", ""),
		DBURL:                  mustEnv("DB_URL"),
		GRPCPort:               getEnvInt("GRPC_PORT", 50061),
		HTTPPort:               getEnvInt("HTTP_PORT", 8090),
		JWTSecret:              mustEnv("JWT_SECRET"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var i int
	fmt.Sscanf(v, "%d", &i)
	if i == 0 {
		return fallback
	}
	return i
}
