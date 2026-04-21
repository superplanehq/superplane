package compute

import (
	"context"
	"fmt"

	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetInstance struct{}

func (c *GetInstance) Name() string {
	return "getInstance"
}

func (c *GetInstance) Label() string {
	return "Get Instance"
}

func (c *GetInstance) Description() string {
	return "Retrieve details of an OCI Compute instance"
}

func (c *GetInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "id",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to retrieve",
		},
	}
}

func (c *GetInstance) Execute(ctx core.ExecutionContext) error {
	var input struct {
		ID string `mapstructure:"id"`
	}

	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get OCI client: %v", err))
	}

	resp, err := client.GetInstance(ctx.Context, ocicore.GetInstanceRequest{
		InstanceId: common.String(input.ID),
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get OCI instance: %v", err))
	}

	payload := map[string]interface{}{
		"id":           *resp.Instance.Id,
		"displayName":  *resp.Instance.DisplayName,
		"state":        string(resp.Instance.LifecycleState),
		"shape":        *resp.Instance.Shape,
		"region":       *resp.Instance.Region,
		"timeCreated":  resp.Instance.TimeCreated.String(),
	}

	return ctx.ExecutionState.Emit("success", "oci.instance", []any{payload})
}

func (c *GetInstance) Setup(ctx core.SetupContext) error { return nil }
func (c *GetInstance) Cleanup(ctx core.SetupContext) error { return nil }
func (c *GetInstance) Actions() []core.Action { return nil }
func (c *GetInstance) HandleAction(ctx core.ActionContext) error { return nil }
func (c *GetInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *GetInstance) Cancel(ctx core.ExecutionContext) error { return nil }
