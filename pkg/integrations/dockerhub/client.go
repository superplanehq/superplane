package dockerhub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	dockerHubAPIURL  = "https://hub.docker.com/v2"
	dockerHubAuthURL = "https://hub.docker.com/v2/users/login"
)

// Client represents a Docker Hub API client
type Client struct {
	Username    string
	AccessToken string
	http        core.HTTPContext
	jwtToken    string
}

// NewClient creates a new Docker Hub client
func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	username, err := ctx.GetConfig("username")
	if err != nil {
		return nil, fmt.Errorf("error getting username: %v", err)
	}

	accessToken, err := ctx.GetConfig("accessToken")
	if err != nil {
		return nil, fmt.Errorf("error getting accessToken: %v", err)
	}

	return &Client{
		Username:    string(username),
		AccessToken: string(accessToken),
		http:        httpCtx,
	}, nil
}

// LoginResponse represents the response from Docker Hub login
type LoginResponse struct {
	Token string `json:"token"`
}

// authenticate gets a JWT token using username and access token
func (c *Client) authenticate() error {
	if c.jwtToken != "" {
		return nil
	}

	loginPayload := map[string]string{
		"username": c.Username,
		"password": c.AccessToken,
	}

	body, err := json.Marshal(loginPayload)
	if err != nil {
		return fmt.Errorf("error marshaling login payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, dockerHubAuthURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error creating login request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error executing login request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading login response: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d: %s", res.StatusCode, string(responseBody))
	}

	var loginResponse LoginResponse
	err = json.Unmarshal(responseBody, &loginResponse)
	if err != nil {
		return fmt.Errorf("error parsing login response: %v", err)
	}

	c.jwtToken = loginResponse.Token
	return nil
}

// execRequest executes an authenticated request to Docker Hub API
func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))

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

// ValidateCredentials verifies that the credentials are valid
func (c *Client) ValidateCredentials() error {
	return c.authenticate()
}

// Tag represents a Docker Hub image tag
type Tag struct {
	Creator             int64       `json:"creator"`
	ID                  int64       `json:"id"`
	LastUpdated         string      `json:"last_updated"`
	LastUpdater         int64       `json:"last_updater"`
	LastUpdaterUsername string      `json:"last_updater_username"`
	Name                string      `json:"name"`
	Repository          int64       `json:"repository"`
	FullSize            int64       `json:"full_size"`
	V2                  bool        `json:"v2"`
	TagStatus           string      `json:"tag_status"`
	TagLastPulled       string      `json:"tag_last_pulled,omitempty"`
	TagLastPushed       string      `json:"tag_last_pushed,omitempty"`
	MediaType           string      `json:"media_type,omitempty"`
	ContentType         string      `json:"content_type,omitempty"`
	Digest              string      `json:"digest,omitempty"`
	Images              []ImageInfo `json:"images,omitempty"`
}

// ImageInfo represents information about a Docker image
type ImageInfo struct {
	Architecture string `json:"architecture"`
	Features     string `json:"features"`
	Variant      string `json:"variant,omitempty"`
	Digest       string `json:"digest"`
	OS           string `json:"os"`
	OSFeatures   string `json:"os_features,omitempty"`
	OSVersion    string `json:"os_version,omitempty"`
	Size         int64  `json:"size"`
	Status       string `json:"status"`
	LastPulled   string `json:"last_pulled,omitempty"`
	LastPushed   string `json:"last_pushed,omitempty"`
}

// ListTagsResponse represents the response from listing tags
type ListTagsResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Results  []Tag  `json:"results"`
}

// ListTagsRequest represents the request parameters for listing tags
type ListTagsRequest struct {
	Repository string
	PageSize   int
	NameFilter string
}

// ListTags lists tags for a Docker Hub repository
func (c *Client) ListTags(req ListTagsRequest) (*ListTagsResponse, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/tags", dockerHubAPIURL, req.Repository)

	params := url.Values{}
	if req.PageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", req.PageSize))
	}
	if req.NameFilter != "" {
		params.Set("name", req.NameFilter)
	}

	if len(params) > 0 {
		apiURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())
	}

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var response ListTagsResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response, nil
}

// Repository represents a Docker Hub repository
type Repository struct {
	User              string   `json:"user"`
	Name              string   `json:"name"`
	Namespace         string   `json:"namespace"`
	RepositoryType    string   `json:"repository_type"`
	Status            int      `json:"status"`
	StatusDescription string   `json:"status_description,omitempty"`
	Description       string   `json:"description"`
	IsPrivate         bool     `json:"is_private"`
	IsAutomated       bool     `json:"is_automated"`
	StarCount         int      `json:"star_count"`
	PullCount         int64    `json:"pull_count"`
	LastUpdated       string   `json:"last_updated"`
	DateRegistered    string   `json:"date_registered,omitempty"`
	Affiliation       string   `json:"affiliation,omitempty"`
	MediaTypes        []string `json:"media_types,omitempty"`
	ContentTypes      []string `json:"content_types,omitempty"`
}

// GetRepository retrieves information about a repository
func (c *Client) GetRepository(repository string) (*Repository, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s", dockerHubAPIURL, repository)

	responseBody, err := c.execRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	err = json.Unmarshal(responseBody, &repo)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &repo, nil
}
