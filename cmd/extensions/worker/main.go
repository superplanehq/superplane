package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	worker "github.com/superplanehq/superplane/pkg/extensions/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	executor := worker.NewRuntimeExecutor(worker.RuntimeExecutorConfig{
		HubURL:     requiredEnv("EXTENSION_WORKER_HUB_URL"),
		CacheDir:   os.Getenv("EXTENSION_WORKER_CACHE_DIR"),
		DenoBinary: os.Getenv("DENO_BINARY"),
	})

	client := worker.NewClient(worker.ClientConfig{
		HubURL:            requiredEnv("EXTENSION_WORKER_HUB_URL"),
		WorkerID:          requiredEnv("EXTENSION_WORKER_ID"),
		RegistrationToken: requiredEnv("EXTENSION_WORKER_REGISTRATION_TOKEN"),
	}, executor.HandleJob)

	if err := client.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("missing required environment variable %s", name)
	}

	return value
}
