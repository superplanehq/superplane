package terraformcloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultHostname = "app.terraform.io"

type Client struct {
	APIToken string
	BaseURL  string
	http     core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	hostname := defaultHostname
	hostnameBytes, err := ctx.GetConfig("hostname")
	if err == nil && len(hostnameBytes) > 0 {
		hostname = string(hostnameBytes)
	}

	return &Client{
		APIToken: string(apiToken),
		BaseURL:  fmt.Sprintf("https://%s/api/v2", hostname),
		http:     http,
	}, nil
}

func NewClientFromConfig(http core.HTTPContext, apiToken, hostname string) *Client {
	if hostname == "" {
		hostname = defaultHostname
	}

	return &Client{
		APIToken: apiToken,
		BaseURL:  fmt.Sprintf("https://%s/api/v2", hostname),
		http:     http,
	}
}

func (c *Client) execRequest(method, requestURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIToken)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type AccountDetails struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Username string `json:"username"`
			Email    string `json:"email"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetCurrentUser() (*AccountDetails, error) {
	reqURL := fmt.Sprintf("%s/account/details", c.BaseURL)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var account AccountDetails
	err = json.Unmarshal(responseBody, &account)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &account, nil
}

type Workspace struct {
	ID         string `json:"id"`
	Attributes struct {
		Name string `json:"name"`
	} `json:"attributes"`
}

type WorkspacesResponse struct {
	Data []Workspace `json:"data"`
}

func (c *Client) ListWorkspaces(organization string) ([]Workspace, error) {
	reqURL := fmt.Sprintf("%s/organizations/%s/workspaces?page[size]=100", c.BaseURL, organization)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response WorkspacesResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Data, nil
}

func (c *Client) GetWorkspace(workspaceID string) (*Workspace, error) {
	reqURL := fmt.Sprintf("%s/workspaces/%s", c.BaseURL, workspaceID)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data Workspace `json:"data"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response.Data, nil
}

type Run struct {
	ID         string `json:"id"`
	Attributes struct {
		Status    string `json:"status"`
		Message   string `json:"message"`
		CreatedAt string `json:"created-at"`
		Source    string `json:"source"`
	} `json:"attributes"`
	Relationships struct {
		Workspace struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"workspace"`
	} `json:"relationships"`
}

func (c *Client) GetRun(runID string) (*Run, error) {
	reqURL := fmt.Sprintf("%s/runs/%s", c.BaseURL, runID)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data Run `json:"data"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response.Data, nil
}

type CreateRunRequest struct {
	Data struct {
		Attributes struct {
			Message   string `json:"message,omitempty"`
			AutoApply bool   `json:"auto-apply"`
		} `json:"attributes"`
		Type          string `json:"type"`
		Relationships struct {
			Workspace struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"workspace"`
		} `json:"relationships"`
	} `json:"data"`
}

type CreateRunResponse struct {
	Data Run `json:"data"`
}

func (c *Client) CreateRun(workspaceID, message string, autoApply bool) (*Run, error) {
	runReq := CreateRunRequest{}
	runReq.Data.Type = "runs"
	runReq.Data.Attributes.Message = message
	runReq.Data.Attributes.AutoApply = autoApply
	runReq.Data.Relationships.Workspace.Data.Type = "workspaces"
	runReq.Data.Relationships.Workspace.Data.ID = workspaceID

	body, err := json.Marshal(runReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	reqURL := fmt.Sprintf("%s/runs", c.BaseURL)
	responseBody, err := c.execRequest("POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateRunResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response.Data, nil
}

type NotificationConfiguration struct {
	ID         string `json:"id"`
	Attributes struct {
		Name            string   `json:"name"`
		URL             string   `json:"url"`
		DestinationType string   `json:"destination-type"`
		Enabled         bool     `json:"enabled"`
		Triggers        []string `json:"triggers"`
	} `json:"attributes"`
}

type CreateNotificationRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			DestinationType string   `json:"destination-type"`
			Enabled         bool     `json:"enabled"`
			Name            string   `json:"name"`
			Token           string   `json:"token"`
			URL             string   `json:"url"`
			Triggers        []string `json:"triggers"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) CreateNotificationConfiguration(workspaceID, name, webhookURL, token string, triggers []string) (*NotificationConfiguration, error) {
	notifReq := CreateNotificationRequest{}
	notifReq.Data.Type = "notification-configurations"
	notifReq.Data.Attributes.DestinationType = "generic"
	notifReq.Data.Attributes.Enabled = true
	notifReq.Data.Attributes.Name = name
	notifReq.Data.Attributes.Token = token
	notifReq.Data.Attributes.URL = webhookURL
	notifReq.Data.Attributes.Triggers = triggers

	body, err := json.Marshal(notifReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	reqURL := fmt.Sprintf("%s/workspaces/%s/notification-configurations", c.BaseURL, workspaceID)
	responseBody, err := c.execRequest("POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data NotificationConfiguration `json:"data"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response.Data, nil
}

func (c *Client) DeleteNotificationConfiguration(notificationID string) error {
	reqURL := fmt.Sprintf("%s/notification-configurations/%s", c.BaseURL, notificationID)
	_, err := c.execRequest("DELETE", reqURL, nil)
	if err != nil {
		return fmt.Errorf("error deleting notification configuration: %v", err)
	}

	return nil
}

func (c *Client) ListNotificationConfigurations(workspaceID string) ([]NotificationConfiguration, error) {
	reqURL := fmt.Sprintf("%s/workspaces/%s/notification-configurations", c.BaseURL, workspaceID)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []NotificationConfiguration `json:"data"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Data, nil
}

func IsTerminalRunStatus(status string) bool {
	switch status {
	case "applied", "planned_and_finished", "discarded", "errored", "canceled", "force_canceled":
		return true
	}
	return false
}

func IsSuccessRunStatus(status string) bool {
	return status == "applied" || status == "planned_and_finished"
}
