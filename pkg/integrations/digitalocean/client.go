package digitalocean

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.digitalocean.com/v2"

type Client struct {
	Token   string
	http    core.HTTPContext
	BaseURL string
}

type DOAPIError struct {
	StatusCode int
	Body       []byte
}

func (e *DOAPIError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.StatusCode, string(e.Body))
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error finding API token: %v", err)
	}

	return &Client{
		Token:   string(apiToken),
		http:    http,
		BaseURL: baseURL,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

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
		return nil, &DOAPIError{
			StatusCode: res.StatusCode,
			Body:       responseBody,
		}
	}

	return responseBody, nil
}

// Account represents a DigitalOcean account
type Account struct {
	Email        string `json:"email"`
	UUID         string `json:"uuid"`
	Status       string `json:"status"`
	DropletLimit int    `json:"droplet_limit"`
}

// GetAccount validates the API token by fetching account info
func (c *Client) GetAccount() (*Account, error) {
	url := fmt.Sprintf("%s/account", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Account Account `json:"account"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Account, nil
}

// Region represents a DigitalOcean region
type Region struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// ListRegions retrieves all available regions
func (c *Client) ListRegions() ([]Region, error) {
	url := fmt.Sprintf("%s/regions", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Regions []Region `json:"regions"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Regions, nil
}

// Size represents a DigitalOcean droplet size
type Size struct {
	Slug         string  `json:"slug"`
	Memory       int     `json:"memory"`
	VCPUs        int     `json:"vcpus"`
	Disk         int     `json:"disk"`
	PriceMonthly float64 `json:"price_monthly"`
	Available    bool    `json:"available"`
}

// ListSizes retrieves all available droplet sizes
func (c *Client) ListSizes() ([]Size, error) {
	url := fmt.Sprintf("%s/sizes?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Sizes []Size `json:"sizes"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Sizes, nil
}

// Image represents a DigitalOcean image
type Image struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Type         string `json:"type"`
	Distribution string `json:"distribution"`
}

// ListImages retrieves images of a given type (e.g., "distribution")
func (c *Client) ListImages(imageType string) ([]Image, error) {
	url := fmt.Sprintf("%s/images?type=%s&per_page=200", c.BaseURL, imageType)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Images []Image `json:"images"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Images, nil
}

// CreateDropletRequest is the payload for creating a droplet
type CreateDropletRequest struct {
	Name     string   `json:"name"`
	Region   string   `json:"region"`
	Size     string   `json:"size"`
	Image    string   `json:"image"`
	SSHKeys  []string `json:"ssh_keys,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	UserData string   `json:"user_data,omitempty"`
}

// Droplet represents a DigitalOcean droplet
type Droplet struct {
	ID       int             `json:"id"`
	Name     string          `json:"name"`
	Memory   int             `json:"memory"`
	VCPUs    int             `json:"vcpus"`
	Disk     int             `json:"disk"`
	Status   string          `json:"status"`
	Region   DropletRegion   `json:"region"`
	Image    DropletImage    `json:"image"`
	SizeSlug string          `json:"size_slug"`
	Networks DropletNetworks `json:"networks"`
	Tags     []string        `json:"tags"`
}

type DropletRegion struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type DropletImage struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type DropletNetworks struct {
	V4 []DropletNetworkV4 `json:"v4"`
}

type DropletNetworkV4 struct {
	IPAddress string `json:"ip_address"`
	Type      string `json:"type"`
}

// CreateDroplet creates a new droplet
func (c *Client) CreateDroplet(req CreateDropletRequest) (*Droplet, error) {
	url := fmt.Sprintf("%s/droplets", c.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Droplet Droplet `json:"droplet"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Droplet, nil
}

// GetDroplet retrieves a droplet by its ID
func (c *Client) GetDroplet(dropletID int) (*Droplet, error) {
	url := fmt.Sprintf("%s/droplets/%d", c.BaseURL, dropletID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Droplet Droplet `json:"droplet"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Droplet, nil
}

// DOAction represents a DigitalOcean action
type DOAction struct {
	ID           int    `json:"id"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at"`
	ResourceID   int    `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	RegionSlug   string `json:"region_slug"`
}

// ListActions retrieves actions filtered by resource type.
// The DigitalOcean /v2/actions API does not support resource_type as a query
// parameter, so we fetch all recent actions and filter client-side.
func (c *Client) ListActions(resourceType string) ([]DOAction, error) {
	url := fmt.Sprintf("%s/actions?page=1&per_page=50", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Actions []DOAction `json:"actions"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	filtered := make([]DOAction, 0, len(response.Actions))
	for _, a := range response.Actions {
		if a.ResourceType == resourceType {
			filtered = append(filtered, a)
		}
	}

	return filtered, nil
}
