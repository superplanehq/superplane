package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// syntheticCheckMetricsToMap converts metrics to a map for JSON output (definition + metrics payload).
func syntheticCheckMetricsToMap(m *SyntheticCheckMetrics) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any)
	if m.Uptime24hPct != nil {
		out["uptime24hPct"] = *m.Uptime24hPct
	}
	if m.Uptime7dPct != nil {
		out["uptime7dPct"] = *m.Uptime7dPct
	}
	if m.AvgDuration7dMs != nil {
		out["avgDuration7dMs"] = *m.AvgDuration7dMs
	}
	if m.Fails7d != nil {
		out["fails7d"] = *m.Fails7d
	}
	if m.LastCheckAt != nil {
		out["lastCheckAt"] = *m.LastCheckAt
	}
	if m.DownFor7dSec != nil {
		out["downFor7dSec"] = *m.DownFor7dSec
	}
	if m.Status != nil {
		out["status"] = *m.Status
	}
	return out
}

type GetHTTPSyntheticCheck struct{}

type GetHTTPSyntheticCheckSpec struct {
	CheckID string `mapstructure:"checkId"`
	Dataset string `mapstructure:"dataset"`
}

func (c *GetHTTPSyntheticCheck) Name() string {
	return "dash0.getHttpSyntheticCheck"
}

func (c *GetHTTPSyntheticCheck) Label() string {
	return "Get HTTP Synthetic Check"
}

func (c *GetHTTPSyntheticCheck) Description() string {
	return "Fetch an existing HTTP synthetic check from Dash0 by ID"
}

func (c *GetHTTPSyntheticCheck) Documentation() string {
	return `The Get HTTP Synthetic Check component retrieves a synthetic check from Dash0 by its ID. Use the check ID from a Create/Update output (e.g. metadata.labels["dash0.com/id"]) or from the Dash0 dashboard.

## Configuration

- **Check ID**: The Dash0 synthetic check ID to fetch (required).
- **Dataset**: The dataset the check belongs to (defaults to "default").

## Output

Returns the synthetic check definition and, when available, key metrics (uptime, duration, fails, status) from Dash0 Prometheus. Payload shape: { definition, metrics? }.`
}

func (c *GetHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (c *GetHTTPSyntheticCheck) Color() string {
	return "slate"
}

func (c *GetHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetHTTPSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkId",
			Label:       "Check ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Dash0 synthetic check ID to fetch",
			Placeholder: "64617368-3073-796e-7468-abc123def456",
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset the check belongs to",
		},
	}
}

func (c *GetHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := GetHTTPSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.CheckID) == "" {
		return errors.New("checkId is required")
	}

	return nil
}

func (c *GetHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := GetHTTPSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	dataset := spec.Dataset
	if dataset == "" {
		dataset = "default"
	}

	definition, err := client.GetSyntheticCheck(spec.CheckID, dataset)
	if err != nil {
		return fmt.Errorf("failed to get synthetic check: %v", err)
	}

	// Optionally fetch key metrics from Prometheus (uptime, duration, fails, etc.)
	metrics, _ := client.GetSyntheticCheckMetrics(spec.CheckID, dataset)
	payload := map[string]any{"definition": definition}
	if metrics != nil {
		payload["metrics"] = syntheticCheckMetricsToMap(metrics)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.syntheticCheck.retrieved",
		[]any{payload},
	)
}

func (c *GetHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetHTTPSyntheticCheck) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetHTTPSyntheticCheck) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
