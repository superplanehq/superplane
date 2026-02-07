package render

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnResourceEventConfiguration struct {
	ServiceID  string   `json:"serviceId" mapstructure:"serviceId"`
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

type WebhookConfiguration struct {
	Strategy     string   `json:"strategy" mapstructure:"strategy"`
	ResourceType string   `json:"resourceType,omitempty" mapstructure:"resourceType"`
	EventTypes   []string `json:"eventTypes,omitempty" mapstructure:"eventTypes"`
}

const (
	renderWorkspacePlanProfessional    = "professional"
	renderWorkspacePlanOrganization    = "organization"
	renderWorkspacePlanEnterpriseAlias = "enterprise"

	renderWebhookStrategyIntegration  = "integration"
	renderWebhookStrategyResourceType = "resource_type"

	renderWebhookResourceTypeDeploy = "deploy"
	renderWebhookResourceTypeBuild  = "build"
)

func renderWebhookConfigurationForResource(
	integration core.IntegrationContext,
	resourceType string,
	eventTypes []string,
) WebhookConfiguration {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return WebhookConfiguration{
			Strategy:   renderWebhookStrategyIntegration,
			EventTypes: normalizeWebhookEventTypes(eventTypes),
		}
	}

	workspacePlan := strings.ToLower(strings.TrimSpace(metadata.WorkspacePlan))
	if workspacePlan != renderWorkspacePlanOrganization && workspacePlan != renderWorkspacePlanEnterpriseAlias {
		return WebhookConfiguration{
			Strategy:   renderWebhookStrategyIntegration,
			EventTypes: normalizeWebhookEventTypes(eventTypes),
		}
	}

	normalizedResourceType := strings.ToLower(strings.TrimSpace(resourceType))
	switch normalizedResourceType {
	case renderWebhookResourceTypeDeploy, renderWebhookResourceTypeBuild:
		return WebhookConfiguration{
			Strategy:     renderWebhookStrategyResourceType,
			ResourceType: normalizedResourceType,
			EventTypes:   normalizeWebhookEventTypes(eventTypes),
		}
	default:
		return WebhookConfiguration{
			Strategy:   renderWebhookStrategyIntegration,
			EventTypes: normalizeWebhookEventTypes(eventTypes),
		}
	}
}

func decodeWebhookConfiguration(configuration any) (WebhookConfiguration, error) {
	webhookConfiguration := WebhookConfiguration{
		Strategy: renderWebhookStrategyIntegration,
	}

	if configuration == nil {
		return webhookConfiguration, nil
	}

	if err := mapstructure.Decode(configuration, &webhookConfiguration); err != nil {
		return WebhookConfiguration{}, err
	}

	return normalizeWebhookConfiguration(webhookConfiguration), nil
}

func normalizeWebhookConfiguration(configuration WebhookConfiguration) WebhookConfiguration {
	normalizedConfiguration := WebhookConfiguration{
		Strategy:     strings.ToLower(strings.TrimSpace(configuration.Strategy)),
		ResourceType: strings.ToLower(strings.TrimSpace(configuration.ResourceType)),
		EventTypes:   normalizeWebhookEventTypes(configuration.EventTypes),
	}

	if normalizedConfiguration.Strategy == "" {
		if normalizedConfiguration.ResourceType == "" {
			normalizedConfiguration.Strategy = renderWebhookStrategyIntegration
		} else {
			normalizedConfiguration.Strategy = renderWebhookStrategyResourceType
		}
	}

	if normalizedConfiguration.Strategy != renderWebhookStrategyResourceType {
		return WebhookConfiguration{
			Strategy:   renderWebhookStrategyIntegration,
			EventTypes: normalizedConfiguration.EventTypes,
		}
	}

	switch normalizedConfiguration.ResourceType {
	case renderWebhookResourceTypeDeploy, renderWebhookResourceTypeBuild:
		return normalizedConfiguration
	default:
		return WebhookConfiguration{
			Strategy:   renderWebhookStrategyIntegration,
			EventTypes: normalizedConfiguration.EventTypes,
		}
	}
}

func renderWebhookName(configuration WebhookConfiguration) string {
	configuration = normalizeWebhookConfiguration(configuration)
	if configuration.Strategy == renderWebhookStrategyResourceType &&
		configuration.ResourceType == renderWebhookResourceTypeDeploy {
		return "SuperPlane Deploy"
	}

	if configuration.Strategy == renderWebhookStrategyResourceType &&
		configuration.ResourceType == renderWebhookResourceTypeBuild {
		return "SuperPlane Build"
	}

	return "SuperPlane"
}

func renderWebhookEventFilter(configuration WebhookConfiguration) []string {
	configuration = normalizeWebhookConfiguration(configuration)
	allowedEventTypes := renderAllowedEventTypesForWebhook(configuration)
	requestedEventTypes := filterAllowedEventTypes(configuration.EventTypes, allowedEventTypes)
	if len(requestedEventTypes) > 0 {
		return requestedEventTypes
	}

	defaultEventTypes := renderDefaultEventTypesForWebhook(configuration)
	if len(defaultEventTypes) > 0 {
		return defaultEventTypes
	}

	return allowedEventTypes
}

func renderAllowedEventTypesForWebhook(configuration WebhookConfiguration) []string {
	if configuration.Strategy == renderWebhookStrategyResourceType {
		switch configuration.ResourceType {
		case renderWebhookResourceTypeDeploy:
			return deployAllowedEventTypes
		case renderWebhookResourceTypeBuild:
			return buildAllowedEventTypes
		}
	}

	eventTypes := make([]string, 0, len(deployAllowedEventTypes)+len(buildAllowedEventTypes))
	eventTypes = append(eventTypes, deployAllowedEventTypes...)
	eventTypes = append(eventTypes, buildAllowedEventTypes...)

	return normalizeWebhookEventTypes(eventTypes)
}

func renderDefaultEventTypesForWebhook(configuration WebhookConfiguration) []string {
	if configuration.Strategy == renderWebhookStrategyResourceType {
		switch configuration.ResourceType {
		case renderWebhookResourceTypeDeploy:
			return normalizeWebhookEventTypes(deployDefaultEventTypes)
		case renderWebhookResourceTypeBuild:
			return normalizeWebhookEventTypes(buildDefaultEventTypes)
		}
	}

	eventTypes := make([]string, 0, len(deployDefaultEventTypes)+len(buildDefaultEventTypes))
	eventTypes = append(eventTypes, deployDefaultEventTypes...)
	eventTypes = append(eventTypes, buildDefaultEventTypes...)
	return normalizeWebhookEventTypes(eventTypes)
}

func renderWebhookConfigurationsEqual(a, b WebhookConfiguration) bool {
	normalizedA := normalizeWebhookConfiguration(a)
	normalizedB := normalizeWebhookConfiguration(b)

	return normalizedA.Strategy == normalizedB.Strategy &&
		normalizedA.ResourceType == normalizedB.ResourceType &&
		slices.Equal(normalizedA.EventTypes, normalizedB.EventTypes)
}

func onResourceEventConfigurationFields(
	eventTypeOptions []configuration.FieldOption,
	defaultEventTypes []string,
) []configuration.Field {
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
			Description: "Render service to listen to",
		},
		{
			Name:        "eventTypes",
			Label:       "Event Types",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     defaultEventTypes,
			Description: "Render event types to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: eventTypeOptions,
				},
			},
		},
	}
}

func decodeOnResourceEventConfiguration(configuration any) (OnResourceEventConfiguration, error) {
	config := OnResourceEventConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return config, err
	}

	config.ServiceID = strings.TrimSpace(config.ServiceID)
	config.EventTypes = normalizeWebhookEventTypes(config.EventTypes)
	return config, nil
}

func handleOnResourceEventWebhook(
	ctx core.WebhookRequestContext,
	config OnResourceEventConfiguration,
	allowedEventTypes []string,
	defaultEventTypes []string,
) (int, error) {
	if err := verifyRenderWebhookSignature(ctx); err != nil {
		return http.StatusForbidden, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %w", err)
	}

	eventType := readString(payload["type"])
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing event type")
	}

	if !slices.Contains(allowedEventTypes, eventType) {
		return http.StatusOK, nil
	}

	data := readMap(payload["data"])
	serviceID := readString(data["serviceId"])
	if config.ServiceID == "" || serviceID == "" || config.ServiceID != serviceID {
		return http.StatusOK, nil
	}

	selectedEventTypes := filterAllowedEventTypes(config.EventTypes, allowedEventTypes)
	if len(selectedEventTypes) == 0 {
		selectedEventTypes = defaultEventTypes
	}

	if !slices.Contains(selectedEventTypes, eventType) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit(renderPayloadType(eventType), data); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil
}

func filterAllowedEventTypes(eventTypes []string, allowedEventTypes []string) []string {
	filteredEventTypes := make([]string, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		if !slices.Contains(allowedEventTypes, eventType) {
			continue
		}

		if slices.Contains(filteredEventTypes, eventType) {
			continue
		}

		filteredEventTypes = append(filteredEventTypes, eventType)
	}

	return filteredEventTypes
}

func normalizeWebhookEventTypes(eventTypes []string) []string {
	normalizedEventTypes := make([]string, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		normalizedEventType := strings.ToLower(strings.TrimSpace(eventType))
		if normalizedEventType == "" {
			continue
		}

		if slices.Contains(normalizedEventTypes, normalizedEventType) {
			continue
		}

		normalizedEventTypes = append(normalizedEventTypes, normalizedEventType)
	}

	sort.Strings(normalizedEventTypes)
	return normalizedEventTypes
}

func verifyRenderWebhookSignature(ctx core.WebhookRequestContext) error {
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

	webhookID := strings.TrimSpace(ctx.Headers.Get("webhook-id"))
	webhookTimestamp := strings.TrimSpace(ctx.Headers.Get("webhook-timestamp"))
	signatureHeader := strings.TrimSpace(ctx.Headers.Get("webhook-signature"))

	if webhookID == "" || webhookTimestamp == "" || signatureHeader == "" {
		return fmt.Errorf("missing signature headers")
	}

	signatures, err := parseRenderWebhookSignatures(signatureHeader)
	if err != nil {
		return err
	}

	signingKeys := renderSigningKeys(secret)
	payloadPrefix := webhookID + "." + webhookTimestamp + "."
	secretText := []byte(strings.TrimSpace(string(secret)))

	for _, key := range signingKeys {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(payloadPrefix))
		h.Write(ctx.Body)
		if matchesAnyRenderSignature(signatures, h.Sum(nil)) {
			return nil
		}

		// Compatibility fallback for providers that document signature input as:
		// webhook-id.webhook-timestamp.body.webhook-secret
		h = hmac.New(sha256.New, key)
		h.Write([]byte(payloadPrefix))
		h.Write(ctx.Body)
		h.Write([]byte("."))
		h.Write(secretText)
		if matchesAnyRenderSignature(signatures, h.Sum(nil)) {
			return nil
		}
	}

	return fmt.Errorf("invalid signature")
}

func parseRenderWebhookSignatures(headerValue string) ([][]byte, error) {
	trimmed := strings.TrimSpace(headerValue)
	if trimmed == "" {
		return nil, fmt.Errorf("invalid signature header")
	}

	// Format follows Standard Webhooks header values and may include multiple signatures,
	// e.g. "v1,<sig>" or "v1,<sig1> v1,<sig2>".
	rawEntries := strings.Fields(trimmed)
	if len(rawEntries) == 0 {
		rawEntries = []string{trimmed}
	}

	signatures := make([][]byte, 0, len(rawEntries))
	seen := map[string]struct{}{}
	for _, entry := range rawEntries {
		parts := strings.Split(strings.TrimSpace(entry), ",")
		if len(parts) < 2 {
			continue
		}

		version := strings.TrimSpace(parts[0])
		if version != "v1" {
			continue
		}

		for _, encoded := range parts[1:] {
			signature := strings.TrimSpace(encoded)
			if signature == "" {
				continue
			}

			decoded, decodeErr := decodeRenderBase64(signature)
			if decodeErr != nil {
				continue
			}

			key := base64.StdEncoding.EncodeToString(decoded)
			if _, exists := seen[key]; exists {
				continue
			}

			seen[key] = struct{}{}
			signatures = append(signatures, decoded)
		}
	}

	if len(signatures) == 0 {
		return nil, fmt.Errorf("invalid signature")
	}

	return signatures, nil
}

func decodeRenderBase64(value string) ([]byte, error) {
	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, decoder := range decoders {
		decoded, err := decoder.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("failed to decode base64 value")
}

func renderSigningKeys(secret []byte) [][]byte {
	trimmedSecret := strings.TrimSpace(string(secret))
	if trimmedSecret == "" {
		return [][]byte{secret}
	}

	keys := [][]byte{[]byte(trimmedSecret)}
	encodedSecret := trimmedSecret
	switch {
	case strings.HasPrefix(trimmedSecret, "whsec_"):
		encodedSecret = strings.TrimPrefix(trimmedSecret, "whsec_")
	case strings.HasPrefix(trimmedSecret, "whsec-"):
		encodedSecret = strings.TrimPrefix(trimmedSecret, "whsec-")
	default:
		return keys
	}

	decodedSecret, err := decodeRenderBase64(encodedSecret)
	if err != nil || len(decodedSecret) == 0 {
		return keys
	}

	key := base64.StdEncoding.EncodeToString(decodedSecret)
	if slices.ContainsFunc(keys, func(existing []byte) bool {
		return base64.StdEncoding.EncodeToString(existing) == key
	}) {
		return keys
	}

	return append(keys, decodedSecret)
}

func matchesAnyRenderSignature(signatures [][]byte, expected []byte) bool {
	return slices.ContainsFunc(signatures, func(signature []byte) bool {
		return hmac.Equal(signature, expected)
	})
}

func renderPayloadType(eventType string) string {
	trimmedEventType := strings.TrimSpace(eventType)
	if trimmedEventType == "" {
		return "render.event"
	}

	parts := strings.Split(trimmedEventType, "_")
	dotCaseParts := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			continue
		}

		dotCaseParts = append(dotCaseParts, strings.ToLower(trimmedPart))
	}

	if len(dotCaseParts) == 0 {
		return "render.event"
	}

	return "render." + strings.Join(dotCaseParts, ".")
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
