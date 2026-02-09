package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WebhookListResponse is the response from listing webhooks.
type WebhookListResponse struct {
	StartAt    int              `json:"startAt"`
	MaxResults int              `json:"maxResults"`
	Total      int              `json:"total"`
	Values     []WebhookDetails `json:"values"`
}

// WebhookDetails contains information about a registered webhook.
type WebhookDetails struct {
	ID             int64    `json:"id"`
	JQLFilter      string   `json:"jqlFilter"`
	FieldIDsFilter []string `json:"fieldIdsFilter"`
	ExpirationDate string   `json:"expirationDate"`
	Events         []string `json:"events"`
}

// ListWebhooks returns all webhooks registered for this OAuth app.
func (c *Client) ListWebhooks() (*WebhookListResponse, error) {
	url := c.apiURL("/rest/api/3/webhook")
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response WebhookListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing webhook list response: %v", err)
	}

	return &response, nil
}

// DeleteWebhookByID deletes a single webhook by its Jira ID.
func (c *Client) DeleteWebhookByID(webhookID int64) error {
	return c.DeleteWebhook([]int64{webhookID})
}

// DeleteAllWebhooks deletes all webhooks registered for this OAuth app.
func (c *Client) DeleteAllWebhooks() error {
	webhooks, err := c.ListWebhooks()
	if err != nil {
		return fmt.Errorf("error listing webhooks: %v", err)
	}

	if len(webhooks.Values) == 0 {
		return nil
	}

	ids := make([]int64, len(webhooks.Values))
	for i, w := range webhooks.Values {
		ids[i] = w.ID
	}

	return c.DeleteWebhook(ids)
}
