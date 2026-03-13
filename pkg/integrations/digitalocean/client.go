package digitalocean

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
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

// ListDroplets retrieves all droplets in the account
func (c *Client) ListDroplets() ([]Droplet, error) {
	url := fmt.Sprintf("%s/droplets", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Droplets []Droplet `json:"droplets"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Droplets, nil
}

// DeleteDroplet deletes a droplet by its ID
func (c *Client) DeleteDroplet(dropletID int) error {
	url := fmt.Sprintf("%s/droplets/%d", c.BaseURL, dropletID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	return nil
}

// PostDropletAction initiates a power action on a droplet
func (c *Client) PostDropletAction(dropletID int, actionType string) (*DOAction, error) {
	url := fmt.Sprintf("%s/droplets/%d/actions", c.BaseURL, dropletID)

	body, err := json.Marshal(map[string]string{"type": actionType})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Action DOAction `json:"action"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Action, nil
}

// GetAction retrieves an action by its ID
func (c *Client) GetAction(actionID int) (*DOAction, error) {
	url := fmt.Sprintf("%s/actions/%d", c.BaseURL, actionID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Action DOAction `json:"action"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Action, nil
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

// DropletNodeMetadata stores metadata about a droplet for display in the UI
type DropletNodeMetadata struct {
	DropletID   int    `json:"dropletId" mapstructure:"dropletId"`
	DropletName string `json:"dropletName" mapstructure:"dropletName"`
}

// resolveDropletMetadata fetches the droplet name from the API and stores it in metadata
// This allows the UI to display the droplet name instead of just the ID
func resolveDropletMetadata(ctx core.SetupContext, dropletStr string) error {
	// If the droplet ID is an expression placeholder, skip metadata resolution
	if strings.Contains(dropletStr, "{{") {
		return ctx.Metadata.Set(DropletNodeMetadata{
			DropletName: dropletStr,
		})
	}
	dropletID, err := strconv.Atoi(dropletStr)
	if err != nil {
		return fmt.Errorf("invalid droplet ID %q: must be a number", dropletStr)
	}

	// If metadata is already set for the same droplet, skip the API call
	var existing DropletNodeMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.DropletID == dropletID && existing.DropletName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client for metadata resolution: %w", err)
	}

	droplet, err := client.GetDroplet(dropletID)
	if err != nil {
		return fmt.Errorf("failed to fetch droplet %d for metadata: %w", dropletID, err)
	}

	return ctx.Metadata.Set(DropletNodeMetadata{
		DropletID:   dropletID,
		DropletName: droplet.Name,
	})
}

// SnapshotNodeMetadata stores metadata about a snapshot for display in the UI
type SnapshotNodeMetadata struct {
	SnapshotID   string `json:"snapshotId" mapstructure:"snapshotId"`
	SnapshotName string `json:"snapshotName" mapstructure:"snapshotName"`
}

// resolveSnapshotMetadata fetches the snapshot name from the API and stores it in metadata
// This allows the UI to display the snapshot name instead of just the ID
func resolveSnapshotMetadata(ctx core.SetupContext, snapshotStr string) error {
	// If the snapshot ID is an expression placeholder, skip metadata resolution
	if strings.Contains(snapshotStr, "{{") {
		return ctx.Metadata.Set(SnapshotNodeMetadata{
			SnapshotName: snapshotStr,
		})
	}

	// If metadata is already set for the same snapshot, skip the API call
	var existing SnapshotNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.SnapshotID == snapshotStr && existing.SnapshotName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client for metadata resolution: %w", err)
	}

	snapshot, err := client.GetSnapshot(snapshotStr)
	if err != nil {
		return fmt.Errorf("failed to fetch snapshot %s for metadata: %w", snapshotStr, err)
	}

	return ctx.Metadata.Set(SnapshotNodeMetadata{
		SnapshotID:   snapshotStr,
		SnapshotName: snapshot.Name,
	})
}

// CreateDropletSnapshot creates a snapshot of a droplet
func (c *Client) CreateDropletSnapshot(dropletID int, name string) (*DOAction, error) {
	url := fmt.Sprintf("%s/droplets/%d/actions", c.BaseURL, dropletID)

	body, err := json.Marshal(map[string]string{
		"type": "snapshot",
		"name": name,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Action DOAction `json:"action"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Action, nil
}

// Snapshot represents a DigitalOcean snapshot
type Snapshot struct {
	ID            json.Number `json:"id"`
	Name          string      `json:"name"`
	CreatedAt     string      `json:"created_at"`
	ResourceID    string      `json:"resource_id"`
	ResourceType  string      `json:"resource_type"`
	Regions       []string    `json:"regions"`
	MinDiskSize   int         `json:"min_disk_size"`
	SizeGigabytes float64     `json:"size_gigabytes"`
}

// GetDropletSnapshots lists snapshots for a given droplet
func (c *Client) GetDropletSnapshots(dropletID int) ([]Snapshot, error) {
	url := fmt.Sprintf("%s/droplets/%d/snapshots", c.BaseURL, dropletID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Snapshots []Snapshot `json:"snapshots"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}
	for i := range response.Snapshots {
		response.Snapshots[i].ResourceID = strconv.Itoa(dropletID)
		response.Snapshots[i].ResourceType = "droplet"
	}

	return response.Snapshots, nil
}

// GetSnapshot retrieves a single snapshot by ID
func (c *Client) GetSnapshot(snapshotID string) (*Snapshot, error) {
	url := fmt.Sprintf("%s/snapshots/%s", c.BaseURL, snapshotID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Snapshot Snapshot `json:"snapshot"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Snapshot, nil
}

// DeleteSnapshot deletes a snapshot by ID
func (c *Client) DeleteSnapshot(snapshotID string) error {
	url := fmt.Sprintf("%s/snapshots/%s", c.BaseURL, snapshotID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// ListSnapshots retrieves all droplet snapshots
func (c *Client) ListSnapshots() ([]Snapshot, error) {
	url := fmt.Sprintf("%s/snapshots?resource_type=droplet&per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Snapshots []Snapshot `json:"snapshots"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Snapshots, nil
}
