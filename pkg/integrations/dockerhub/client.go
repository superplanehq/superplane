package dockerhub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultBaseURL    = "https://hub.docker.com"
	defaultPageSize   = 100
	tokenTTLLeeway    = 9 * time.Minute
	authTokenEndpoint = "/v2/auth/token"
)

type Client struct {
	Identifier string
	Secret     string
	BaseURL    string
	http       core.HTTPContext

	token       string
	tokenExpiry time.Time
	tokenMu     sync.Mutex
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

func NewClient(httpClient core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("no integration context")
	}

	username, err := integration.GetConfig("username")
	if err != nil {
		return nil, err
	}

	accessToken, err := integration.GetConfig("accessToken")
	if err != nil {
		return nil, err
	}

	identifier := strings.TrimSpace(string(username))
	if identifier == "" {
		return nil, fmt.Errorf("username is required")
	}

	secret := strings.TrimSpace(string(accessToken))
	if secret == "" {
		return nil, fmt.Errorf("accessToken is required")
	}

	return &Client{
		Identifier: identifier,
		Secret:     secret,
		BaseURL:    defaultBaseURL,
		http:       httpClient,
	}, nil
}

type accessTokenRequest struct {
	Identifier string `json:"identifier"`
	Secret     string `json:"secret"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (c *Client) ensureAccessToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	token, err := c.createAccessToken()
	if err != nil {
		return "", err
	}

	c.token = token
	c.tokenExpiry = time.Now().Add(tokenTTLLeeway)
	return token, nil
}

func (c *Client) createAccessToken() (string, error) {
	payload, err := json.Marshal(accessTokenRequest{
		Identifier: c.Identifier,
		Secret:     c.Secret,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal access token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+authTokenEndpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create access token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("access token request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read access token response: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", &APIError{StatusCode: res.StatusCode, Body: string(body)}
	}

	var response accessTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse access token response: %w", err)
	}

	if response.AccessToken == "" {
		return "", fmt.Errorf("access token response was empty")
	}

	return response.AccessToken, nil
}

func (c *Client) doRequest(method, URL string, body io.Reader) (*http.Response, []byte, error) {
	token, err := c.ensureAccessToken()
	if err != nil {
		return nil, nil, err
	}

	finalURL := URL
	if !strings.HasPrefix(URL, "http") {
		finalURL = c.BaseURL + URL
	}

	req, err := http.NewRequest(method, finalURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return res, responseBody, nil
}

type Repository struct {
	Name        string `json:"name" mapstructure:"name"`
	Namespace   string `json:"namespace" mapstructure:"namespace"`
	Description string `json:"description" mapstructure:"description"`
	RepoURL     string `json:"repo_url" mapstructure:"repo_url"`
	IsPrivate   bool   `json:"is_private" mapstructure:"is_private"`
	StarCount   int    `json:"star_count" mapstructure:"star_count"`
	PullCount   int    `json:"pull_count" mapstructure:"pull_count"`
	Status      string `json:"status_description" mapstructure:"status_description"`
}

type listRepositoriesResponse struct {
	Next    string       `json:"next"`
	Results []Repository `json:"results"`
}

func (c *Client) ValidateCredentials(namespace string) error {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	_, err := c.ListRepositories(namespace)
	return err
}

func (c *Client) ListRepositories(namespace string) ([]Repository, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	path := fmt.Sprintf("/v2/namespaces/%s/repositories?page_size=%d", namespace, defaultPageSize)
	repositories := []Repository{}

	for path != "" {
		_, responseBody, err := c.doRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var response listRepositoriesResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("failed to parse repositories response: %w", err)
		}

		repositories = append(repositories, response.Results...)
		path = response.Next
	}

	return repositories, nil
}

func (c *Client) GetRepository(namespace, repository string) (*Repository, error) {
	namespace = strings.TrimSpace(namespace)
	repository = strings.TrimSpace(repository)
	if namespace == "" || repository == "" {
		return nil, fmt.Errorf("namespace and repository are required")
	}

	path := fmt.Sprintf("/v2/namespaces/%s/repositories/%s", namespace, repository)
	_, responseBody, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := json.Unmarshal(responseBody, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse repository response: %w", err)
	}

	return &repo, nil
}

type Image struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Digest       string `json:"digest"`
	Size         int64  `json:"size"`
	Status       string `json:"status"`
	LastPulled   string `json:"last_pulled"`
	LastPushed   string `json:"last_pushed"`
}

type ImageSet []Image

func (i *ImageSet) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	if data[0] == '[' {
		var images []Image
		if err := json.Unmarshal(data, &images); err != nil {
			return err
		}
		*i = images
		return nil
	}

	var image Image
	if err := json.Unmarshal(data, &image); err != nil {
		return err
	}
	*i = []Image{image}
	return nil
}

type Tag struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	FullSize           int64    `json:"full_size"`
	LastUpdated        string   `json:"last_updated"`
	LastUpdater        int64    `json:"last_updater"`
	LastUpdaterUsername string  `json:"last_updater_username"`
	Status             string   `json:"status"`
	TagLastPulled       string   `json:"tag_last_pulled"`
	TagLastPushed       string   `json:"tag_last_pushed"`
	Repository          int64    `json:"repository"`
	V2                  string   `json:"v2"`
	Images              ImageSet `json:"images"`
}

func (c *Client) GetRepositoryTag(namespace, repository, tag string) (*Tag, error) {
	namespace = strings.TrimSpace(namespace)
	repository = strings.TrimSpace(repository)
	tag = strings.TrimSpace(tag)
	if namespace == "" || repository == "" || tag == "" {
		return nil, fmt.Errorf("namespace, repository, and tag are required")
	}

	path := fmt.Sprintf("/v2/namespaces/%s/repositories/%s/tags/%s", namespace, repository, tag)
	_, responseBody, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result Tag
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tag response: %w", err)
	}

	return &result, nil
}
