package terraform

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
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
	currentConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}

	requestedConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(requested, &requestedConfig); err != nil {
		return nil, false, err
	}

	mergedConfig := WebhookConfiguration{
		WorkspaceID: currentConfig.WorkspaceID,
		Events:      requestedConfig.Events,
	}

	// Check if the configuration actually changed
	changed, _ := h.CompareConfig(currentConfig, mergedConfig)
	// CompareConfig returns true if configs are equal, so we invert it
	return mergedConfig, !changed, nil
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

	resolvedWsId, err := client.ResolveWorkspaceID(config.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	targetURL := ctx.Webhook.GetURL()

	var webhookSecretBytes []byte
	secrets, err := ctx.Integration.GetSecrets()
	if err == nil {
		for _, s := range secrets {
			if s.Name == "webhookSecret" {
				webhookSecretBytes = s.Value
				break
			}
		}
	}
	webhookSecret := strings.TrimSpace(string(webhookSecretBytes))

	triggers := []string{
		"run:created",
		"run:planning",
		"run:needs_attention",
		"run:applying",
		"run:completed",
		"run:errored",
	}

	listPath := fmt.Sprintf("/api/v2/workspaces/%s/notification-configurations", resolvedWsId)
	req, err := client.newRequest(http.MethodGet, listPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		var listPayload struct {
			Data []struct {
				ID         string `json:"id"`
				Attributes struct {
					URL string `json:"url"`
				} `json:"attributes"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&listPayload); err == nil {
			for _, hook := range listPayload.Data {
				if hook.Attributes.URL == targetURL {
					updateOpts := map[string]any{
						"data": map[string]any{
							"type": "notification-configurations",
							"attributes": map[string]any{
								"enabled":          true,
								"name":             "SuperPlane",
								"destination-type": "generic",
								"url":              targetURL,
								"triggers":         triggers,
							},
						},
					}
					if webhookSecret != "" {
						updateOpts["data"].(map[string]any)["attributes"].(map[string]any)["token"] = webhookSecret
					}

					updatePath := fmt.Sprintf("/api/v2/notification-configurations/%s", url.PathEscape(hook.ID))
					uReq, err := client.newRequest(http.MethodPatch, updatePath, updateOpts)
					if err != nil {
						return nil, fmt.Errorf("failed to create update request: %w", err)
					}

					uResp, err := client.HTTPClient.Do(uReq)
					if err != nil {
						return nil, fmt.Errorf("failed to update existing webhook: %w", err)
					}
					defer func() { _ = uResp.Body.Close() }()

					if uResp.StatusCode >= 400 {
						return nil, fmt.Errorf("failed to update existing webhook, status code: %d", uResp.StatusCode)
					}

					return map[string]string{
						"notification_configuration_id": hook.ID,
					}, nil
				}
			}
		}
	}

	createOpts := map[string]any{
		"data": map[string]any{
			"type": "notification-configurations",
			"attributes": map[string]any{
				"enabled":          true,
				"name":             "SuperPlane",
				"destination-type": "generic",
				"url":              targetURL,
				"triggers":         triggers,
			},
		},
	}

	if webhookSecret != "" {
		createOpts["data"].(map[string]any)["attributes"].(map[string]any)["token"] = webhookSecret
	}

	var ncID string
	var createErr error
	maxRetries := 3
	attempts := 0

	for i := 0; i < maxRetries; i++ {
		attempts++
		cReq, err := client.newRequest(http.MethodPost, listPath, createOpts)
		if err != nil {
			createErr = err
			break
		}

		cResp, err := client.HTTPClient.Do(cReq)
		if err != nil {
			createErr = err
			if cResp != nil && cResp.Body != nil {
				_ = cResp.Body.Close()
			}
		} else {
			if cResp.StatusCode >= 400 {
				bodyBytes, _ := io.ReadAll(cResp.Body)
				createErr = fmt.Errorf("bad status code: %d, body: %s", cResp.StatusCode, string(bodyBytes))
				cResp.Body.Close()
				if cResp.StatusCode < 500 && cResp.StatusCode != 429 {
					break
				}
			} else {
				var createPayload struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				}
				if err := json.NewDecoder(cResp.Body).Decode(&createPayload); err != nil {
					createErr = err
				} else {
					ncID = createPayload.Data.ID
					createErr = nil
				}
				cResp.Body.Close()
			}

			if createErr == nil {
				break
			}
		}

		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	if createErr != nil {
		return nil, fmt.Errorf("failed to create terraform webhook after %d attempts: %w", attempts, createErr)
	}

	return map[string]string{
		"notification_configuration_id": ncID,
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
	deletePath := fmt.Sprintf("/api/v2/notification-configurations/%s", url.PathEscape(id))
	req, err := client.newRequest(http.MethodDelete, deletePath, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete terraform webhook %s: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to delete terraform webhook %s, status: %d", id, resp.StatusCode)
	}

	return nil
}

func ParseAndValidateWebhook(ctx core.WebhookRequestContext) (map[string]any, int, error) {
	var webhookSecretBytes []byte
	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch secrets: %w", err)
	}
	for _, s := range secrets {
		if s.Name == "webhookSecret" {
			webhookSecretBytes = s.Value
			break
		}
	}
	if len(webhookSecretBytes) == 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to get webhook secret or none configured")
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
			if len(p1.Notifications) > 1 {
				log.Warnf("terraform webhook payload v1 contains %d notifications, only the first will be processed; all: %+v", len(p1.Notifications), p1.Notifications)
			}
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
			if len(p2.Notifications) > 1 {
				log.Warnf("terraform webhook payload v2 contains %d notifications, only the first will be processed; all: %+v", len(p2.Notifications), p2.Notifications)
			}
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
	evBytes, err := json.Marshal(eventData)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal event data: %w", err)
	}
	if err := json.Unmarshal(evBytes, &out); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return out, http.StatusOK, nil
}
