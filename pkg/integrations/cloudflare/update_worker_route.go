package cloudflare

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

type UpdateWorkerRoute struct{}

type UpdateWorkerRouteSpec struct {
	AccountID    string `json:"accountId"`
	Zone         string `json:"zone"`
	RouteID      string `json:"routeId"`
	Pattern      string `json:"pattern"`
	WorkerScript string `json:"workerScript"`
}

func (u *UpdateWorkerRoute) Name() string {
	return "cloudflare.updateWorkerRoute"
}

func (u *UpdateWorkerRoute) Label() string {
	return "Update Worker Route"
}

func (u *UpdateWorkerRoute) Description() string {
	return "Create or update a zone route that maps a URL pattern to a Worker script"
}

func (u *UpdateWorkerRoute) Documentation() string {
	return `The Update Worker Route component manages **zone routes** for Workers.

## Create vs update

- Leave **Route ID** empty to **create** a new route (` + "`POST /zones/{zone}/workers/routes`" + `).
- Set **Route ID** to **update** an existing route (` + "`PUT /zones/{zone}/workers/routes/{id}`" + `).

## Configuration

- **Zone**: Cloudflare zone for the route.
- **Pattern**: URL pattern (for example ` + "`example.com/*`" + `).
- **Worker Script**: Worker script invoked for matching traffic (picker lists scripts for the account).

## Output

Emits the route id, pattern, and script returned by the Cloudflare API.`
}

func (u *UpdateWorkerRoute) Icon() string {
	return "cloud"
}

func (u *UpdateWorkerRoute) Color() string {
	return "orange"
}

func (u *UpdateWorkerRoute) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateWorkerRoute) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Cloudflare zone where the route is created or updated",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "zone",
				},
			},
		},
		{
			Name:        "routeId",
			Label:       "Route ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Existing route ID to update; leave empty to create a new route",
			Placeholder: "023e105f4ecef8ad9ca31a8372d0c353",
		},
		{
			Name:        "pattern",
			Label:       "URL pattern",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Pattern to match incoming requests (for example example.com/*)",
			Placeholder: "example.com/*",
		},
		{
			Name:        "workerScript",
			Label:       "Worker Script",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Worker Script invoked when the route matches",
			Placeholder: "Select a Worker script",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "workerScript",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "accountId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "accountId"},
						},
					},
				},
			},
		},
	}
}

func (u *UpdateWorkerRoute) Setup(ctx core.SetupContext) error {
	spec := UpdateWorkerRouteSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Zone == "" {
		return errors.New("zone is required")
	}

	if strings.TrimSpace(spec.Pattern) == "" {
		return errors.New("pattern is required")
	}

	workerScript := strings.TrimSpace(spec.WorkerScript)
	if workerScript == "" {
		return errors.New("workerScript is required")
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	return resolveWorkerScriptMetadata(ctx, accountID, workerScript)
}

func (u *UpdateWorkerRoute) Execute(ctx core.ExecutionContext) error {
	spec := UpdateWorkerRouteSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Zone == "" {
		return errors.New("zone is required")
	}

	pattern := strings.TrimSpace(spec.Pattern)
	workerScript := strings.TrimSpace(spec.WorkerScript)
	if pattern == "" {
		return errors.New("pattern is required")
	}
	if workerScript == "" {
		return errors.New("workerScript is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	routeID := strings.TrimSpace(spec.RouteID)
	var route *WorkerRoute
	if routeID == "" {
		route, err = client.CreateWorkerRoute(spec.Zone, pattern, workerScript)
		if err != nil {
			return fmt.Errorf("failed to create worker route: %w", err)
		}
	} else {
		route, err = client.UpdateWorkerRoute(spec.Zone, routeID, pattern, workerScript)
		if err != nil {
			return fmt.Errorf("failed to update worker route: %w", err)
		}
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)

	result := map[string]any{
		"accountId": accountID,
		"zoneId":    spec.Zone,
		"route": map[string]any{
			"id":      route.ID,
			"pattern": route.Pattern,
			"script":  route.Script,
		},
	}

	eventType := "cloudflare.workerRoute.created"
	if routeID != "" {
		eventType = "cloudflare.workerRoute.updated"
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		eventType,
		[]any{result},
	)
}

func (u *UpdateWorkerRoute) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateWorkerRoute) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateWorkerRoute) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateWorkerRoute) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (u *UpdateWorkerRoute) Hooks() []core.Hook {
	return []core.Hook{}
}

func (u *UpdateWorkerRoute) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
