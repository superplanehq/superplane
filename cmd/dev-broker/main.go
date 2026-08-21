// Command dev-broker runs a task broker for local development, so Runner
// components work without SuperPlane Cloud. See pkg/devbroker for the security
// caveats — it executes arbitrary shell commands and must stay on a private
// development network.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/superplanehq/superplane/pkg/devbroker"
)

func main() {
	// Deliberately not PORT: the dev container already sets that for the API
	// server, and reusing it makes the broker steal the API's port.
	port := os.Getenv("DEV_BROKER_PORT")
	if port == "" {
		port = "9000"
	}

	authToken := os.Getenv("TASK_BROKER_AUTH_TOKEN")
	if authToken == "" {
		log.Fatal("TASK_BROKER_AUTH_TOKEN is not set")
	}

	workDir := os.Getenv("TASK_BROKER_WORK_DIR")
	if workDir == "" {
		workDir = os.TempDir()
	}

	server := devbroker.New(devbroker.Options{AuthToken: authToken, WorkDir: workDir})

	log.Printf("dev broker listening on :%s, work dir %s", port, workDir)
	if err := http.ListenAndServe(":"+port, server.Handler()); err != nil {
		log.Fatalf("dev broker stopped: %v", err)
	}
}
