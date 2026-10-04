package jenkins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL  string
	Username string
	APIToken string
	http     core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	baseURL, err := ctx.GetConfig("baseUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting baseUrl: %w", err)
	}

	username, err := ctx.GetConfig("username")
	if err != nil {
		return nil, fmt.Errorf("error getting username: %w", err)
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %w", err)
	}

	return &Client{
		BaseURL:  strings.TrimRight(string(baseURL), "/"),
		Username: string(username),
		APIToken: string(apiToken),
		http:     http,
	}, nil
}

// newRequest builds a Basic-authenticated request against the configured Base URL.
// Callers that need extra headers (e.g. a CSRF crumb) should set them before calling do().
func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, path), body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.APIToken)

	return req, nil
}

func (c *Client) do(req *http.Request) (int, []byte, http.Header, error) {
	res, err := c.http.Do(req)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("error reading body: %w", err)
	}

	return res.StatusCode, responseBody, res.Header, nil
}

func (c *Client) execRequest(method, path string, body io.Reader) (int, []byte, error) {
	req, err := c.newRequest(method, path, body)
	if err != nil {
		return 0, nil, err
	}

	statusCode, responseBody, _, err := c.do(req)
	return statusCode, responseBody, err
}

// Verify checks that the configured Base URL and credentials are valid.
func (c *Client) Verify() error {
	statusCode, body, err := c.execRequest(http.MethodGet, "/api/json", nil)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return statusError(statusCode, body)
	}

	return nil
}

const maxErrorBodyLen = 200

// statusError builds a user-facing error for a non-2xx Jenkins response.
// Jenkins renders most errors as Jetty's default HTML error page, which is
// noisy and unhelpful to show as-is, so known status codes get a clean
// message and anything else has its body truncated/sanitized.
func statusError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (401): check your Jenkins username and API token")
	case http.StatusForbidden:
		return fmt.Errorf("access denied (403): check your Jenkins username and API token permissions")
	case http.StatusNotFound:
		return fmt.Errorf("not found (404): check your Jenkins Base URL")
	default:
		return fmt.Errorf("request got %d: %s", statusCode, sanitizeErrorBody(body))
	}
}

type CrumbResponse struct {
	CrumbRequestField string `json:"crumbRequestField"`
	Crumb             string `json:"crumb"`
}

// getCrumb fetches a CSRF crumb to send on POST requests. Returns a nil
// crumb (no error) when the crumb issuer is disabled on the Jenkins server (404).
func (c *Client) getCrumb() (*CrumbResponse, error) {
	statusCode, body, err := c.execRequest(http.MethodGet, "/crumbIssuer/api/json", nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, nil
	}

	if statusCode != http.StatusOK {
		return nil, statusError(statusCode, body)
	}

	var crumb CrumbResponse
	if err := json.Unmarshal(body, &crumb); err != nil {
		return nil, fmt.Errorf("error unmarshaling crumb response: %w", err)
	}

	return &crumb, nil
}

type TriggerBuildResult struct {
	QueueURL string
}

// TriggerBuild starts a build for the given job. When params is non-empty,
// it POSTs to buildWithParameters instead of build.
func (c *Client) TriggerBuild(job string, params map[string]string) (*TriggerBuildResult, error) {
	path := fmt.Sprintf("/job/%s/build", url.PathEscape(job))

	var body io.Reader
	if len(params) > 0 {
		path = fmt.Sprintf("/job/%s/buildWithParameters", url.PathEscape(job))
		form := url.Values{}
		for name, value := range params {
			form.Set(name, value)
		}
		body = strings.NewReader(form.Encode())
	}

	req, err := c.newRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	crumb, err := c.getCrumb()
	if err != nil {
		return nil, fmt.Errorf("error getting CSRF crumb: %w", err)
	}
	if crumb != nil {
		req.Header.Set(crumb.CrumbRequestField, crumb.Crumb)
	}

	statusCode, responseBody, headers, err := c.do(req)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("job %q not found", job)
	}

	if statusCode != http.StatusCreated {
		return nil, statusError(statusCode, responseBody)
	}

	return &TriggerBuildResult{QueueURL: headers.Get("Location")}, nil
}

type BuildResponse struct {
	Building bool    `json:"building"`
	Result   *string `json:"result"`
	Number   int     `json:"number"`
	URL      string  `json:"url"`
	Duration int64   `json:"duration"`
}

// GetBuild fetches the status of a specific build for a job.
func (c *Client) GetBuild(job string, number int) (*BuildResponse, error) {
	path := fmt.Sprintf("/job/%s/%d/api/json", url.PathEscape(job), number)

	statusCode, body, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("build %d not found for job %q", number, job)
	}

	if statusCode != http.StatusOK {
		return nil, statusError(statusCode, body)
	}

	var build BuildResponse
	if err := json.Unmarshal(body, &build); err != nil {
		return nil, fmt.Errorf("error unmarshaling build response: %w", err)
	}

	return &build, nil
}

func sanitizeErrorBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "(empty response)"
	}

	if strings.HasPrefix(text, "<") {
		return "(non-JSON response)"
	}

	if len(text) > maxErrorBodyLen {
		return text[:maxErrorBodyLen] + "..."
	}

	return text
}
