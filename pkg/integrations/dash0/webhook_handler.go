package dash0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
)

const notificationChannelIDLabel = "dash0.com/id"

// NotificationChannel is a summarized view of a Dash0 notification channel.
type NotificationChannel struct {
	ID   string
	Name string
}

// NotificationChannelDefinition is the CRD-enveloped notification channel resource.
type NotificationChannelDefinition struct {
	Kind     string                      `json:"kind"`
	Metadata NotificationChannelMetadata `json:"metadata"`
	Spec     NotificationChannelSpec     `json:"spec"`
}

type NotificationChannelMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

type NotificationChannelSpec struct {
	Type    string                           `json:"type"`
	Config  NotificationChannelWebhookConfig `json:"config"`
	Routing *NotificationChannelRouting      `json:"routing,omitempty"`
}

type NotificationChannelWebhookConfig struct {
	URL string `json:"url"`
}

type NotificationChannelRouting struct {
	Assets  []NotificationChannelRoutingAsset      `json:"assets"`
	Filters [][]NotificationChannelAttributeFilter `json:"filters"`
}

type NotificationChannelRoutingAsset struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Dataset string `json:"dataset"`
}

type NotificationChannelAttributeFilter struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
}

func notificationChannelName(integrationID uuid.UUID) string {
	return fmt.Sprintf("SuperPlane (%s)", integrationID.String())
}

func buildNotificationChannelDefinition(name, webhookURL string) NotificationChannelDefinition {
	return NotificationChannelDefinition{
		Kind: "Dash0NotificationChannel",
		Metadata: NotificationChannelMetadata{
			Name: name,
		},
		Spec: NotificationChannelSpec{
			Type: "webhook",
			Config: NotificationChannelWebhookConfig{
				URL: webhookURL,
			},
			Routing: &NotificationChannelRouting{
				Assets: []NotificationChannelRoutingAsset{},
				Filters: [][]NotificationChannelAttributeFilter{
					{
						{
							Key:      "dash0.failed_check.max_status",
							Operator: "is_any",
						},
					},
				},
			},
		},
	}
}

func extractNotificationChannelID(def NotificationChannelDefinition) string {
	if def.Metadata.Labels != nil {
		if id := def.Metadata.Labels[notificationChannelIDLabel]; id != "" {
			return id
		}
	}
	return ""
}

// CreateNotificationChannel creates a webhook notification channel with match-all routing.
func (c *Client) CreateNotificationChannel(name, webhookURL string) (string, error) {
	apiURL := fmt.Sprintf("%s/api/notification-channels", c.BaseURL)
	body, err := json.Marshal(buildNotificationChannelDefinition(name, webhookURL))
	if err != nil {
		return "", fmt.Errorf("error marshalling notification channel: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, apiURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return "", err
	}

	var response NotificationChannelDefinition
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("error parsing notification channel response: %v", err)
	}

	id := extractNotificationChannelID(response)
	if id == "" {
		return "", fmt.Errorf("notification channel created but response did not include %s", notificationChannelIDLabel)
	}

	return id, nil
}

// ListNotificationChannels returns all notification channels for the organization.
func (c *Client) ListNotificationChannels() ([]NotificationChannel, error) {
	apiURL := fmt.Sprintf("%s/api/notification-channels", c.BaseURL)

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil, "")
	if err != nil {
		return nil, err
	}

	var definitions []NotificationChannelDefinition
	if err := json.Unmarshal(responseBody, &definitions); err != nil {
		return nil, fmt.Errorf("error parsing notification channels response: %v", err)
	}

	channels := make([]NotificationChannel, 0, len(definitions))
	for _, def := range definitions {
		id := extractNotificationChannelID(def)
		if id == "" {
			continue
		}
		channels = append(channels, NotificationChannel{
			ID:   id,
			Name: def.Metadata.Name,
		})
	}

	return channels, nil
}

// UpdateNotificationChannel updates an existing notification channel's webhook URL and routing.
func (c *Client) UpdateNotificationChannel(originOrID, name, webhookURL string) error {
	apiURL := fmt.Sprintf("%s/api/notification-channels/%s", c.BaseURL, url.PathEscape(originOrID))
	body, err := json.Marshal(buildNotificationChannelDefinition(name, webhookURL))
	if err != nil {
		return fmt.Errorf("error marshalling notification channel: %v", err)
	}

	_, err = c.execRequest(http.MethodPut, apiURL, bytes.NewReader(body), "application/json")
	return err
}

// DeleteNotificationChannel deletes a notification channel. A 404 is treated as success.
func (c *Client) DeleteNotificationChannel(originOrID string) error {
	apiURL := fmt.Sprintf("%s/api/notification-channels/%s", c.BaseURL, url.PathEscape(originOrID))

	req, err := http.NewRequest(http.MethodDelete, apiURL, nil)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(res.Body, MaxResponseSize))
		return fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return nil
}

// provisionNotificationChannel finds or creates the SuperPlane notification channel for an integration.
func provisionNotificationChannel(client *Client, integration core.IntegrationContext, webhookURL string) (string, error) {
	name := notificationChannelName(integration.ID())

	var existing Metadata
	if meta := integration.GetMetadata(); meta != nil {
		_ = mapstructure.Decode(meta, &existing)
	}

	if existing.NotificationChannelID != "" {
		if err := client.UpdateNotificationChannel(existing.NotificationChannelID, name, webhookURL); err != nil {
			return "", fmt.Errorf("error updating notification channel: %w", err)
		}
		return existing.NotificationChannelID, nil
	}

	channels, err := client.ListNotificationChannels()
	if err != nil {
		return "", fmt.Errorf("error listing notification channels: %w", err)
	}

	for _, channel := range channels {
		if channel.Name == name {
			if err := client.UpdateNotificationChannel(channel.ID, name, webhookURL); err != nil {
				return "", fmt.Errorf("error updating notification channel: %w", err)
			}
			return channel.ID, nil
		}
	}

	id, err := client.CreateNotificationChannel(name, webhookURL)
	if err != nil {
		return "", fmt.Errorf("error creating notification channel: %w", err)
	}

	return id, nil
}

// deleteProvisionedNotificationChannel removes the notification channel created for an integration.
func deleteProvisionedNotificationChannel(client *Client, integration core.IntegrationContext, logger *logrus.Entry) error {
	var metadata Metadata
	if meta := integration.GetMetadata(); meta != nil {
		if err := mapstructure.Decode(meta, &metadata); err != nil {
			return fmt.Errorf("failed to decode metadata: %w", err)
		}
	}

	if metadata.NotificationChannelID == "" {
		return nil
	}

	if err := client.DeleteNotificationChannel(metadata.NotificationChannelID); err != nil {
		if logger != nil {
			logger.Warnf("failed to delete dash0 notification channel %s: %v", metadata.NotificationChannelID, err)
		}
		return err
	}

	return nil
}
