package render

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	GetServicePayloadType    = "render.get.service"
	GetServiceOutputChannel  = "default"
)

type GetService struct{}

type GetServiceConfiguration struct {
	ServiceID string `json:"serviceId" mapstructure:"serviceId"`
}

type ServiceResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Slug          string `json:"slug"`
	Suspended     string `json:"suspended"`
	AutoDeploy    string `json:"autoDeploy"`
	ServiceDetails *json.RawMessage `json:"serviceDetails,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

func (c *GetService) Name() string {
	return "render.getService"
}

func (c *GetService) Label() string {
	return "Get Service"
}

func (c *GetService) Description() string {
	return "Fetch details of a Render service by ID"
}

func (c *GetService) Documentation() string {
	return `The Get Service component fetches the details of a Render service by its ID.

## Use Cases

- **Inspect service state**: Retrieve the current status and configuration of a service
- **Conditional logic**: Branch pipeline based on service properties (suspended, type, etc.)

## Configuration

- **Service ID**: The Render service ID to fetch (e.g. ` + "`srv-...`" + `)

## Output

Emits the full service object returned by the Render API on the default channel.`
}

func (c *GetService) Icon() string {
	return "search"
}

func (c *GetService) Color() string {
	return "gray"
}

func (c *GetService) ExampleOutput() map[string]any {
	return nil
}

func (c *GetService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: GetServiceOutputChannel, Label: "Default"},
	}
}

func (c *GetService) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "serviceId",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service to fetch",
		},
	}
}

func decodeGetServiceConfiguration(configuration any) (GetServiceConfiguration, error) {
	spec := GetServiceConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetServiceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ServiceID = strings.TrimSpace(spec.ServiceID)
	if spec.ServiceID == "" {
		return GetServiceConfiguration{}, fmt.Errorf("serviceId is required")
	}

	return spec, nil
}

func (c *GetService) Setup(ctx core.SetupContext) error {
	_, err := decodeGetServiceConfiguration(ctx.Configuration)
	return err
}

func (c *GetService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetService) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetServiceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	service, err := client.GetService(spec.ServiceID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"serviceId": service.ID,
		"name":      service.Name,
		"type":      service.Type,
		"suspended": service.Suspended,
		"createdAt": service.CreatedAt,
		"updatedAt": service.UpdatedAt,
	}

	return ctx.ExecutionState.Emit(GetServiceOutputChannel, GetServicePayloadType, []any{payload})
}

func (c *GetService) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetService) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *GetService) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (c *GetService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetService) Cleanup(ctx core.SetupContext) error {
	return nil
}

// Client method for GetService
func (cl *Client) GetService(serviceID string) (ServiceResponse, error) {
	if serviceID == "" {
		return ServiceResponse{}, fmt.Errorf("serviceID is required")
	}

	_, body, err := cl.execRequestWithResponse(
		"GET",
		"/services/"+url.PathEscape(serviceID),
		nil,
		nil,
	)
	if err != nil {
		return ServiceResponse{}, err
	}

	response := ServiceResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return ServiceResponse{}, fmt.Errorf("failed to unmarshal service response: %w", err)
	}

	return response, nil
}
