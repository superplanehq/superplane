package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/superplanehq/superplane/pkg/jwt"
)

// Script to generate a runner registration token for a given organization, pool, and runner.
// Use environment variables to pass information.
func main() {
	organizationID := os.Getenv("ORGANIZATION_ID")
	poolID := os.Getenv("POOL_ID")
	runnerID := os.Getenv("RUNNER_ID")

	if organizationID == "" || poolID == "" || runnerID == "" {
		fmt.Println("ORGANIZATION_ID, POOL_ID, and RUNNER_ID must be set")
		os.Exit(1)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		fmt.Println("JWT_SECRET is not set")
		os.Exit(1)
	}

	log.Printf("Generating registration token for organization %s, pool %s, runner %s", organizationID, poolID, runnerID)

	signer := jwt.NewSigner(secret)
	token, err := signer.GenerateWithClaims(runnerID, 24*time.Hour, map[string]any{
		"organizationId": organizationID,
		"poolId":         poolID,
		"runnerId":       runnerID,
	})

	if err != nil {
		fmt.Printf("Error generating registration token: %v", err)
		os.Exit(1)
	}

	fmt.Println(token)
}
