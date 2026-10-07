package linear

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	// TeamID is the single team used by triggers that listen to one team.
	TeamID string `json:"teamId" mapstructure:"teamId"`
	// TeamIDs is the set of teams a project-scoped trigger listens on.
	// Linear cannot subscribe a webhook to a project, so each team gets
	// its own webhook and SuperPlane filters by project.
	TeamIDs      []string `json:"teamIds" mapstructure:"teamIds"`
	ResourceType string   `json:"resourceType" mapstructure:"resourceType"`
}

type WebhookMetadata struct {
	// ID is the first Linear webhook. Older rows store only this field.
	ID string `json:"id" mapstructure:"id"`
	// IDs holds every Linear webhook this subscription created.
	IDs []string `json:"ids" mapstructure:"ids"`
	// AppLevel is true when Linear delivers events to the OAuth application
	// webhook instead of a webhook created for this subscription.
	AppLevel bool `json:"appLevel,omitempty" mapstructure:"appLevel,omitempty"`
}

func (c WebhookConfiguration) resolvedTeamIDs() []string {
	ids := normalizeIDs(c.TeamIDs)
	if len(ids) > 0 {
		slices.Sort(ids)
		return ids
	}
	if teamID := strings.TrimSpace(c.TeamID); teamID != "" {
		return []string{teamID}
	}
	return nil
}

func (m WebhookMetadata) webhookIDs() []string {
	ids := normalizeIDs(m.IDs)
	if id := strings.TrimSpace(m.ID); id != "" && !slices.Contains(ids, id) {
		ids = append(ids, id)
	}
	return ids
}

type LinearWebhookHandler struct{}

func (h *LinearWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *LinearWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	if !slices.Equal(configA.resolvedTeamIDs(), configB.resolvedTeamIDs()) {
		return false, nil
	}

	return configA.ResourceType == configB.ResourceType, nil
}

func (h *LinearWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	if AppWebhookSigningSecret(ctx.Integration) != "" {
		return &WebhookMetadata{AppLevel: true}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook config: %v", err)
	}

	//
	// Linear generates a signing secret when one is not supplied, but it accepts
	// ours, so we keep SuperPlane's webhook secret as the single source of truth.
	//
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	teamIDs := config.resolvedTeamIDs()
	if len(teamIDs) == 0 {
		teamIDs = []string{""}
	}

	ids := make([]string, 0, len(teamIDs))
	for _, teamID := range teamIDs {
		webhook, err := client.CreateWebhook(
			ctx.Webhook.GetURL(),
			string(secret),
			"SuperPlane",
			teamID,
			[]string{config.ResourceType},
		)
		if err != nil {
			for _, id := range ids {
				_ = client.DeleteWebhook(id)
			}
			return nil, fmt.Errorf("error creating webhook: %v", err)
		}
		ids = append(ids, webhook.ID)
	}

	return &WebhookMetadata{ID: ids[0], IDs: ids}, nil
}

func (h *LinearWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %v", err)
	}

	ids := metadata.webhookIDs()
	if len(ids) == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	for _, id := range ids {
		if err := client.DeleteWebhook(id); err != nil {
			return fmt.Errorf("error deleting webhook: %v", err)
		}
	}

	return nil
}
