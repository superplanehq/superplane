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

type DeleteHTTPSyntheticCheck struct{}

type DeleteHTTPSyntheticCheckSpec struct {
	CheckID string `mapstructure:"checkId"`
	Dataset string `mapstructure:"dataset"`
}

func (c *DeleteHTTPSyntheticCheck) Name() string {
	return "dash0.deleteHttpSyntheticCheck"
}

func (c *DeleteHTTPSyntheticCheck) Label() string {
	return "Delete HTTP Synthetic Check"
}

func (c *DeleteHTTPSyntheticCheck) Description() string {
	return "Delete an HTTP synthetic check from Dash0 by ID"
}

func (c *DeleteHTTPSyntheticCheck) Documentation() string {
	return `The Delete HTTP Synthetic Check component removes a synthetic check from Dash0 by its ID. Use the check ID from a Create/Get/Update output (e.g. metadata.labels["dash0.com/id"]) or from the Dash0 dashboard.

## Configuration

- **Check ID**: The Dash0 synthetic check ID to delete (required).
- **Dataset**: The dataset the check belongs to (defaults to "default").

## Output

Returns a confirmation payload (e.g. deleted id).`
}

func (c *DeleteHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (c *DeleteHTTPSyntheticCheck) Color() string {
	return "red"
}

func (c *DeleteHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteHTTPSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkId",
			Label:       "Check ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The synthetic check to delete",
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
			Description: "The dataset the check belongs to",
		},
	}
}

func (c *DeleteHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := DeleteHTTPSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.CheckID) == "" {
		return errors.New("checkId is required")
	}

	return nil
}

func (c *DeleteHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := DeleteHTTPSyntheticCheckSpec{}
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

	data, err := client.DeleteSyntheticCheck(spec.CheckID, dataset)
	if err != nil {
		return fmt.Errorf("failed to delete synthetic check: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.syntheticCheck.deleted",
		[]any{data},
	)
}

func (c *DeleteHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteHTTPSyntheticCheck) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteHTTPSyntheticCheck) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
