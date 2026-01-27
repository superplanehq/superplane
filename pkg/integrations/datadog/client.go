package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	APIKey  string
	AppKey  string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error getting apiKey: %v", err)
	}

	appKey, err := ctx.GetConfig("appKey")
	if err != nil {
		return nil, fmt.Errorf("error getting appKey: %v", err)
	}

	site, err := ctx.GetConfig("site")
	if err != nil {
		return nil, fmt.Errorf("error getting site: %v", err)
	}

	return &Client{
		APIKey:  string(apiKey),
		AppKey:  string(appKey),
		BaseURL: fmt.Sprintf("https://api.%s", site),
		http:    http,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", c.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", c.AppKey)

	res, err := c.http.Do(req)
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

// ValidateCredentials verifies that the API and Application keys are valid
// by calling the Datadog validate endpoint.
func (c *Client) ValidateCredentials() error {
	url := fmt.Sprintf("%s/api/v1/validate", c.BaseURL)
	_, err := c.execRequest(http.MethodGet, url, nil)
	return err
}

// CreateEventRequest represents the request payload for creating a Datadog event.
type CreateEventRequest struct {
	Title     string   `json:"title"`
	Text      string   `json:"text"`
	AlertType string   `json:"alert_type,omitempty"`
	Priority  string   `json:"priority,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// Event represents a Datadog event response.
type Event struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	Text         string   `json:"text"`
	DateHappened int64    `json:"date_happened"`
	AlertType    string   `json:"alert_type"`
	Priority     string   `json:"priority"`
	Tags         []string `json:"tags"`
	URL          string   `json:"url"`
}

// CreateEventResponse represents the response from creating an event.
type CreateEventResponse struct {
	Event  Event  `json:"event"`
	Status string `json:"status"`
}

// CreateEvent creates a new event in Datadog.
func (c *Client) CreateEvent(req CreateEventRequest) (*Event, error) {
	url := fmt.Sprintf("%s/api/v1/events", c.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateEventResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Event, nil
}
