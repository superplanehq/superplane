package octopus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnDeploymentEvent struct{}

type OnDeploymentEventConfiguration struct {
	EventCategories []string `json:"eventCategories" mapstructure:"eventCategories"`
	Project         string   `json:"project" mapstructure:"project"`
	Environment     string   `json:"environment" mapstructure:"environment"`
}

var deploymentEventCategoryOptions = []configuration.FieldOption{
	{Label: "Deployment Queued", Value: EventCategoryDeploymentQueued},
	{Label: "Deployment Started", Value: EventCategoryDeploymentStarted},
	{Label: "Deployment Succeeded", Value: EventCategoryDeploymentSucceeded},
	{Label: "Deployment Failed", Value: EventCategoryDeploymentFailed},
}

var deploymentAllowedEventCategories = []string{
	EventCategoryDeploymentQueued,
	EventCategoryDeploymentStarted,
	EventCategoryDeploymentSucceeded,
	EventCategoryDeploymentFailed,
}

var deploymentDefaultEventCategories = []string{
	EventCategoryDeploymentSucceeded,
	EventCategoryDeploymentFailed,
}

func (t *OnDeploymentEvent) Name() string {
	return "octopus.onDeploymentEvent"
}

func (t *OnDeploymentEvent) Label() string {
	return "On Deployment Event"
}

func (t *OnDeploymentEvent) Description() string {
	return "Listen to Octopus Deploy deployment events"
}

func (t *OnDeploymentEvent) Documentation() string {
	return `The On Deployment Event trigger emits events when a deployment's status changes in Octopus Deploy.

## Use Cases

- **Deploy notifications**: Notify Slack or PagerDuty when deployments succeed or fail
- **Post-deploy automation**: Trigger smoke tests after a successful deployment
- **Incident creation**: Create a ticket automatically when a deployment fails
- **Deployment tracking**: Log deployment events for audit or reporting

## Configuration

- **Event Categories**: Deployment event types to listen for. Defaults to ` + "`DeploymentSucceeded`" + ` and ` + "`DeploymentFailed`" + `.
- **Project** (optional): Filter events to a specific Octopus Deploy project.
- **Environment** (optional): Filter events to a specific deployment environment.

## Event Categories

|      Category       |             Description             |    
|---------------------|-------------------------------------|
| DeploymentQueued    | A deployment has been queued        |
| DeploymentStarted   | A deployment has started executing  |
| DeploymentSucceeded | A deployment completed successfully |
| DeploymentFailed    | A deployment has failed             |

## Webhook Verification

Octopus Deploy subscriptions are configured with a custom header secret. SuperPlane verifies incoming webhooks by comparing the ` + "`X-SuperPlane-Webhook-Secret`" + ` header value against the stored secret.

## Event Data

Each emitted event includes the following fields:

` +
		"|      Field      |                        Description                         |\n" +
		"|-----------------|------------------------------------------------------------|\n" +
		"| `eventType`     | The deployment event category (e.g. `DeploymentSucceeded`) |\n" +
		"| `category`      | Same as `eventType`                                        |\n" +
		"| `timestamp`     | When the subscription payload was sent                     |\n" +
		"| `occurredAt`    | When the event occurred in Octopus                         |\n" +
		"| `message`       | Human-readable event description                           |\n" +
		"| `projectId`     | Octopus project ID (e.g. `Projects-123`)                   |\n" +
		"| `environmentId` | Octopus environment ID (e.g. `Environments-1`)             |\n" +
		"| `releaseId`     | Octopus release ID (e.g. `Releases-789`)                   |\n" +
		"| `deploymentId`  | Octopus deployment ID (e.g. `Deployments-1011`)            |\n" +
		"| `serverUri`     | Your Octopus Deploy server URL                             |"
}

func (t *OnDeploymentEvent) Icon() string {
	return "rocket"
}

func (t *OnDeploymentEvent) Color() string {
	return "blue"
}

func (t *OnDeploymentEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "eventCategories",
			Label:       "Event Categories",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     deploymentDefaultEventCategories,
			Description: "Deployment event types to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: deploymentEventCategoryOptions,
				},
			},
		},
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
			Description: "Optional: Filter events to a specific Octopus Deploy project",
		},
		{
			Name:     "environment",
			Label:    "Environment",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "environment",
				},
			},
			Description: "Optional: Filter events to a specific deployment environment",
		},
	}
}

func (t *OnDeploymentEvent) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnDeploymentEventConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Resolve human-readable names for display in the UI
	nodeMetadata := resolveNodeMetadata(ctx.HTTP, ctx.Integration, config.Project, "", config.Environment)
	if err := ctx.Metadata.Set(nodeMetadata); err != nil {
		return fmt.Errorf("failed to store node metadata: %w", err)
	}

	selectedCategories := filterAllowedEventCategories(config.EventCategories, deploymentAllowedEventCategories)
	if len(selectedCategories) == 0 {
		selectedCategories = deploymentDefaultEventCategories
	}

	webhookConfig := WebhookConfiguration{
		EventCategories: selectedCategories,
	}

	if config.Project != "" {
		webhookConfig.Projects = []string{config.Project}
	}

	if config.Environment != "" {
		webhookConfig.Environments = []string{config.Environment}
	}

	return ctx.Integration.RequestWebhook(webhookConfig)
}

func (t *OnDeploymentEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnDeploymentEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnDeploymentEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := verifyWebhookHeader(ctx); err != nil {
		return http.StatusForbidden, err
	}

	if !webhookRequestIsJSON(ctx) {
		return okResponse()
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return errorResponse(http.StatusBadRequest, "error parsing request body: %w", err)
	}

	config, err := decodeOnDeploymentEventConfiguration(ctx.Configuration)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "failed to decode configuration: %w", err)
	}

	// Octopus Deploy webhook payload structure:
	// {
	//   "Timestamp": "...",
	//   "EventType": "SubscriptionPayload",
	//   "Payload": {
	//     "Event": { "Category": "DeploymentSucceeded", ... },
	//     ...
	//   }
	// }
	// The actual event category is in Payload.Event.Category, not the top-level EventType.
	eventPayload := readMap(payload["Payload"])
	event := readMap(eventPayload["Event"])
	eventType := readString(event["Category"])
	if eventType == "" {
		return okResponse()
	}

	// Filter by event category
	selectedCategories := filterAllowedEventCategories(config.EventCategories, deploymentAllowedEventCategories)
	if len(selectedCategories) == 0 {
		selectedCategories = deploymentDefaultEventCategories
	}

	if !isDeploymentEventCategory(eventType) {
		return okResponse()
	}

	if len(selectedCategories) > 0 && !containsEventCategory(selectedCategories, eventType) {
		return okResponse()
	}

	// Filter by project and/or environment if configured
	relatedDocIDs := readRelatedDocumentIDs(event)
	if config.Project != "" && !containsRelatedDocument(relatedDocIDs, "Projects", config.Project) {
		return okResponse()
	}

	if config.Environment != "" && !containsRelatedDocument(relatedDocIDs, "Environments", config.Environment) {
		return okResponse()
	}

	// Build the emitted data
	emittedData := buildTriggerEventData(payload, eventType, eventPayload, event)

	// Enrich with human-readable names (best-effort; errors are ignored)
	enrichTriggerEventDataWithNames(ctx, emittedData)

	if err := ctx.Events.Emit(payloadType(eventType), emittedData); err != nil {
		return errorResponse(http.StatusInternalServerError, "error emitting event: %w", err)
	}

	return okResponse()
}

func (t *OnDeploymentEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func decodeOnDeploymentEventConfiguration(configuration any) (OnDeploymentEventConfiguration, error) {
	config := OnDeploymentEventConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return config, err
	}

	config.EventCategories = normalizeEventCategories(config.EventCategories)
	config.Project = strings.TrimSpace(config.Project)
	config.Environment = strings.TrimSpace(config.Environment)
	return config, nil
}

func isDeploymentEventCategory(eventType string) bool {
	switch eventType {
	case EventCategoryDeploymentQueued,
		EventCategoryDeploymentStarted,
		EventCategoryDeploymentSucceeded,
		EventCategoryDeploymentFailed:
		return true
	default:
		return false
	}
}

func containsEventCategory(categories []string, eventType string) bool {
	for _, cat := range categories {
		if strings.EqualFold(cat, eventType) {
			return true
		}
	}
	return false
}

func readRelatedDocumentIDs(event map[string]any) map[string][]string {
	result := map[string][]string{}

	relatedDocs, ok := event["RelatedDocumentIds"]
	if !ok {
		return result
	}

	docSlice, ok := relatedDocs.([]any)
	if !ok {
		return result
	}

	for _, doc := range docSlice {
		docStr, ok := doc.(string)
		if !ok {
			continue
		}

		// Octopus uses IDs like "Projects-123", "Environments-456"
		parts := strings.SplitN(docStr, "-", 2)
		if len(parts) != 2 {
			continue
		}

		category := parts[0]
		result[category] = append(result[category], docStr)
	}

	return result
}

func containsRelatedDocument(relatedDocIDs map[string][]string, category, targetID string) bool {
	docs, ok := relatedDocIDs[category]
	if !ok {
		return false
	}

	for _, docID := range docs {
		if docID == targetID {
			return true
		}
	}

	return false
}

// enrichTriggerEventDataWithNames resolves Octopus resource IDs to human-readable
// names and adds them to the event data as projectName, environmentName, releaseName.
// Enrichment is best-effort: errors are ignored so webhook processing is not blocked.
func enrichTriggerEventDataWithNames(ctx core.WebhookRequestContext, data map[string]any) {
	if ctx.HTTP == nil {
		return
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return
	}

	spaceID, err := spaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return
	}

	if projectID, ok := data["projectId"].(string); ok && projectID != "" {
		if project, err := client.GetProject(spaceID, projectID); err == nil && project.Name != "" {
			data["projectName"] = project.Name
		}
	}

	if envID, ok := data["environmentId"].(string); ok && envID != "" {
		if env, err := client.GetEnvironment(spaceID, envID); err == nil && env.Name != "" {
			data["environmentName"] = env.Name
		}
	}

	if releaseID, ok := data["releaseId"].(string); ok && releaseID != "" {
		if release, err := client.GetRelease(spaceID, releaseID); err == nil && release.Version != "" {
			data["releaseName"] = release.Version
		}
	}
}

func buildTriggerEventData(
	payload map[string]any,
	eventType string,
	eventPayload map[string]any,
	event map[string]any,
) map[string]any {
	data := map[string]any{
		"eventType": eventType,
		"timestamp": readString(payload["Timestamp"]),
	}

	if category := readString(event["Category"]); category != "" {
		data["category"] = category
	}

	if message := readString(event["Message"]); message != "" {
		data["message"] = message
	}

	if occurredAt := readString(event["Occurred"]); occurredAt != "" {
		data["occurredAt"] = occurredAt
	}

	// Extract related document IDs for context
	relatedDocs := readRelatedDocumentIDs(event)
	if projectIDs, ok := relatedDocs["Projects"]; ok && len(projectIDs) > 0 {
		data["projectId"] = projectIDs[0]
	}
	if envIDs, ok := relatedDocs["Environments"]; ok && len(envIDs) > 0 {
		data["environmentId"] = envIDs[0]
	}
	if releaseIDs, ok := relatedDocs["Releases"]; ok && len(releaseIDs) > 0 {
		data["releaseId"] = releaseIDs[0]
	}
	if deploymentIDs, ok := relatedDocs["Deployments"]; ok && len(deploymentIDs) > 0 {
		data["deploymentId"] = deploymentIDs[0]
	}

	if serverURI := readString(eventPayload["ServerUri"]); serverURI != "" {
		data["serverUri"] = serverURI
	}

	return data
}
