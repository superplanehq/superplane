package argocd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const versionPath = "/api/version"

type Client struct {
	serverURL string
	authToken string
	http      core.HTTPContext
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("Argo CD API returned status %d", e.StatusCode)
	}

	return fmt.Sprintf("Argo CD API returned status %d: %s", e.StatusCode, e.Message)
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	serverURL, err := readServerURL(integration)
	if err != nil {
		return nil, err
	}

	authToken, err := integration.GetConfig("authToken")
	if err != nil {
		return nil, fmt.Errorf("authToken is required: %w", err)
	}

	if strings.TrimSpace(string(authToken)) == "" {
		return nil, fmt.Errorf("authToken is required")
	}

	return &Client{
		serverURL: serverURL,
		authToken: string(authToken),
		http:      httpCtx,
	}, nil
}

func readServerURL(integration core.IntegrationContext) (string, error) {
	value, err := integration.GetConfig("serverUrl")
	if err != nil {
		return "", fmt.Errorf("serverUrl is required: %w", err)
	}

	serverURL := strings.TrimSpace(string(value))
	if serverURL == "" {
		return "", fmt.Errorf("serverUrl is required")
	}

	parsed, err := url.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("invalid serverUrl: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid serverUrl: use an HTTP or HTTPS URL")
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("invalid serverUrl: include a host")
	}

	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid serverUrl: do not include credentials, a query, or a fragment")
	}

	parsed.Path = strings.TrimSuffix(strings.TrimRight(parsed.Path, "/"), "/api/v1")
	parsed.Path = strings.TrimRight(parsed.Path, "/")

	return strings.TrimRight(parsed.String(), "/"), nil
}

func (c *Client) Verify() error {
	response, err := c.do(http.MethodGet, versionPath, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return parseAPIError(response)
	}

	return nil
}

func (c *Client) GetApplication(project, name, appNamespace string) (Application, error) {
	query := url.Values{"project": []string{project}}
	if appNamespace != "" {
		query.Set("appNamespace", appNamespace)
	}

	response, err := c.do(http.MethodGet, "/api/v1/applications/"+url.PathEscape(name), query)
	if err != nil {
		return Application{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Application{}, parseAPIError(response)
	}

	application := Application{}
	if err := json.NewDecoder(response.Body).Decode(&application); err != nil {
		return Application{}, fmt.Errorf("decode Argo CD application response: %w", err)
	}

	return application, nil
}

func (c *Client) do(method, path string, query url.Values) (*http.Response, error) {
	target := c.serverURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	request, err := http.NewRequest(method, target, nil)
	if err != nil {
		return nil, fmt.Errorf("build Argo CD request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.authToken)

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send Argo CD request: %w", err)
	}

	return response, nil
}

func parseAPIError(response *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return fmt.Errorf("read Argo CD error response: %w", err)
	}

	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if payload.Error != "" {
			return &APIError{StatusCode: response.StatusCode, Message: payload.Error}
		}
		if payload.Message != "" {
			return &APIError{StatusCode: response.StatusCode, Message: payload.Message}
		}
	}

	return &APIError{
		StatusCode: response.StatusCode,
		Message:    strings.TrimSpace(string(body)),
	}
}
