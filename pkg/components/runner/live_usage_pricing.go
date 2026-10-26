package runner

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// HostedUsagePricing prices live display usage without writing a billing event.
type HostedUsagePricing struct {
	Rate      billing.Rate `json:"rate"`
	MarkupBPS int          `json:"markup_bps"`
}

// ResolveHostedUsagePricing returns display pricing for a hosted execution.
func ResolveHostedUsagePricing(tx *gorm.DB, orgID uuid.UUID, component string, configuration map[string]any) (*HostedUsagePricing, error) {
	source := configurationCredentialsSource(configuration)
	if !IsHostedCredentials(source) && !(source == "" && component == models.SuperPlaneRunnerComponent) {
		return nil, nil
	}
	provider := configurationString(configuration, "hostedProvider")
	model := configurationString(configuration, "model")
	if selected, err := models.ParseHostedLLMModelKey(model); err == nil && selected.IsSet() {
		provider, model = selected.Provider, selected.Model
	}
	rate, ok := billing.LookupRate(provider, model)
	if !ok {
		return nil, nil
	}
	markup, err := billing.ResolveMarkupBPS(tx, orgID)
	if err != nil {
		return nil, err
	}
	return &HostedUsagePricing{Rate: rate, MarkupBPS: markup}, nil
}
