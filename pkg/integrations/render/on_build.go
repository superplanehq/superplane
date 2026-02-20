package render

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnBuild struct{}

var buildEventTypeOptions = []configuration.FieldOption{
	{Label: "Build Ended", Value: "build_ended"},
	{Label: "Build Started", Value: "build_started"},
}

var buildAllowedEventTypes = normalizeWebhookEventTypes([]string{
	"build_ended",
	"build_started",
})

var buildDefaultEventTypes = []string{"build_ended"}

func (t *OnBuild) Name() string {
	return "render.onBuild"
}

func (t *OnBuild) Label() string {
	return "On Build"
}

func (t *OnBuild) Description() string {
	return "Listen to Render build events for a service"
}

func (t *OnBuild) Documentation() string {
	return `The On Build trigger emits build-related Render events for one selected service.

## Use Cases

- **Build failure alerts**: Notify your team when builds fail
- **Build success hooks**: Trigger follow-up automation after successful builds

## Configuration

- **Service**: Required Render service.
- **Event Types**: Build event states to listen for. Defaults to ` + "`build_ended`" + `.

## Webhook Verification

Render webhooks are validated using the secret generated when SuperPlane creates the webhook via the Render API. Verification checks:
- ` + "`webhook-id`" + `
- ` + "`webhook-timestamp`" + `
- ` + "`webhook-signature`" + ` (` + "`v1,<base64-signature>`" + `)

## Event Data

The default output emits payload data fields like ` + "`buildId`" + `, ` + "`eventId`" + `, ` + "`serviceId`" + `, ` + "`serviceName`" + `, and ` + "`status`" + ` (when present).`
}

func (t *OnBuild) Icon() string {
	return "server"
}

func (t *OnBuild) Color() string {
	return "gray"
}

func (t *OnBuild) Configuration() []configuration.Field {
	return onResourceEventConfigurationFields(buildEventTypeOptions, buildDefaultEventTypes)
}

func (t *OnBuild) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnResourceEventConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := ensureServiceInMetadata(ctx, config); err != nil {
		return err
	}

	requestedEventTypes := filterAllowedEventTypes(config.EventTypes, buildAllowedEventTypes)
	if len(requestedEventTypes) == 0 {
		requestedEventTypes = buildDefaultEventTypes
	}

	return ctx.Integration.RequestWebhook(
		webhookConfigurationForResource(ctx.Integration, webhookResourceTypeBuild, requestedEventTypes),
	)
}

func (t *OnBuild) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnBuild) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnBuild) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config, err := decodeOnResourceEventConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return handleOnResourceEventWebhook(
		ctx,
		config,
		buildAllowedEventTypes,
		buildDefaultEventTypes,
		"buildId",
	)
}

func (t *OnBuild) Cleanup(ctx core.TriggerContext) error {
	return nil
}
