package fluxcd

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnReconciliationCompleted struct{}

type OnReconciliationCompletedConfig struct {
	SharedSecret string   `json:"sharedSecret" mapstructure:"sharedSecret"`
	Kinds        []string `json:"kinds" mapstructure:"kinds"`
}

var reconciliationKindOptions = []configuration.FieldOption{
	{Label: "Kustomization", Value: "Kustomization"},
	{Label: "HelmRelease", Value: "HelmRelease"},
	{Label: "GitRepository", Value: "GitRepository"},
	{Label: "HelmRepository", Value: "HelmRepository"},
	{Label: "OCIRepository", Value: "OCIRepository"},
	{Label: "Bucket", Value: "Bucket"},
}

func (t *OnReconciliationCompleted) Name() string {
	return "fluxcd.onReconciliationCompleted"
}

func (t *OnReconciliationCompleted) Label() string {
	return "On Reconciliation Completed"
}

func (t *OnReconciliationCompleted) Description() string {
	return "Trigger when a Flux reconciliation completes successfully"
}

func (t *OnReconciliationCompleted) Documentation() string {
	return `The On Reconciliation Completed trigger fires when a Flux CD reconciliation finishes successfully.

## Setup

1. Save the canvas to generate the webhook URL.
2. In your cluster, create a FluxCD Notification Provider of type ` + "`generic`" + `:
   ` + "```yaml" + `
   apiVersion: notification.toolkit.fluxcd.io/v1beta3
   kind: Provider
   metadata:
     name: superplane
     namespace: flux-system
   spec:
     type: generic
     address: <webhook-url-from-superplane>
   ` + "```" + `
3. Create a FluxCD Alert to send events to the provider:
   ` + "```yaml" + `
   apiVersion: notification.toolkit.fluxcd.io/v1beta3
   kind: Alert
   metadata:
     name: superplane-alert
     namespace: flux-system
   spec:
     providerRef:
       name: superplane
     eventSources:
       - kind: Kustomization
         name: "*"
       - kind: HelmRelease
         name: "*"
     eventSeverity: info
   ` + "```" + `

## Event Data

The trigger emits the FluxCD notification payload, including:
- ` + "`involvedObject`" + `: The Flux resource (kind, name, namespace)
- ` + "`severity`" + `: Event severity (info, error)
- ` + "`message`" + `: Human-readable reconciliation message
- ` + "`reason`" + `: Reconciliation reason (e.g. ReconciliationSucceeded)
- ` + "`metadata`" + `: Additional data such as revision
- ` + "`timestamp`" + `: When the event occurred
`
}

func (t *OnReconciliationCompleted) Icon() string {
	return "git-branch"
}

func (t *OnReconciliationCompleted) Color() string {
	return "gray"
}

func (t *OnReconciliationCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sharedSecret",
			Label:       "Shared Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    false,
			Description: "Optional shared secret sent as Authorization: Bearer <secret> header by the FluxCD notification provider",
			Placeholder: "your-secret",
		},
		{
			Name:     "kinds",
			Label:    "Resource Kinds",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			Default:  []string{},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: reconciliationKindOptions,
				},
			},
			Description: "Filter events by resource kind. Leave empty to receive events from all kinds.",
		},
	}
}

func (t *OnReconciliationCompleted) Setup(ctx core.TriggerContext) error {
	if ctx.Integration == nil {
		return fmt.Errorf("missing integration context")
	}

	config := OnReconciliationCompletedConfig{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	bindingKey := getWebhookBindingKey(ctx)
	if bindingKey == "" {
		bindingKey = uuid.NewString()
	}

	requestConfig := WebhookConfiguration{
		SharedSecret:      strings.TrimSpace(config.SharedSecret),
		WebhookBindingKey: bindingKey,
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	if err := ctx.Integration.RequestWebhook(requestConfig); err != nil {
		return err
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return err
	}

	if ctx.Metadata != nil {
		if err := setWebhookMetadata(ctx, webhookURL, bindingKey); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("fluxcd onReconciliationCompleted: failed to store webhook url metadata: %v", err)
		}
	}

	return nil
}

func (t *OnReconciliationCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnReconciliationCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnReconciliationCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	sharedSecret, err := resolveWebhookSharedSecret(ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if sharedSecret != "" {
		authHeader := strings.TrimSpace(ctx.Headers.Get("Authorization"))
		if authHeader == "" {
			return http.StatusUnauthorized, fmt.Errorf("missing Authorization header")
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return http.StatusUnauthorized, fmt.Errorf("invalid Authorization header")
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if subtle.ConstantTimeCompare([]byte(token), []byte(sharedSecret)) != 1 {
			return http.StatusUnauthorized, fmt.Errorf("invalid Authorization token")
		}
	}

	if len(ctx.Body) == 0 {
		return http.StatusBadRequest, fmt.Errorf("empty body")
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	if !isSuccessfulReconciliation(payload) {
		return http.StatusOK, nil
	}

	config := OnReconciliationCompletedConfig{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error decoding configuration: %v", err)
	}

	if len(config.Kinds) > 0 {
		kind := extractInvolvedObjectField(payload, "kind")
		if kind == "" || !containsKind(config.Kinds, kind) {
			return http.StatusOK, nil
		}
	}

	if err := ctx.Events.Emit("fluxcd.reconciliation", payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnReconciliationCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func isSuccessfulReconciliation(payload map[string]any) bool {
	severity := extractString(payload["severity"])
	if !strings.EqualFold(severity, "info") {
		return false
	}

	reason := extractString(payload["reason"])
	return strings.Contains(strings.ToLower(reason), "succeeded")
}

func extractInvolvedObjectField(payload map[string]any, field string) string {
	obj, ok := payload["involvedObject"]
	if !ok {
		return ""
	}

	objMap, ok := obj.(map[string]any)
	if !ok {
		return ""
	}

	return extractString(objMap[field])
}

func containsKind(kinds []string, kind string) bool {
	for _, k := range kinds {
		if strings.EqualFold(k, kind) {
			return true
		}
	}
	return false
}

func extractString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func resolveWebhookSharedSecret(ctx core.WebhookRequestContext) (string, error) {
	if ctx.Webhook != nil {
		secret, err := ctx.Webhook.GetSecret()
		if err != nil {
			return "", fmt.Errorf("error getting webhook secret: %w", err)
		}

		normalizedSecret := strings.TrimSpace(string(secret))
		if normalizedSecret != "" {
			return normalizedSecret, nil
		}
	}

	config := OnReconciliationCompletedConfig{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return "", fmt.Errorf("error decoding configuration: %v", err)
	}

	return strings.TrimSpace(config.SharedSecret), nil
}

func getWebhookBindingKey(ctx core.TriggerContext) string {
	if ctx.Metadata == nil {
		return ""
	}

	existing := ctx.Metadata.Get()
	existingMap, ok := existing.(map[string]any)
	if !ok {
		return ""
	}

	return strings.TrimSpace(extractString(existingMap["webhookBindingKey"]))
}

func setWebhookMetadata(ctx core.TriggerContext, webhookURL, bindingKey string) error {
	metadata := map[string]any{}
	if existing := ctx.Metadata.Get(); existing != nil {
		if existingMap, ok := existing.(map[string]any); ok {
			for key, value := range existingMap {
				metadata[key] = value
			}
		}
	}

	metadata["webhookUrl"] = webhookURL
	metadata["webhookBindingKey"] = bindingKey
	return ctx.Metadata.Set(metadata)
}
