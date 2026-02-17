package honeycomb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	BaseURLUS = "https://api.honeycomb.io"
)

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKeyAny, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("api key is required")
	}
	apiKey := strings.TrimSpace(string(apiKeyAny))
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	siteAny, err := ctx.GetConfig("site")
	if err != nil {
		siteAny = []byte("api.honeycomb.io")
	}
	site := strings.TrimSpace(string(siteAny))
	if site == "" {
		site = strings.TrimPrefix(BaseURLUS, "https://")
	}

	baseURL := site
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	return &Client{
		APIKey:  apiKey,
		BaseURL: baseURL,
		http:    httpCtx,
	}, nil

}

func (c *Client) Validate() error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/1/auth", nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Honeycomb-Team", c.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate api key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid api key")
	}

	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("api key is valid, but does not have permission for this account/team")
	}

	return fmt.Errorf("honeycomb authentication failed (http %d)", resp.StatusCode)
}

func (c *Client) CreateEvent(datasetSlug string, fields map[string]any) error {
	datasetSlug = strings.TrimSpace(datasetSlug)
	if datasetSlug == "" {
		return fmt.Errorf("dataset is required")
	}

	u, _ := url.Parse(c.BaseURL)
	u.Path = fmt.Sprintf("/1/events/%s", url.PathEscape(datasetSlug))

	body, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Honeycomb-Team", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Set event timestamp header only if "time" field is not provided in the fields map.
	// Honeycomb uses this header as the authoritative event timestamp, so we only set it
	// when the user hasn't provided their own timestamp. Use RFC3339Nano in UTC to match
	// Honeycomb's expected format (same as libhoney-go).
	if _, hasTimeField := fields["time"]; !hasTimeField {
		req.Header.Set("X-Honeycomb-Event-Time", time.Now().UTC().Format(time.RFC3339Nano))
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("honeycomb create event failed (status %d): %s", resp.StatusCode, string(b))
}
