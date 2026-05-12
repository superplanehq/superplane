package restate

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type HealthCheck struct{}

func (c *HealthCheck) Name() string {
	return "restate.healthCheck"
}

func (c *HealthCheck) Label() string {
	return "Health Check"
}

func (c *HealthCheck) Description() string {
	return "Check the health status of the Restate cluster"
}

func (c *HealthCheck) Icon() string {
	return "repeat"
}

func (c *HealthCheck) Color() string {
	return "gray"
}

func (c *HealthCheck) Documentation() string {
	return `The Health Check component verifies the Restate cluster is healthy and operational.

## Use Cases

- **Pre-deploy gate**: Verify the cluster is healthy before registering a new deployment
- **Post-deploy validation**: Confirm the cluster is still healthy after a deployment
- **Monitoring workflows**: Periodic health checks as part of operational workflows
- **Incident investigation**: Check cluster health during incident triage

## Outputs

The component emits an event containing:
- ` + "`healthy`" + `: Boolean indicating if the basic health check passed
- ` + "`cluster_health`" + `: Detailed cluster health information
- ` + "`version`" + `: Restate server version information
`
}

func (c *HealthCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *HealthCheck) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *HealthCheck) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *HealthCheck) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Basic health check
	healthErr := client.CheckHealth()
	healthy := healthErr == nil

	result := map[string]any{
		"healthy": healthy,
	}

	if !healthy {
		result["error"] = healthErr.Error()
		return ctx.ExecutionState.Fail("unhealthy", fmt.Sprintf("Restate cluster health check failed: %v", healthErr))
	}

	// Get detailed cluster health
	clusterHealth, err := client.GetClusterHealth()
	if err == nil {
		result["cluster_health"] = clusterHealth
	}

	// Get version info
	version, err := client.GetVersion()
	if err == nil {
		result["version"] = version
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.health",
		[]any{result},
	)
}

func (c *HealthCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *HealthCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *HealthCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *HealthCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *HealthCheck) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *HealthCheck) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
