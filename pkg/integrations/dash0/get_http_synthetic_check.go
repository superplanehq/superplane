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

type GetHTTPSyntheticCheck struct{}

type GetHTTPSyntheticCheckSpec struct {
	CheckID string `mapstructure:"checkId"`
	Dataset string `mapstructure:"dataset"`
}

type GetHTTPSyntheticCheckNodeMetadata struct {
	CheckName string `json:"checkName" mapstructure:"checkName"`
}

func (c *GetHTTPSyntheticCheck) Name() string {
	return "dash0.getHttpSyntheticCheck"
}

func (c *GetHTTPSyntheticCheck) Label() string {
	return "Get HTTP Synthetic Check"
}

func (c *GetHTTPSyntheticCheck) Description() string {
	return "Retrieve an HTTP synthetic check configuration and operational metrics from Dash0"
}

func (c *GetHTTPSyntheticCheck) Documentation() string {
	return `The Get HTTP Synthetic Check component retrieves the full configuration and operational metrics of an existing HTTP synthetic check from Dash0.

## Use Cases

- **Health dashboards**: Fetch current uptime and performance metrics for display in workflows
- **Audit and reporting**: Retrieve check configurations for compliance or documentation
- **Incident response**: Quickly gather check status and recent performance data during incidents

## Configuration

- **Check ID**: The ID of the synthetic check to retrieve (from Dash0)
- **Dataset**: The Dash0 dataset the check belongs to (defaults to "default")

## Output Channels

- **Healthy**: The check is passing — the most recent run outcome is "Healthy"
- **Critical**: The check is failing — the most recent run outcome is "Critical"

## Output

Returns a combined payload with:

### Configuration
The full synthetic check configuration from the Dash0 API, including:
- Name, URL, HTTP method
- Schedule (interval, locations, strategy)
- Assertions (critical and degraded thresholds)
- Retry settings

### Metrics
Operational metrics from the Dash0 Prometheus API:
- **Healthy Runs (24h/7d)**: Number of successful check runs
- **Critical Runs (24h/7d)**: Number of failed check runs
- **Total Runs (24h/7d)**: Total number of check runs
- **Avg Duration (24h/7d)**: Mean end-to-end response time (in milliseconds)
- **Last Outcome**: Most recent run outcome (Healthy or Critical)

Note: Metrics are fetched on a best-effort basis. If Prometheus metrics are unavailable for a check, the configuration is still returned with null metric values.`
}

func (c *GetHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (c *GetHTTPSyntheticCheck) Color() string {
	return "blue"
}

const ChannelNameHealthy = "healthy"

func (c *GetHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameHealthy, Label: "Healthy", Description: "The check is passing"},
		{Name: ChannelNameCritical, Label: "Critical", Description: "The check is failing"},
	}
}

func (c *GetHTTPSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkId",
			Label:       "Check ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The synthetic check to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "synthetic-check",
				},
			},
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset the synthetic check belongs to",
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
	if strings.TrimSpace(spec.Dataset) == "" {
		return errors.New("dataset is required")
	}

	// If metadata is already set, skip the API call.
	var nodeMetadata GetHTTPSyntheticCheckNodeMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("error decoding metadata: %v", err)
	}
	if nodeMetadata.CheckName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client during setup: %v", err)
	}

	checkConfig, err := client.GetSyntheticCheck(spec.CheckID, spec.Dataset)
	if err != nil {
		return fmt.Errorf("failed to get synthetic check during setup: %v", err)
	}

	checkName := checkConfig.Spec.Display.Name
	if checkName == "" {
		checkName = checkConfig.Metadata.Name
	}

	return ctx.Metadata.Set(GetHTTPSyntheticCheckNodeMetadata{
		CheckName: checkName,
	})
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

	// Fetch the check configuration from the Dash0 API.
	checkConfig, err := client.GetSyntheticCheck(spec.CheckID, dataset)
	if err != nil {
		return fmt.Errorf("failed to get synthetic check: %v", err)
	}

	// Fetch operational metrics from Prometheus (best-effort).
	metrics := FetchSyntheticCheckMetrics(ctx, client, dataset, spec.CheckID)

	// Determine the output channel from the last run outcome.
	channel := outcomeToChannel(metrics.LastOutcome)

	// If no outcome data is available, pass without emitting.
	if channel == "" {
		return ctx.ExecutionState.Pass()
	}

	output := map[string]any{
		"configuration": checkConfig,
		"metrics":       metrics,
	}

	return ctx.ExecutionState.Emit(
		channel,
		"dash0.syntheticCheck.fetched",
		[]any{output},
	)
}

// outcomeToChannel maps a dash0.synthetic_check.outcome value to an output channel name.
// Returns an empty string if the outcome is not recognized or empty.
func outcomeToChannel(outcome string) string {
	switch outcome {
	case "Healthy":
		return ChannelNameHealthy
	case "Critical":
		return ChannelNameCritical
	default:
		return ""
	}
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
