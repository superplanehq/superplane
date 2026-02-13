package hetzner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultHetznerBaseURL = "https://api.hetzner.cloud/v1"

type Client struct {
	Token  string
	BaseURL string
	http    core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Hetzner API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("Hetzner API error %d: %s", e.StatusCode, e.Body)
}

type createServerRequest struct {
	Name        string   `json:"name"`
	ServerType  string   `json:"server_type"`
	Image       string   `json:"image"`
	Location    string   `json:"location,omitempty"`
	SSHKeys     []string `json:"ssh_keys,omitempty"`
	UserData    string   `json:"user_data,omitempty"`
	StartAfterCreate *bool `json:"start_after_create,omitempty"`
}

type createServerResponse struct {
	Server *ServerResponse `json:"server"`
	Action *ActionResponse `json:"action"`
}

type ServerResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Created  string `json:"created"`
	PublicNet struct {
		IPv4 struct {
			IP string `json:"ip"`
		} `json:"ipv4"`
	} `json:"public_net"`
}

type ActionResponse struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	Command   string `json:"command"`
	Progress  int    `json:"progress"`
	Started   string `json:"started"`
	Finished  string `json:"finished"`
	Error     *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type getActionResponse struct {
	Action ActionResponse `json:"action"`
}

const (
	ActionStatusRunning = "running"
	ActionStatusSuccess = "success"
	ActionStatusError   = "error"
)

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	token, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("apiToken is required: %w", err)
	}
	return &Client{
		Token:   string(token),
		BaseURL: defaultHetznerBaseURL,
		http:    httpCtx,
	}, nil
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	var errPayload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errPayload) == nil && errPayload.Error.Message != "" {
		apiErr.Message = errPayload.Error.Message
	}
	return apiErr
}

func (c *Client) CreateServer(name, serverType, image, location string, sshKeys []string, userData string) (*ServerResponse, *ActionResponse, error) {
	req := createServerRequest{
		Name:       name,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		SSHKeys:    sshKeys,
		UserData:   userData,
	}
	startAfter := true
	req.StartAfterCreate = &startAfter

	resp, err := c.do("POST", "/servers", req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, nil, c.parseError(resp)
	}

	var out createServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil, fmt.Errorf("decode create server response: %w", err)
	}
	if out.Server == nil || out.Action == nil {
		return nil, nil, fmt.Errorf("create server response missing server or action")
	}
	return out.Server, out.Action, nil
}

func (c *Client) GetAction(actionID int) (*ActionResponse, error) {
	resp, err := c.do("GET", "/actions/"+strconv.Itoa(actionID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out getActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode get action response: %w", err)
	}
	return &out.Action, nil
}

func (c *Client) DeleteServer(serverID int) (*ActionResponse, error) {
	resp, err := c.do("DELETE", "/servers/"+strconv.Itoa(serverID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Action ActionResponse `json:"action"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode delete server response: %w", err)
	}
	return &out.Action, nil
}

func (c *Client) ListServers() ([]ServerResponse, error) {
	resp, err := c.do("GET", "/servers?per_page=50", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var out struct {
		Servers []ServerResponse `json:"servers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode list servers response: %w", err)
	}
	return out.Servers, nil
}

func (c *Client) Verify() error {
	resp, err := c.do("GET", "/servers?per_page=1", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}
