package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/superplanehq/superplane/pkg/runner"
	"github.com/superplanehq/superplane/pkg/runner/executors"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hubURL := os.Getenv("HUB_URL")
	registrationToken := os.Getenv("TOKEN")
	cacheDir := os.Getenv("CACHE_DIR")
	denoBinary := os.Getenv("DENO_BINARY")

	if hubURL == "" || registrationToken == "" {
		log.Fatalf("HUB_URL and TOKEN must be set")
	}

	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "superplane-runner-cache")
	}

	if denoBinary == "" {
		denoBinary = "deno"
	}

	log.Printf("Starting runner")
	log.Printf("> Hub URL: %s", hubURL)
	log.Printf("> Cache Dir: %s", cacheDir)
	log.Printf("> Deno location: %s", denoBinary)

	executor, err := executors.New(executors.Config{
		HubURL:     hubURL,
		CacheDir:   cacheDir,
		DenoBinary: denoBinary,
		Runner:     executors.ExecRunner{},
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	})

	if err != nil {
		log.Fatalf("Error creating runtime executor: %v", err)
	}

	config := runner.ClientConfig{
		HubURL:            hubURL,
		RegistrationToken: registrationToken,
		ReconnectDelay:    time.Second,
	}

	runner, err := runner.New(config, executor.HandleJob)
	if err != nil {
		log.Fatalf("Error creating runner: %v", err)
	}

	if err := runner.Run(ctx); err != nil {
		log.Printf("Error starting runner: %v", err)
		log.Fatal(err)
	}
}
