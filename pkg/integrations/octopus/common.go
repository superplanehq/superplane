package octopus

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// Deployment event categories in Octopus Deploy
	EventCategoryDeploymentQueued    = "DeploymentQueued"
	EventCategoryDeploymentStarted   = "DeploymentStarted"
	EventCategoryDeploymentSucceeded = "DeploymentSucceeded"
	EventCategoryDeploymentFailed    = "DeploymentFailed"

	// Task states in Octopus Deploy
	TaskStateQueued     = "Queued"
	TaskStateExecuting  = "Executing"
	TaskStateSuccess    = "Success"
	TaskStateFailed     = "Failed"
	TaskStateCanceled   = "Canceled"
	TaskStateTimedOut   = "TimedOut"
	TaskStateCancelling = "Cancelling"

	// Custom header for webhook verification
	webhookHeaderKey = "X-SuperPlane-Webhook-Secret"
)

// NodeMetadata contains resolved human-readable names for display in the UI.
// Stored on component/trigger nodes during Setup so the frontend can show
// names (e.g. "My Project") instead of Octopus IDs (e.g. "Projects-1").
type NodeMetadata struct {
	ProjectName     string `json:"projectName,omitempty"`
	ReleaseName     string `json:"releaseName,omitempty"`
	EnvironmentName string `json:"environmentName,omitempty"`
}

// resolveNodeMetadata fetches human-readable names for Octopus resource IDs.
// Skips resolution for empty values or expression placeholders (containing "{{").
// Errors during resolution are non-fatal: fields are left empty and the UI
// falls back to showing the raw ID from configuration.
func resolveNodeMetadata(http core.HTTPContext, integration core.IntegrationContext, projectID, releaseID, environmentID string) NodeMetadata {
	metadata := NodeMetadata{}

	if http == nil {
		return metadata
	}

	client, err := NewClient(http, integration)
	if err != nil {
		return metadata
	}

	spaceID, err := spaceIDForIntegration(client, integration)
	if err != nil {
		return metadata
	}

	if isResolvedValue(projectID) {
		project, err := client.GetProject(spaceID, projectID)
		if err == nil && project.Name != "" {
			metadata.ProjectName = project.Name
		}
	}

	if isResolvedValue(releaseID) {
		release, err := client.GetRelease(spaceID, releaseID)
		if err == nil && release.Version != "" {
			metadata.ReleaseName = release.Version
		}
	}

	if isResolvedValue(environmentID) {
		env, err := client.GetEnvironment(spaceID, environmentID)
		if err == nil && env.Name != "" {
			metadata.EnvironmentName = env.Name
		}
	}

	return metadata
}

func verifyWebhookHeader(ctx core.WebhookRequestContext) error {
	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("error reading webhook secret")
	}

	if len(secret) == 0 {
		return fmt.Errorf("missing webhook secret")
	}

	headerValue := strings.TrimSpace(ctx.Headers.Get(webhookHeaderKey))
	if headerValue == "" {
		return fmt.Errorf("missing %s header", webhookHeaderKey)
	}

	if subtle.ConstantTimeCompare([]byte(headerValue), secret) != 1 {
		return fmt.Errorf("invalid webhook secret")
	}

	return nil
}

func normalizeEventCategories(categories []string) []string {
	normalized := make([]string, 0, len(categories))
	for _, category := range categories {
		trimmed := strings.TrimSpace(category)
		if trimmed == "" {
			continue
		}

		if slices.Contains(normalized, trimmed) {
			continue
		}

		normalized = append(normalized, trimmed)
	}

	sort.Strings(normalized)
	return normalized
}

func filterAllowedEventCategories(categories []string, allowed []string) []string {
	filtered := make([]string, 0, len(categories))
	for _, category := range categories {
		if !slices.Contains(allowed, category) {
			continue
		}

		if slices.Contains(filtered, category) {
			continue
		}

		filtered = append(filtered, category)
	}

	return filtered
}

func readString(value any) string {
	if value == nil {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

func readMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	m, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return m
}

func isTaskCompleted(state string) bool {
	switch state {
	case TaskStateSuccess, TaskStateFailed, TaskStateCanceled, TaskStateTimedOut:
		return true
	default:
		return false
	}
}

func isTaskSuccessful(state string) bool {
	return state == TaskStateSuccess
}

func payloadType(eventCategory string) string {
	switch eventCategory {
	case EventCategoryDeploymentQueued:
		return "octopus.deployment.queued"
	case EventCategoryDeploymentStarted:
		return "octopus.deployment.started"
	case EventCategoryDeploymentSucceeded:
		return "octopus.deployment.succeeded"
	case EventCategoryDeploymentFailed:
		return "octopus.deployment.failed"
	default:
		return "octopus.deployment." + strings.ToLower(eventCategory)
	}
}

func webhookRequestIsJSON(ctx core.WebhookRequestContext) bool {
	contentType := ctx.Headers.Get("Content-Type")
	return strings.Contains(strings.ToLower(contentType), "application/json")
}

func errorResponse(statusCode int, format string, a ...any) (int, error) {
	return statusCode, fmt.Errorf(format, a...)
}

func okResponse() (int, error) {
	return http.StatusOK, nil
}

func isResolvedValue(value string) bool {
	return value != "" && !strings.Contains(value, "{{")
}
