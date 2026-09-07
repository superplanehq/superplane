package vercel

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	targetProduction = "production"
	targetPreview    = "preview"

	TriggerDeploymentPayloadType = "vercel.deployment.triggered"
	GetDeploymentPayloadType     = "vercel.deployment"
	ListDeploymentsPayloadType   = "vercel.deployments"
	RollbackPayloadType          = "vercel.rollback"
	ProjectPayloadType           = "vercel.project"
	EnvVarPayloadType            = "vercel.envVar"
	DomainPayloadType            = "vercel.projectDomain"
)

var deploymentTargetOptions = []configuration.FieldOption{
	{Label: "Production", Value: targetProduction},
	{Label: "Preview", Value: targetPreview},
}

var deploymentStateOptions = []string{
	"BLOCKED",
	"BUILDING",
	"CANCELED",
	"ERROR",
	"INITIALIZING",
	"QUEUED",
	"READY",
}

var envTargets = []string{"development", "preview", "production"}

var envTypes = []configuration.FieldOption{
	{Label: "Encrypted", Value: "encrypted"},
	{Label: "Plain", Value: "plain"},
	{Label: "Sensitive", Value: "sensitive"},
}

var allowedDeploymentEventTypes = normalizeEventTypes([]string{
	"deployment.canceled",
	"deployment.created",
	"deployment.error",
	"deployment.promoted",
	"deployment.succeeded",
})

var defaultDeploymentEventTypes = []string{"deployment.succeeded"}

var allowedGitTypes = []string{"github", "gitlab", "bitbucket"}

type OnEventConfiguration struct {
	Project    string   `json:"project" mapstructure:"project"`
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

// WebhookConfiguration is empty on purpose: one account-level webhook serves
// every trigger, so there is nothing to compare or merge.
type WebhookConfiguration struct{}

func decodeOnEventConfiguration(configuration any) (OnEventConfiguration, error) {
	config := OnEventConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Project = strings.TrimSpace(config.Project)
	config.EventTypes = normalizeEventTypes(config.EventTypes)
	return config, nil
}

// selectedEventTypes returns the configured event types, or the default set
// when none were selected.
func selectedEventTypes(config OnEventConfiguration) []string {
	selected := filterAllowedEventTypes(config.EventTypes, allowedDeploymentEventTypes)
	if len(selected) == 0 {
		return defaultDeploymentEventTypes
	}

	return selected
}

func eventPayloadType(eventType string) string {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return "vercel.deployment.event"
	}

	return "vercel." + eventType
}

func deploymentData(deployment *Deployment) map[string]any {
	data := map[string]any{
		"deploymentId": deployment.ID,
		"name":         deployment.Name,
		"url":          deployment.URL,
		"readyState":   deployment.ReadyState,
		"target":       deployment.Target,
		"projectId":    deployment.ProjectID,
	}

	if deployment.CreatedAt > 0 {
		data["createdAt"] = deployment.CreatedAt
	}

	return data
}

func projectData(project *Project) map[string]any {
	data := map[string]any{
		"projectId": project.ID,
		"name":      project.Name,
	}

	if project.Framework != "" {
		data["framework"] = project.Framework
	}
	if project.CreatedAt > 0 {
		data["createdAt"] = project.CreatedAt
	}

	return data
}

func filterAllowedEventTypes(eventTypes []string, allowedEventTypes []string) []string {
	filtered := make([]string, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		if !slices.Contains(allowedEventTypes, eventType) {
			continue
		}

		if slices.Contains(filtered, eventType) {
			continue
		}

		filtered = append(filtered, eventType)
	}

	return filtered
}

func normalizeEventTypes(eventTypes []string) []string {
	normalized := make([]string, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		value := strings.ToLower(strings.TrimSpace(eventType))
		if value == "" || slices.Contains(normalized, value) {
			continue
		}

		normalized = append(normalized, value)
	}

	sort.Strings(normalized)
	return normalized
}

func verifyWebhookSignature(ctx core.WebhookRequestContext) error {
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

	signature := strings.TrimSpace(ctx.Headers.Get("x-vercel-signature"))
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	mac := hmac.New(sha1.New, []byte(strings.TrimSpace(string(secret))))
	mac.Write(ctx.Body)
	expected := []byte(hex.EncodeToString(mac.Sum(nil)))
	if !hmac.Equal(expected, []byte(signature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

func readString(value any) string {
	if value == nil {
		return ""
	}

	s, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(s)
}

func readMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return item
}
