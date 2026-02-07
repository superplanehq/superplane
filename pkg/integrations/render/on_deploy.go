package render

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnDeploy struct{}

var deployEventTypeOptions = []configuration.FieldOption{
	{Label: "Deploy Ended", Value: "deploy_ended"},
	{Label: "Deploy Started", Value: "deploy_started"},
	{Label: "Image Pull Failed", Value: "image_pull_failed"},
	{Label: "Pipeline Minutes Exhausted", Value: "pipeline_minutes_exhausted"},
	{Label: "Pre-Deploy Ended", Value: "pre_deploy_ended"},
	{Label: "Pre-Deploy Started", Value: "pre_deploy_started"},
}

var deployAllowedEventTypes = normalizeWebhookEventTypes([]string{
	"deploy_ended",
	"deploy_started",
	"image_pull_failed",
	"pipeline_minutes_exhausted",
	"pre_deploy_ended",
	"pre_deploy_started",
})

var deployDefaultEventTypes = []string{"deploy_ended"}

func (t *OnDeploy) Name() string {
	return "render.onDeploy"
}

func (t *OnDeploy) Label() string {
	return "On Deploy"
}

func (t *OnDeploy) Description() string {
	return "Listen to Render deploy events for a service"
}

func (t *OnDeploy) Documentation() string {
	return `The On Deploy trigger emits deploy-related Render events for one selected service.

## Use Cases

- **Deploy notifications**: Notify Slack or PagerDuty when deploys succeed/fail
- **Post-deploy automation**: Trigger smoke tests after successful deploy completion events
- **Release orchestration**: Trigger downstream workflows when deploy stages change

## Configuration

- **Service**: Required Render service.
- **Event Types**: Deploy event states to listen for. Defaults to ` + "`deploy_ended`" + `.

## Webhook Verification

Render webhooks are validated using the secret generated when SuperPlane creates the webhook via the Render API. Verification checks:
- ` + "`webhook-id`" + `
- ` + "`webhook-timestamp`" + `
- ` + "`webhook-signature`" + ` (` + "`v1,<base64-signature>`" + `)

## Event Data

The default output emits payload data fields like ` + "`id`" + `, ` + "`serviceId`" + `, ` + "`serviceName`" + `, and ` + "`status`" + ` (when present).`
}

func (t *OnDeploy) Icon() string {
	return "server"
}

func (t *OnDeploy) Color() string {
	return "gray"
}

func (t *OnDeploy) Configuration() []configuration.Field {
	return onResourceEventConfigurationFields(deployEventTypeOptions, deployDefaultEventTypes)
}

func (t *OnDeploy) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnResourceEventConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	if err := ensureServiceInMetadata(ctx, config); err != nil {
		return err
	}

	requestedEventTypes := filterAllowedEventTypes(config.EventTypes, deployAllowedEventTypes)
	if len(requestedEventTypes) == 0 {
		requestedEventTypes = deployDefaultEventTypes
	}

	return ctx.Integration.RequestWebhook(
		webhookConfigurationForResource(ctx.Integration, renderWebhookResourceTypeDeploy, requestedEventTypes),
	)
}

func (t *OnDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnDeploy) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config, err := decodeOnResourceEventConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return handleOnResourceEventWebhook(ctx, config, deployAllowedEventTypes, deployDefaultEventTypes)
}

func (t *OnDeploy) Cleanup(ctx core.TriggerContext) error {
	return nil
}
