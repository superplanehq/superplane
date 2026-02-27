package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type WebhookHandler struct{}

func (h *WebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.WorkspaceID == configB.WorkspaceID, nil
}

func (h *WebhookHandler) Merge(current, requested any) (any, bool, error) {
	return requested, false, nil
}

func (h *WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return nil, err
	}

	config := WebhookConfiguration{}
	configuration := ctx.Webhook.GetConfiguration()
	if configuration != nil {
		if err := mapstructure.Decode(configuration, &config); err != nil {
			return nil, fmt.Errorf("invalid webhook configuration: %w", err)
		}
	}

	resolvedWsId, err := client.ResolveWorkspaceID(context.Background(), config.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	targetURL := ctx.Webhook.GetURL()

	webhookSecretBytes, err := ctx.Integration.GetConfig("webhookSecret")
	var webhookSecret string
	if err == nil {
		webhookSecret = string(webhookSecretBytes)
	}

	// Check if webhook already exists and update it if needed
	listOpts := &tfe.NotificationConfigurationListOptions{}
	existingHooks, err := client.TFE.NotificationConfigurations.List(context.Background(), resolvedWsId, listOpts)
	if err == nil && existingHooks != nil {
		for _, hook := range existingHooks.Items {
			if hook.URL == targetURL {
				// Update the existing webhook to ensure HMAC token is current
				updateOpts := tfe.NotificationConfigurationUpdateOptions{
					Enabled: tfe.Bool(true),
					Name:    tfe.String("SuperPlane"),
				}
				if webhookSecret != "" {
					updateOpts.Token = tfe.String(webhookSecret)
				}

				_, err := client.TFE.NotificationConfigurations.Update(context.Background(), hook.ID, updateOpts)
				if err != nil {
					return nil, fmt.Errorf("failed to update existing webhook: %w", err)
				}

				return map[string]string{
					"notification_configuration_id": hook.ID,
				}, nil
			}
		}
	}

	dType := tfe.NotificationDestinationType("generic")
	createOpts := tfe.NotificationConfigurationCreateOptions{
		DestinationType: &dType,
		Enabled:         tfe.Bool(true),
		Name:            tfe.String("SuperPlane"),
		URL:             tfe.String(targetURL),
		Triggers: []tfe.NotificationTriggerType{
			tfe.NotificationTriggerCreated,
			tfe.NotificationTriggerPlanning,
			tfe.NotificationTriggerNeedsAttention,
			tfe.NotificationTriggerApplying,
			tfe.NotificationTriggerCompleted,
			tfe.NotificationTriggerErrored,
			tfe.NotificationTriggerAssessmentDrifted,
			tfe.NotificationTriggerAssessmentFailed,
		},
	}

	if webhookSecret != "" {
		createOpts.Token = tfe.String(webhookSecret)
	}

	var nc *tfe.NotificationConfiguration
	var createErr error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		nc, createErr = client.TFE.NotificationConfigurations.Create(context.Background(), resolvedWsId, createOpts)
		if createErr == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	if createErr != nil {
		return nil, fmt.Errorf("failed to create terraform webhook after %d attempts: %w", maxRetries, createErr)
	}

	return map[string]string{
		"notification_configuration_id": nc.ID,
	}, nil
}

func (h *WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	metadata, ok := ctx.Webhook.GetMetadata().(map[string]any)
	if !ok {
		return nil
	}

	idVal, ok := metadata["notification_configuration_id"]
	if !ok {
		return nil
	}

	id, ok := idVal.(string)
	if !ok {
		return fmt.Errorf("notification_configuration_id is not a string")
	}
	err = client.TFE.NotificationConfigurations.Delete(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to delete terraform webhook %s: %w", id, err)
	}

	return nil
}

func ParseAndValidateWebhook(ctx core.WebhookRequestContext) (map[string]any, int, error) {
	webhookSecretBytes, err := ctx.Integration.GetConfig("webhookSecret")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to get webhook secret: %w", err)
	}

	webhookSecret := strings.TrimSpace(string(webhookSecretBytes))
	if webhookSecret == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("webhook secret is not configured")
	}

	signature := ctx.Headers.Get("X-TFE-Notification-Signature")
	if signature == "" {
		return nil, http.StatusUnauthorized, fmt.Errorf("missing signature header")
	}

	if err := crypto.VerifySignatureSHA512([]byte(webhookSecret), ctx.Body, signature); err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("invalid HMAC-SHA512 signature: %w", err)
	}

	var base map[string]any
	if err := json.Unmarshal(ctx.Body, &base); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid json representation: %w", err)
	}

	versionFloat, ok := base["payload_version"].(float64)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("missing or invalid payload_version")
	}

	if notifs, ok := base["notifications"].([]any); ok && len(notifs) > 0 {
		for _, n := range notifs {
			if noteMap, ok := n.(map[string]any); ok {
				if trigger, ok := noteMap["trigger"].(string); ok && trigger == "verification" {
					return nil, http.StatusOK, nil
				}
			}
		}
	}

	var eventData RunEventData

	switch int(versionFloat) {
	case 1:
		var p1 PayloadVersion1
		if err := json.Unmarshal(ctx.Body, &p1); err != nil {
			return nil, http.StatusBadRequest, err
		}
		if len(p1.Notifications) > 0 {
			eventData = RunEventData{
				RunID:            p1.RunID,
				RunURL:           p1.RunURL,
				RunMessage:       p1.RunMessage,
				WorkspaceID:      p1.WorkspaceID,
				WorkspaceName:    p1.WorkspaceName,
				OrganizationName: p1.OrganizationName,
				Action:           p1.Notifications[0].Trigger,
				RunStatus:        p1.Notifications[0].RunStatus,
				RunCreatedBy:     p1.RunCreatedBy,
			}
		}
	case 2:
		var p2 PayloadVersion2
		if err := json.Unmarshal(ctx.Body, &p2); err != nil {
			return nil, http.StatusBadRequest, err
		}
		if len(p2.Notifications) > 0 {
			eventData = RunEventData{
				WorkspaceID:      p2.WorkspaceID,
				WorkspaceName:    p2.WorkspaceName,
				OrganizationName: p2.OrganizationName,
				Action:           p2.Notifications[0].Trigger,
				RunStatus:        p2.Notifications[0].RunStatus,
			}
		}
	default:
		return nil, http.StatusBadRequest, fmt.Errorf("unsupported payload_version: %.0f", versionFloat)
	}

	var out map[string]any
	evBytes, _ := json.Marshal(eventData)
	json.Unmarshal(evBytes, &out)

	return out, http.StatusOK, nil
}
