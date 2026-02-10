package dash0

import (
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// Dash0WebhookHandler manages SuperPlane webhook lifecycle for Dash0 triggers.
type Dash0WebhookHandler struct{}

// CompareConfig returns true when configuration A can satisfy configuration B.
func (h *Dash0WebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := OnAlertEventConfiguration{}
	configB := OnAlertEventConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	eventsA := uniqueNormalizedEventTypes(configA.EventTypes)
	eventsB := uniqueNormalizedEventTypes(configB.EventTypes)

	if len(eventsB) == 0 {
		return true, nil
	}

	for _, event := range eventsB {
		if !slices.Contains(eventsA, event) {
			return false, nil
		}
	}

	return true, nil
}

// Setup is a no-op because Dash0 webhook endpoints are configured manually in Dash0.
func (h *Dash0WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return map[string]any{}, nil
}

// Cleanup is a no-op because there are no external webhook resources to delete.
func (h *Dash0WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

// uniqueNormalizedEventTypes lowercases, trims, sorts, and deduplicates event types.
func uniqueNormalizedEventTypes(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		event := normalizeAlertEventType(value)
		if event == "" {
			continue
		}

		if !slices.Contains(normalized, event) {
			normalized = append(normalized, event)
		}
	}

	slices.Sort(normalized)
	return normalized
}
