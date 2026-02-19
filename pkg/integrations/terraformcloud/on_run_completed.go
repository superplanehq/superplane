package terraformcloud

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnRunCompleted struct{}

type OnRunCompletedMetadata struct {
	WorkspaceID   string `json:"workspaceId"`
	WorkspaceName string `json:"workspaceName"`
	Organization  string `json:"organization"`
}

type OnRunCompletedConfiguration struct {
	Organization string `json:"organization"`
	WorkspaceID  string `json:"workspaceId"`
}

func (t *OnRunCompleted) Name() string {
	return "terraformcloud.onRunCompleted"
}

func (t *OnRunCompleted) Label() string {
	return "On Run Completed"
}

func (t *OnRunCompleted) Description() string {
	return "Listen to Terraform Cloud run completion events"
}

func (t *OnRunCompleted) Documentation() string {
	return `Triggers when a Terraform Cloud run reaches a terminal state (success or failure).

## Use Cases

- **Post-apply automation**: Run follow-up tasks after a Terraform apply completes
- **Status monitoring**: Monitor infrastructure changes and notify on failures
- **Workflow chaining**: Start SuperPlane workflows based on Terraform run outcomes
- **Notifications**: Send alerts when Terraform runs succeed or fail

## Configuration

- **Organization**: Your Terraform Cloud organization name
- **Workspace**: The workspace to monitor for run completions

## Event Data

Each run completion event includes:
- **run_id**: The ID of the completed run
- **run_status**: The final status (applied, errored, etc.)
- **run_message**: The message associated with the run
- **workspace_id**: The workspace ID
- **workspace_name**: The workspace name
- **organization_name**: The organization name
- **run_url**: Direct link to the run in Terraform Cloud

## Webhook Setup

This trigger automatically sets up a Terraform Cloud notification configuration when configured. The notification is managed by SuperPlane and cleaned up when the trigger is removed.`
}

func (t *OnRunCompleted) Icon() string {
	return "cloud"
}

func (t *OnRunCompleted) Color() string {
	return "purple"
}

func (t *OnRunCompleted) ExampleData() map[string]any {
	return exampleDataOnRunCompleted()
}

func (t *OnRunCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Terraform Cloud organization name",
			Placeholder: "my-organization",
		},
		{
			Name:        "workspaceId",
			Label:       "Workspace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The workspace to monitor for run completions",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeWorkspace,
					Parameters: []configuration.ParameterRef{
						{
							Name: "organization",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "organization",
							},
						},
					},
				},
			},
		},
	}
}

func (t *OnRunCompleted) Setup(ctx core.TriggerContext) error {
	var metadata OnRunCompletedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := OnRunCompletedConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Organization == "" {
		return fmt.Errorf("organization is required")
	}

	if config.WorkspaceID == "" {
		return fmt.Errorf("workspace is required")
	}

	workspaceChanged := metadata.WorkspaceID != config.WorkspaceID ||
		metadata.Organization != config.Organization

	if workspaceChanged {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		workspace, err := client.GetWorkspace(config.WorkspaceID)
		if err != nil {
			return fmt.Errorf("workspace not found or inaccessible: %w", err)
		}

		err = ctx.Metadata.Set(OnRunCompletedMetadata{
			WorkspaceID:   workspace.ID,
			WorkspaceName: workspace.Attributes.Name,
			Organization:  config.Organization,
		})
		if err != nil {
			return fmt.Errorf("error setting metadata: %v", err)
		}
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		WorkspaceID: config.WorkspaceID,
		Triggers:    []string{"run:completed", "run:errored"},
	})
}

func (t *OnRunCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnRunCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnRunCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Tfe-Notification-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := verifyTFCSignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	notifications, ok := data["notifications"].([]any)
	if !ok || len(notifications) == 0 {
		return http.StatusOK, nil
	}

	firstNotification, ok := notifications[0].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid notification format")
	}

	trigger, _ := firstNotification["trigger"].(string)
	if trigger != "run:completed" && trigger != "run:errored" {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit("terraformcloud.run.completed", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnRunCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func verifyTFCSignature(key []byte, data []byte, signature string) error {
	h := hmac.New(sha512.New, key)
	h.Write(data)
	computed := fmt.Sprintf("%x", h.Sum(nil))
	if computed != signature {
		return fmt.Errorf("invalid signature")
	}
	return nil
}
