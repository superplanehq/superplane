package dokploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const apiPathPrefix = "/api"

type Client struct {
	Token   string
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
		return fmt.Sprintf("Dokploy API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("Dokploy API error %d: %s", e.StatusCode, e.Body)
}

// Application is an application as returned nested inside project.all.
// Dokploy exposes no flat "list applications" endpoint, so these are only
// ever read through a project's environments.
type Application struct {
	ApplicationID string `json:"applicationId"`
	Name          string `json:"name"`
	Status        string `json:"applicationStatus"`
}

// Environment groups applications inside a project. Dokploy nests
// project -> environments -> applications.
type Environment struct {
	EnvironmentID string        `json:"environmentId"`
	Name          string        `json:"name"`
	IsDefault     bool          `json:"isDefault"`
	Applications  []Application `json:"applications"`
}

type Project struct {
	ProjectID    string        `json:"projectId"`
	Name         string        `json:"name"`
	Environments []Environment `json:"environments"`
}

// ApplicationSummary is a flattened application carrying the project and
// environment it belongs to. Application names are only unique within an
// environment, so the surrounding context is needed to identify one.
type ApplicationSummary struct {
	ApplicationID   string
	Name            string
	Status          string
	ProjectID       string
	ProjectName     string
	EnvironmentID   string
	EnvironmentName string
}

// DeployRequest is the body accepted by application.deploy. Title and
// description label the deployment in Dokploy's log; when title is empty
// Dokploy records "Manual deployment".
type DeployRequest struct {
	ApplicationID string `json:"applicationId"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	baseURL, err := readBaseURL(integration)
	if err != nil {
		return nil, err
	}

	token, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("apiToken is required: %w", err)
	}

	return &Client{
		Token:   string(token),
		BaseURL: baseURL,
		http:    httpCtx,
	}, nil
}

// readBaseURL fetches and validates the Dokploy base URL from the integration
// configuration. It strips any trailing slash and a trailing API path prefix
// so that a user pasting either "https://dokploy.example.com" or
// "https://dokploy.example.com/api" produces the same result.
func readBaseURL(integration core.IntegrationContext) (string, error) {
	raw, err := integration.GetConfig("baseUrl")
	if err != nil {
		return "", fmt.Errorf("baseUrl is required: %w", err)
	}

	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", fmt.Errorf("baseUrl is required")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid baseUrl: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid baseUrl: must include scheme and host (e.g. https://dokploy.example.com)")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid baseUrl: unsupported scheme %q (expected http or https)", parsed.Scheme)
	}

	value = strings.TrimRight(value, "/")
	value = strings.TrimSuffix(value, apiPathPrefix)
	return strings.TrimRight(value, "/"), nil
}

// do issues a request against the Dokploy API. Dokploy generates its HTTP API
// from tRPC routers, so paths are dot-named procedures ("application.deploy"),
// queries are GET and mutations are POST with a JSON body.
func (c *Client) do(method, procedure string, query url.Values, body any) (*http.Response, error) {
	target := c.BaseURL + apiPathPrefix + "/" + procedure
	if len(query) > 0 {
		target = target + "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode %s request: %w", procedure, err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, target, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.http.Do(req)
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(body)}

	// tRPC surfaces failures either as a flat {"message": ...} or nested
	// under {"error": {"message": ...}} depending on the endpoint.
	var errPayload struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errPayload) == nil {
		switch {
		case errPayload.Message != "":
			apiErr.Message = errPayload.Message
		case errPayload.Error.Message != "":
			apiErr.Message = errPayload.Error.Message
		}
	}
	return apiErr
}

// Verify confirms the credentials are usable. Dokploy has no dedicated
// version or whoami endpoint, so the cheapest authenticated read is used.
func (c *Client) Verify() error {
	resp, err := c.do(http.MethodGet, "project.all", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// ListApplications flattens every application across all projects and
// environments the API token can reach.
func (c *Client) ListApplications() ([]ApplicationSummary, error) {
	resp, err := c.do(http.MethodGet, "project.all", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("decode list projects response: %w", err)
	}

	summaries := []ApplicationSummary{}
	for _, project := range projects {
		for _, environment := range project.Environments {
			for _, application := range environment.Applications {
				if strings.TrimSpace(application.ApplicationID) == "" {
					continue
				}
				summaries = append(summaries, ApplicationSummary{
					ApplicationID:   application.ApplicationID,
					Name:            application.Name,
					Status:          application.Status,
					ProjectID:       project.ProjectID,
					ProjectName:     project.Name,
					EnvironmentID:   environment.EnvironmentID,
					EnvironmentName: environment.Name,
				})
			}
		}
	}
	return summaries, nil
}

// Deploy queues a deployment for the given application.
//
// application.deploy returns no useful body: on Dokploy Cloud it responds with
// `true`, and on self-hosted instances it enqueues the job and returns nothing
// at all. Success is therefore determined by status code alone.
func (c *Client) Deploy(request DeployRequest) error {
	resp, err := c.do(http.MethodPost, "application.deploy", nil, request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}
