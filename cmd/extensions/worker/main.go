package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	worker "github.com/superplanehq/superplane/pkg/extensions/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hubURL := os.Getenv("EXTENSION_WORKER_HUB_URL")
	registrationToken := os.Getenv("EXTENSION_WORKER_REGISTRATION_TOKEN")
	cacheDir := os.Getenv("EXTENSION_WORKER_CACHE_DIR")
	denoBinary := os.Getenv("DENO_BINARY")

	if hubURL == "" || registrationToken == "" {
		log.Fatalf("EXTENSION_WORKER_HUB_URL and EXTENSION_WORKER_REGISTRATION_TOKEN must be set")
	}

	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "superplane-extension-worker-cache")
	}

	if denoBinary == "" {
		denoBinary = "deno"
	}

	log.Printf("Starting extension worker")
	log.Printf("> Hub URL: %s", hubURL)
	log.Printf("> Cache Dir: %s", cacheDir)
	log.Printf("> Deno location: %s", denoBinary)

	executor := worker.NewRuntimeExecutor(worker.RuntimeExecutorConfig{
		HubURL:     hubURL,
		CacheDir:   cacheDir,
		DenoBinary: denoBinary,
	})

	client := worker.NewClient(worker.ClientConfig{
		HubURL:            hubURL,
		RegistrationToken: registrationToken,
		ReconnectDelay:    time.Second,
	}, executor.HandleJob)

	if err := client.Run(ctx); err != nil {
		log.Printf("Error running extension worker: %v", err)
		log.Fatal(err)
	}
}
