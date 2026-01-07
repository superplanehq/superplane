package pagerduty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	AuthType string
	Token    string
	BaseURL  string
}

func NewClient(ctx core.AppInstallationContext) (*Client, error) {
	authType, err := ctx.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("error finding auth type: %v", err)
	}

	switch string(authType) {
	case AuthTypeAPIToken:
		apiToken, err := ctx.GetConfig("apiToken")
		if err != nil {
			return nil, err
		}

		return &Client{
			Token:    string(apiToken),
			AuthType: AuthTypeAPIToken,
			BaseURL:  "https://api.pagerduty.com",
		}, nil

	case AuthTypeAppOAuth:
		secrets, err := ctx.GetSecrets()
		if err != nil {
			return nil, fmt.Errorf("failed to get secrets: %v", err)
		}

		var accessToken string
		for _, secret := range secrets {
			if secret.Name == AppAccessToken {
				accessToken = string(secret.Value)
				break
			}
		}

		if accessToken == "" {
			return nil, fmt.Errorf("app OAuth access token not found")
		}

		return &Client{
			Token:    accessToken,
			AuthType: AuthTypeAppOAuth,
			BaseURL:  "https://api.pagerduty.com",
		}, nil
	}

	return nil, fmt.Errorf("unknown auth type %s", authType)
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	if c.AuthType == AuthTypeAppOAuth {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", c.Token))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type CreateIncidentRequest struct {
	Incident IncidentPayload `json:"incident"`
}

type IncidentPayload struct {
	Type    string           `json:"type"` // "incident"
	Title   string           `json:"title"`
	Service ServiceReference `json:"service"`
	Urgency string           `json:"urgency,omitempty"` // "high" or "low"
	Body    *IncidentBody    `json:"body,omitempty"`
}

type ServiceReference struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "service_reference"
}

type IncidentBody struct {
	Type    string `json:"type"` // "incident_body"
	Details string `json:"details"`
}

func (c *Client) CreateIncident(title, service, urgency, description string) (any, error) {
	request := CreateIncidentRequest{
		Incident: IncidentPayload{
			Type:    "incident",
			Title:   title,
			Urgency: urgency,
			Service: ServiceReference{
				ID:   service,
				Type: "service_reference",
			},
		},
	}

	if description != "" {
		request.Incident.Body = &IncidentBody{
			Type:    "incident_body",
			Details: description,
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	url := fmt.Sprintf("%s/incidents", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

type WebhookSubscription struct {
	ID             string         `json:"id"`
	Events         []string       `json:"events"`
	DeliveryMethod DeliveryMethod `json:"delivery_method"`
}

type DeliveryMethod struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

func (c *Client) CreateWebhookSubscription(url string, events []string, filter WebhookFilter) (*WebhookSubscription, error) {
	apiURL := fmt.Sprintf("%s/webhook_subscriptions", c.BaseURL)
	subscription := map[string]any{
		"webhook_subscription": map[string]any{
			"type": "webhook_subscription",
			"delivery_method": map[string]any{
				"type": "http_delivery_method",
				"url":  url,
			},
			"events": events,
			"filter": filter,
		},
	}

	body, err := json.Marshal(subscription)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		WebhookSubscription WebhookSubscription `json:"webhook_subscription"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.WebhookSubscription, nil
}

func (c *Client) DeleteWebhookSubscription(id string) error {
	url := fmt.Sprintf("%s/webhook_subscriptions/%s", c.BaseURL, id)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

type Service struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

func (c *Client) ListServices() ([]Service, error) {
	apiURL := fmt.Sprintf("%s/services", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Services []Service `json:"services"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Services, nil
}

func (c *Client) GetService(id string) (*Service, error) {
	apiURL := fmt.Sprintf("%s/services/%s", c.BaseURL, id)
	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Service Service `json:"service"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Service, nil
}
