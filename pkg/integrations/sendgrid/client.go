package sendgrid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://api.sendgrid.com/v3"

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	if len(apiKey) == 0 {
		return nil, fmt.Errorf("apiKey is required")
	}

	return &Client{
		APIKey:  string(apiKey),
		BaseURL: defaultBaseURL,
		http:    httpClient,
	}, nil
}

func (c *Client) Verify() error {
	_, _, err := c.execRequestWithResponse(http.MethodGet, c.BaseURL+"/user/profile", nil)
	return err
}

type EventWebhookSettings struct {
	Enabled          bool   `json:"enabled"`
	URL              string `json:"url"`
	Processed        bool   `json:"processed"`
	Delivered        bool   `json:"delivered"`
	Deferred         bool   `json:"deferred"`
	Bounce           bool   `json:"bounce"`
	Dropped          bool   `json:"dropped"`
	Open             bool   `json:"open"`
	Click            bool   `json:"click"`
	SpamReport       bool   `json:"spam_report"`
	Unsubscribe      bool   `json:"unsubscribe"`
	GroupUnsubscribe bool   `json:"group_unsubscribe"`
	GroupResubscribe bool   `json:"group_resubscribe"`
}

type SignedWebhookSettings struct {
	Enabled bool `json:"enabled"`
}

type SignedWebhookResponse struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key"`
}

func (c *Client) UpdateEventWebhookSettings(settings EventWebhookSettings) error {
	body, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook settings: %v", err)
	}

	_, _, err = c.execRequestWithResponse(http.MethodPatch, c.BaseURL+"/user/webhooks/event/settings", bytes.NewReader(body))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetEventWebhookSettings() (*EventWebhookSettings, error) {
	_, responseBody, err := c.execRequestWithResponse(http.MethodGet, c.BaseURL+"/user/webhooks/event/settings", nil)
	if err != nil {
		return nil, err
	}

	var settings EventWebhookSettings
	if err := json.Unmarshal(responseBody, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook settings: %v", err)
	}

	return &settings, nil
}

func (c *Client) EnableEventWebhookSignature() (string, error) {
	body, err := json.Marshal(SignedWebhookSettings{Enabled: true})
	if err != nil {
		return "", fmt.Errorf("failed to marshal signed webhook settings: %v", err)
	}

	_, responseBody, err := c.execRequestWithResponse(
		http.MethodPatch,
		c.BaseURL+"/user/webhooks/event/settings/signed",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}

	var response SignedWebhookResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal signed webhook response: %v", err)
	}

	return response.PublicKey, nil
}

type EmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type EmailContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Personalization struct {
	To                  []EmailAddress `json:"to,omitempty"`
	Cc                  []EmailAddress `json:"cc,omitempty"`
	Bcc                 []EmailAddress `json:"bcc,omitempty"`
	DynamicTemplateData map[string]any `json:"dynamic_template_data,omitempty"`
	Subject             string         `json:"subject,omitempty"`
}

type MailSendRequest struct {
	Personalizations []Personalization `json:"personalizations"`
	From             EmailAddress      `json:"from"`
	ReplyTo          *EmailAddress     `json:"reply_to,omitempty"`
	Subject          string            `json:"subject,omitempty"`
	Content          []EmailContent    `json:"content,omitempty"`
	TemplateID       string            `json:"template_id,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
}

type MailSendResult struct {
	MessageID  string `json:"messageId"`
	StatusCode int    `json:"statusCode"`
	Status     string `json:"status"`
}

type ContactInput struct {
	Email        string         `json:"email"`
	FirstName    string         `json:"first_name,omitempty"`
	LastName     string         `json:"last_name,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
}

type UpsertContactsRequest struct {
	Contacts []ContactInput `json:"contacts"`
	ListIDs  []string       `json:"list_ids,omitempty"`
}

type UpsertContactsResponse struct {
	JobID string `json:"job_id"`
}

type UpsertContactResult struct {
	JobID      string `json:"jobId"`
	StatusCode int    `json:"statusCode"`
	Status     string `json:"status"`
}

func (c *Client) SendEmail(request MailSendRequest) (*MailSendResult, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	res, _, err := c.execRequestWithResponse(http.MethodPost, c.BaseURL+"/mail/send", bytes.NewReader(payload))
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			return nil, apiErr
		}
		return nil, err
	}

	return &MailSendResult{
		MessageID:  res.Header.Get("X-Message-Id"),
		StatusCode: res.StatusCode,
		Status:     res.Status,
	}, nil
}

func (c *Client) UpsertContact(request UpsertContactsRequest) (*UpsertContactResult, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	res, responseBody, err := c.execRequestWithResponse(http.MethodPut, c.BaseURL+"/marketing/contacts", bytes.NewReader(payload))
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			return nil, apiErr
		}
		return nil, err
	}

	var response UpsertContactsResponse
	if len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}
	}

	return &UpsertContactResult{
		JobID:      response.JobID,
		StatusCode: res.StatusCode,
		Status:     res.Status,
	}, nil
}

func (c *Client) execRequestWithResponse(method, URL string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return res, responseBody, nil
}
