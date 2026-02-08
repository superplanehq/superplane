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
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnResourceEventConfiguration struct {
	Service    string   `json:"service" mapstructure:"service"`
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

type ServiceMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OnResourceEventMetadata struct {
	Service *ServiceMetadata `json:"service"`
}

type WebhookConfiguration struct {
	Strategy     string   `json:"strategy" mapstructure:"strategy"`
	ResourceType string   `json:"resourceType,omitempty" mapstructure:"resourceType"`
	EventTypes   []string `json:"eventTypes,omitempty" mapstructure:"eventTypes"`
}

const (
	workspacePlanProfessional = "professional"
	workspacePlanOrganization = "organization"

	webhookStrategyIntegration  = "integration"
	webhookStrategyResourceType = "resource_type"

	webhookResourceTypeDeploy = "deploy"
	webhookResourceTypeBuild  = "build"

	webhookTimestampMaxSkew = 5 * time.Minute
)

func webhookConfigurationForResource(
	integration core.IntegrationContext,
	resourceType string,
	eventTypes []string,
) WebhookConfiguration {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: normalizeWebhookEventTypes(eventTypes),
		}
	}

	workspacePlan := metadata.workspacePlan()
	if workspacePlan != workspacePlanOrganization {
		return WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: normalizeWebhookEventTypes(eventTypes),
		}
	}

	normalizedResourceType := strings.ToLower(strings.TrimSpace(resourceType))
	switch normalizedResourceType {
	case webhookResourceTypeDeploy, webhookResourceTypeBuild:
		return WebhookConfiguration{
			Strategy:     webhookStrategyResourceType,
			ResourceType: normalizedResourceType,
			EventTypes:   normalizeWebhookEventTypes(eventTypes),
		}
	default:
		return WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: normalizeWebhookEventTypes(eventTypes),
		}
	}
}

func decodeWebhookConfiguration(configuration any) (WebhookConfiguration, error) {
	webhookConfiguration := WebhookConfiguration{
		Strategy: webhookStrategyIntegration,
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
			normalizedConfiguration.Strategy = webhookStrategyIntegration
		} else {
			normalizedConfiguration.Strategy = webhookStrategyResourceType
		}
	}

	if normalizedConfiguration.Strategy != webhookStrategyResourceType {
		return WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: normalizedConfiguration.EventTypes,
		}
	}

	switch normalizedConfiguration.ResourceType {
	case webhookResourceTypeDeploy, webhookResourceTypeBuild:
		return normalizedConfiguration
	default:
		return WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: normalizedConfiguration.EventTypes,
		}
	}
}

func webhookName(configuration WebhookConfiguration) string {
	configuration = normalizeWebhookConfiguration(configuration)
	if configuration.Strategy == webhookStrategyResourceType &&
		configuration.ResourceType == webhookResourceTypeDeploy {
		return "SuperPlane Deploy"
	}

	if configuration.Strategy == webhookStrategyResourceType &&
		configuration.ResourceType == webhookResourceTypeBuild {
		return "SuperPlane Build"
	}

	return "SuperPlane"
}

func webhookEventFilter(configuration WebhookConfiguration) []string {
	configuration = normalizeWebhookConfiguration(configuration)
	allowedEventTypes := allowedEventTypesForWebhook(configuration)
	requestedEventTypes := filterAllowedEventTypes(configuration.EventTypes, allowedEventTypes)
	if len(requestedEventTypes) > 0 {
		return requestedEventTypes
	}

	defaultEventTypes := defaultEventTypesForWebhook(configuration)
	if len(defaultEventTypes) > 0 {
		return defaultEventTypes
	}

	return allowedEventTypes
}

func combineDeployAndBuildEventTypes(deploy, build []string) []string {
	out := make([]string, 0, len(deploy)+len(build))
	out = append(out, deploy...)
	out = append(out, build...)
	return normalizeWebhookEventTypes(out)
}

func allowedEventTypesForWebhook(configuration WebhookConfiguration) []string {
	if configuration.Strategy == webhookStrategyResourceType {
		switch configuration.ResourceType {
		case webhookResourceTypeDeploy:
			return deployAllowedEventTypes
		case webhookResourceTypeBuild:
			return buildAllowedEventTypes
		}
	}
	return combineDeployAndBuildEventTypes(deployAllowedEventTypes, buildAllowedEventTypes)
}

func defaultEventTypesForWebhook(configuration WebhookConfiguration) []string {
	if configuration.Strategy == webhookStrategyResourceType {
		switch configuration.ResourceType {
		case webhookResourceTypeDeploy:
			return normalizeWebhookEventTypes(deployDefaultEventTypes)
		case webhookResourceTypeBuild:
			return normalizeWebhookEventTypes(buildDefaultEventTypes)
		}
	}
	return combineDeployAndBuildEventTypes(deployDefaultEventTypes, buildDefaultEventTypes)
}

func webhookConfigurationsEqual(a, b WebhookConfiguration) bool {
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
			Name:     "service",
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

	config.Service = strings.TrimSpace(config.Service)
	config.EventTypes = normalizeWebhookEventTypes(config.EventTypes)
	return config, nil
}

func ensureServiceInMetadata(ctx core.TriggerContext, config OnResourceEventConfiguration) error {
	serviceValue := strings.TrimSpace(config.Service)
	if serviceValue == "" {
		return fmt.Errorf("service is required")
	}

	nodeMetadata := OnResourceEventMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if nodeMetadata.Service != nil &&
		(nodeMetadata.Service.ID == serviceValue || nodeMetadata.Service.Name == serviceValue) {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	workspaceID, err := workspaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return err
	}

	services, err := client.ListServices(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to list Render services: %w", err)
	}

	service := findService(services, serviceValue)
	if service == nil {
		return fmt.Errorf("service %s is not accessible with this API key", serviceValue)
	}

	return ctx.Metadata.Set(OnResourceEventMetadata{
		Service: &ServiceMetadata{
			ID:   service.ID,
			Name: service.Name,
		},
	})
}

func findService(services []Service, value string) *Service {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil
	}

	idIndex := slices.IndexFunc(services, func(service Service) bool {
		return strings.TrimSpace(service.ID) == trimmedValue
	})
	if idIndex >= 0 {
		return &services[idIndex]
	}

	nameIndex := slices.IndexFunc(services, func(service Service) bool {
		return strings.EqualFold(strings.TrimSpace(service.Name), trimmedValue)
	})
	if nameIndex < 0 {
		return nil
	}

	return &services[nameIndex]
}

func handleOnResourceEventWebhook(
	ctx core.WebhookRequestContext,
	config OnResourceEventConfiguration,
	allowedEventTypes []string,
	defaultEventTypes []string,
) (int, error) {
	if err := verifyWebhookSignature(ctx); err != nil {
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
	if config.Service == "" || serviceID == "" || config.Service != serviceID {
		return http.StatusOK, nil
	}

	selectedEventTypes := filterAllowedEventTypes(config.EventTypes, allowedEventTypes)
	if len(selectedEventTypes) == 0 {
		selectedEventTypes = defaultEventTypes
	}

	if !slices.Contains(selectedEventTypes, eventType) {
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit(payloadType(eventType), data); err != nil {
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

	webhookID := strings.TrimSpace(ctx.Headers.Get("webhook-id"))
	webhookTimestamp := strings.TrimSpace(ctx.Headers.Get("webhook-timestamp"))
	signatureHeader := strings.TrimSpace(ctx.Headers.Get("webhook-signature"))

	if webhookID == "" || webhookTimestamp == "" || signatureHeader == "" {
		return fmt.Errorf("missing signature headers")
	}

	timestamp, err := parseWebhookTimestamp(webhookTimestamp)
	if err != nil {
		return fmt.Errorf("invalid webhook timestamp")
	}

	if absDuration(time.Now().UTC().Sub(timestamp)) > webhookTimestampMaxSkew {
		return fmt.Errorf("webhook timestamp expired")
	}

	signatures, err := parseWebhookSignatures(signatureHeader)
	if err != nil {
		return err
	}

	signingKeys := signingKeys(secret)
	payloadPrefix := webhookID + "." + webhookTimestamp + "."
	secretText := []byte(strings.TrimSpace(string(secret)))

	for _, key := range signingKeys {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(payloadPrefix))
		h.Write(ctx.Body)
		if matchesAnySignature(signatures, h.Sum(nil)) {
			return nil
		}

		// Compatibility fallback for providers that document signature input as:
		// webhook-id.webhook-timestamp.body.webhook-secret
		h = hmac.New(sha256.New, key)
		h.Write([]byte(payloadPrefix))
		h.Write(ctx.Body)
		h.Write([]byte("."))
		h.Write(secretText)
		if matchesAnySignature(signatures, h.Sum(nil)) {
			return nil
		}
	}

	return fmt.Errorf("invalid signature")
}

func parseWebhookSignatures(headerValue string) ([][]byte, error) {
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

			decoded, decodeErr := decodeBase64(signature)
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

func decodeBase64(value string) ([]byte, error) {
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

func signingKeys(secret []byte) [][]byte {
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

	decodedSecret, err := decodeBase64(encodedSecret)
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

func matchesAnySignature(signatures [][]byte, expected []byte) bool {
	return slices.ContainsFunc(signatures, func(signature []byte) bool {
		return hmac.Equal(signature, expected)
	})
}

func payloadType(eventType string) string {
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

func parseWebhookTimestamp(value string) (time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return time.Time{}, fmt.Errorf("missing timestamp")
	}

	seconds, err := strconv.ParseInt(trimmedValue, 10, 64)
	if err == nil {
		return time.Unix(seconds, 0).UTC(), nil
	}

	timestamp, err := time.Parse(time.RFC3339Nano, trimmedValue)
	if err == nil {
		return timestamp.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("invalid timestamp")
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}

	return value
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
