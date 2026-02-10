package dockerhub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultBaseURL  = "https://hub.docker.com"
	defaultPageSize = 100
)

type Client struct {
	AccessToken string
	BaseURL     string
	http        core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("no integration context")
	}

	accessToken, err := findSecret(integration, accessTokenSecretName)
	if err != nil {
		return nil, fmt.Errorf("access token not configured: %w", err)
	}

	token := strings.TrimSpace(accessToken)
	if token == "" {
		return nil, fmt.Errorf("access token is required")
	}

	return &Client{
		AccessToken: token,
		BaseURL:     defaultBaseURL,
		http:        httpClient,
	}, nil
}

func findSecret(ctx core.IntegrationContext, secretName string) (string, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			return string(secret.Value), nil
		}
	}

	return "", fmt.Errorf("secret %s not found", secretName)
}

func (c *Client) doRequest(method, URL string, body io.Reader) (*http.Response, []byte, error) {
	finalURL := URL
	if !strings.HasPrefix(URL, "http") {
		finalURL = c.BaseURL + URL
	}

	req, err := http.NewRequest(method, finalURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

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
		return nil, nil, fmt.Errorf("request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return res, responseBody, nil
}

type Repository struct {
	Name        string `json:"name" mapstructure:"name"`
	Namespace   string `json:"namespace" mapstructure:"namespace"`
	Description string `json:"description" mapstructure:"description"`
	IsPrivate   bool   `json:"is_private" mapstructure:"is_private"`
	StarCount   int    `json:"star_count" mapstructure:"star_count"`
	PullCount   int    `json:"pull_count" mapstructure:"pull_count"`
	Status      string `json:"status_description" mapstructure:"status_description"`
}

type ListRepositoriesResponse struct {
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

		var response ListRepositoriesResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("failed to parse repositories response: %w", err)
		}

		repositories = append(repositories, response.Results...)
		path = response.Next
	}

	return repositories, nil
}

func (c *Client) GetRepository(namespace, repository string) (*Repository, error) {
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
	ID                  int64    `json:"id"`
	Name                string   `json:"name"`
	FullSize            int64    `json:"full_size"`
	LastUpdated         string   `json:"last_updated"`
	LastUpdater         int64    `json:"last_updater"`
	LastUpdaterUsername string   `json:"last_updater_username"`
	Status              string   `json:"status"`
	TagLastPulled       string   `json:"tag_last_pulled"`
	TagLastPushed       string   `json:"tag_last_pushed"`
	Repository          int64    `json:"repository"`
	Images              ImageSet `json:"images"`
}

func (c *Client) GetRepositoryTag(namespace, repository, tag string) (*Tag, error) {
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
