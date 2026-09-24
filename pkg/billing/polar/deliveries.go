package polar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	defaultWebhookDeliveryPage  = 1
	defaultWebhookDeliveryLimit = 50
	maxWebhookDeliveryLimit     = 100
)

type WebhookDeliveryFilter struct {
	Succeeded *bool
	EventType string
	Page      int
	Limit     int
}

type WebhookDeliveryPage struct {
	Items []WebhookDelivery
	Total int
	Page  int
	Limit int
}

type WebhookDelivery struct {
	ID             string
	CreatedAt      string
	Succeeded      bool
	HTTPCode       *int
	Response       string
	EventType      string
	EventID        string
	EventSucceeded *bool
	Payload        string
}

type listWebhookDeliveriesJSON struct {
	Items      []webhookDeliveryJSON `json:"items"`
	Pagination struct {
		TotalCount int `json:"total_count"`
		MaxPage    int `json:"max_page"`
	} `json:"pagination"`
}

type webhookDeliveryJSON struct {
	ID           string           `json:"id"`
	CreatedAt    string           `json:"created_at"`
	Succeeded    bool             `json:"succeeded"`
	HTTPCode     *int             `json:"http_code"`
	Response     string           `json:"response"`
	WebhookEvent webhookEventJSON `json:"webhook_event"`
}

type webhookEventJSON struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Succeeded *bool           `json:"succeeded"`
	Payload   json.RawMessage `json:"payload"`
}

func (c *Client) ListWebhookDeliveries(ctx context.Context, filter WebhookDeliveryFilter) (*WebhookDeliveryPage, error) {
	page := filter.Page
	if page < 1 {
		page = defaultWebhookDeliveryPage
	}
	limit := filter.Limit
	if limit < 1 {
		limit = defaultWebhookDeliveryLimit
	}
	if limit > maxWebhookDeliveryLimit {
		limit = maxWebhookDeliveryLimit
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(limit))
	if filter.Succeeded != nil {
		query.Set("succeeded", strconv.FormatBool(*filter.Succeeded))
	}
	if eventType := strings.TrimSpace(filter.EventType); eventType != "" {
		query.Set("event_type", eventType)
	}

	var payload listWebhookDeliveriesJSON
	if err := c.get(ctx, "/webhooks/deliveries?"+query.Encode(), &payload); err != nil {
		return nil, err
	}

	items := make([]WebhookDelivery, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, item.toDelivery())
	}

	return &WebhookDeliveryPage{
		Items: items,
		Total: payload.Pagination.TotalCount,
		Page:  page,
		Limit: limit,
	}, nil
}

func (c *Client) RedeliverWebhookEvent(ctx context.Context, eventID string) error {
	id := strings.TrimSpace(eventID)
	if id == "" {
		return fmt.Errorf("webhook event id is required")
	}
	return c.post(ctx, "/webhooks/events/"+url.PathEscape(id)+"/redeliver", nil, nil)
}

func (item webhookDeliveryJSON) toDelivery() WebhookDelivery {
	return WebhookDelivery{
		ID:             strings.TrimSpace(item.ID),
		CreatedAt:      strings.TrimSpace(item.CreatedAt),
		Succeeded:      item.Succeeded,
		HTTPCode:       item.HTTPCode,
		Response:       strings.TrimSpace(item.Response),
		EventType:      strings.TrimSpace(item.WebhookEvent.Type),
		EventID:        strings.TrimSpace(item.WebhookEvent.ID),
		EventSucceeded: item.WebhookEvent.Succeeded,
		Payload:        payloadString(item.WebhookEvent.Payload),
	}
}

func payloadString(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}

	var asString string
	if err := json.Unmarshal(trimmed, &asString); err == nil {
		return asString
	}
	return string(trimmed)
}
