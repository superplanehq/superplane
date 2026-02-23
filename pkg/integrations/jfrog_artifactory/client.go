package jfrogartifactory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL     string
	AccessToken string
	http        core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	rawURL, err := ctx.GetConfig("url")
	if err != nil {
		return nil, fmt.Errorf("error getting url: %v", err)
	}

	accessToken, err := ctx.GetConfig("accessToken")
	if err != nil {
		return nil, fmt.Errorf("error getting accessToken: %v", err)
	}

	return &Client{
		BaseURL:     strings.TrimRight(string(rawURL), "/"),
		AccessToken: string(accessToken),
		http:        httpCtx,
	}, nil
}

func (c *Client) execRequest(method, requestURL string, body io.Reader, contentType string, allowedStatuses ...int) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("error building request: %v", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res, nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, responseBody, nil
	}

	for _, status := range allowedStatuses {
		if res.StatusCode == status {
			return res, responseBody, nil
		}
	}

	return res, nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
}

// apiURL builds a URL for Artifactory REST API calls.
// The BaseURL is the platform URL (e.g. https://mycompany.jfrog.io),
// so Artifactory-specific paths are prefixed with /artifactory.
func (c *Client) apiURL(path string) string {
	return fmt.Sprintf("%s/artifactory%s", c.BaseURL, path)
}

// platformURL builds a URL for JFrog Platform-level API calls (e.g. webhooks).
func (c *Client) platformURL(path string) string {
	return fmt.Sprintf("%s%s", c.BaseURL, path)
}

// Ping verifies the Artifactory instance is reachable and credentials are valid.
// The ping endpoint returns plain text, so we skip the default JSON accept header.
func (c *Client) Ping() error {
	req, err := http.NewRequest(http.MethodGet, c.apiURL("/api/system/ping"), nil)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(res.Body)
	return fmt.Errorf("request got %d code: %s", res.StatusCode, string(body))
}

// Repository represents a JFrog Artifactory repository.
type Repository struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	PackageType string `json:"packageType"`
}

// ListRepositories returns all repositories from the Artifactory instance.
func (c *Client) ListRepositories() ([]Repository, error) {
	_, responseBody, err := c.execRequest(http.MethodGet, c.apiURL("/api/repositories"), nil, "")
	if err != nil {
		return nil, err
	}

	var repos []Repository
	if err := json.Unmarshal(responseBody, &repos); err != nil {
		return nil, fmt.Errorf("error parsing repositories response: %v", err)
	}

	return repos, nil
}

// ArtifactInfo represents metadata about an artifact in Artifactory.
type ArtifactInfo struct {
	Repo         string            `json:"repo"`
	Path         string            `json:"path"`
	Created      string            `json:"created"`
	CreatedBy    string            `json:"createdBy"`
	LastModified string            `json:"lastModified"`
	ModifiedBy   string            `json:"modifiedBy"`
	LastUpdated  string            `json:"lastUpdated"`
	DownloadURI  string            `json:"downloadUri"`
	MimeType     string            `json:"mimeType"`
	Size         string            `json:"size"`
	Checksums    *ArtifactChecksum `json:"checksums"`
	URI          string            `json:"uri"`
}

// ArtifactChecksum contains the checksums of an artifact.
type ArtifactChecksum struct {
	SHA1   string `json:"sha1"`
	MD5    string `json:"md5"`
	SHA256 string `json:"sha256"`
}

// GetArtifactInfo returns metadata about an artifact.
func (c *Client) GetArtifactInfo(repoKey, path string) (*ArtifactInfo, error) {
	path = strings.TrimPrefix(path, "/")
	requestURL := c.apiURL(fmt.Sprintf("/api/storage/%s/%s", repoKey, path))
	_, responseBody, err := c.execRequest(http.MethodGet, requestURL, nil, "")
	if err != nil {
		return nil, err
	}

	var info ArtifactInfo
	if err := json.Unmarshal(responseBody, &info); err != nil {
		return nil, fmt.Errorf("error parsing artifact info response: %v", err)
	}

	return &info, nil
}

// DeleteArtifact removes an artifact from the specified repository and path.
func (c *Client) DeleteArtifact(repoKey, path string) error {
	path = strings.TrimPrefix(path, "/")
	requestURL := c.apiURL(fmt.Sprintf("/%s/%s", repoKey, path))
	_, _, err := c.execRequest(http.MethodDelete, requestURL, nil, "", http.StatusNoContent)
	return err
}

// JFrogWebhookEventFilter represents the event filter for a JFrog webhook.
type JFrogWebhookEventFilter struct {
	Domain     string                `json:"domain"`
	EventTypes []string              `json:"event_types"`
	Criteria   *JFrogWebhookCriteria `json:"criteria,omitempty"`
}

// JFrogWebhookCriteria represents filtering criteria for a JFrog webhook.
type JFrogWebhookCriteria struct {
	RepoKeys []string `json:"repoKeys,omitempty"`
}

// JFrogWebhookHandlerDef represents a handler definition in a JFrog webhook.
type JFrogWebhookHandlerDef struct {
	HandlerType         string `json:"handler_type"`
	URL                 string `json:"url"`
	Secret              string `json:"secret"`
	UseSecretForSigning bool   `json:"use_secret_for_signing"`
}

// CreateWebhookRequest is the request body for creating a JFrog webhook.
type CreateWebhookRequest struct {
	Key         string                   `json:"key"`
	Description string                   `json:"description"`
	Enabled     bool                     `json:"enabled"`
	EventFilter JFrogWebhookEventFilter  `json:"event_filter"`
	Handlers    []JFrogWebhookHandlerDef `json:"handlers"`
}

// CreateWebhookResponse is the response from creating a JFrog webhook.
type CreateWebhookResponse struct {
	Key string `json:"key"`
}

// CreateWebhook registers a webhook in JFrog Artifactory for artifact deploy events.
// Returns the webhook key used to identify it for later deletion.
func (c *Client) CreateWebhook(webhookURL, secret, repoKey string) (string, error) {
	key := fmt.Sprintf("superplane-%s", uuid.New().String())

	var criteria *JFrogWebhookCriteria
	if repoKey != "" {
		criteria = &JFrogWebhookCriteria{RepoKeys: []string{repoKey}}
	}

	reqBody := CreateWebhookRequest{
		Key:         key,
		Description: "Managed by SuperPlane",
		Enabled:     true,
		EventFilter: JFrogWebhookEventFilter{
			Domain:     "artifact",
			EventTypes: []string{"deployed"},
			Criteria:   criteria,
		},
		Handlers: []JFrogWebhookHandlerDef{
			{
				HandlerType:         "webhook",
				URL:                 webhookURL,
				Secret:              secret,
				UseSecretForSigning: true,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling webhook request: %v", err)
	}

	_, responseBody, err := c.execRequest(http.MethodPost, c.platformURL("/event/api/v1/subscriptions"), bytes.NewReader(bodyBytes), "application/json")
	if err != nil {
		return "", fmt.Errorf("error creating webhook: %v", err)
	}

	var resp CreateWebhookResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return "", fmt.Errorf("error parsing webhook response: %v", err)
	}

	if resp.Key == "" {
		return key, nil
	}

	return resp.Key, nil
}

// DeleteWebhook removes a webhook from JFrog Artifactory by its key.
// A 404 response is treated as success since the webhook is already gone.
func (c *Client) DeleteWebhook(key string) error {
	_, _, err := c.execRequest(http.MethodDelete, c.platformURL(fmt.Sprintf("/event/api/v1/subscriptions/%s", key)), nil, "", http.StatusNoContent, http.StatusNotFound)
	return err
}
