package runner

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/runners"
)

// BrokerEnvironmentVariable is forwarded to fleet-manager as JSON.
// Kept for backward compatibility with commands.go resolveEnvironment.
type BrokerEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// legacyFleetFromEnv builds a RunnerFleet from the legacy TASK_BROKER_* env vars.
// This supports installations that have not yet registered their fleet via the admin API.
func legacyFleetFromEnv() (*runners.RunnerFleet, error) {
	baseURL := os.Getenv("TASK_BROKER_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("no fleet configured: set fleet_id in the node spec or TASK_BROKER_BASE_URL env var")
	}

	fleetID := os.Getenv("TASK_BROKER_FLEET_ID")
	if fleetID == "" {
		return nil, fmt.Errorf("TASK_BROKER_FLEET_ID is not set")
	}

	authToken := os.Getenv("TASK_BROKER_AUTH_TOKEN")
	if authToken == "" {
		return nil, fmt.Errorf("TASK_BROKER_AUTH_TOKEN is not set")
	}

	// Return an ephemeral fleet — it is never persisted to the DB.
	// The FleetID field stores the fleet_id string expected by the broker API,
	// placed in the Name for reference. The actual UUID is generated on the fly
	// but never saved (the legacy broker does not need a runner_tasks record).
	return &runners.RunnerFleet{
		FleetURL:  baseURL,
		AuthToken: authToken,
		// Name carries the broker fleet_id used in API requests.
		Name: fleetID,
	}, nil
}
