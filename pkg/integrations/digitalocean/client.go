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

// DeleteDroplet deletes a droplet by ID
func (c *Client) DeleteDroplet(dropletID int) error {
	url := fmt.Sprintf("%s/droplets/%d", c.BaseURL, dropletID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// PerformDropletAction performs an action on a droplet (e.g. power_on, shutdown, reboot)
func (c *Client) PerformDropletAction(dropletID int, actionType string) (*DOAction, error) {
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

// CreateDropletSnapshot creates a snapshot of a droplet
func (c *Client) CreateDropletSnapshot(dropletID int, name string) (*DOAction, error) {
	url := fmt.Sprintf("%s/droplets/%d/actions", c.BaseURL, dropletID)
	payload := map[string]string{"type": "snapshot"}
	if name != "" {
		payload["name"] = name
	}

	body, err := json.Marshal(payload)
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

// GetAction retrieves a single action by ID
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

// Snapshot represents a DigitalOcean snapshot
type Snapshot struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	CreatedAt     string   `json:"created_at"`
	Regions       []string `json:"regions"`
	ResourceID    string   `json:"resource_id"`
	ResourceType  string   `json:"resource_type"`
	MinDiskSize   int      `json:"min_disk_size"`
	SizeGigabytes float64  `json:"size_gigabytes"`
}

// DeleteSnapshot deletes a snapshot by ID
func (c *Client) DeleteSnapshot(snapshotID string) error {
	url := fmt.Sprintf("%s/snapshots/%s", c.BaseURL, snapshotID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// DNSRecord represents a DigitalOcean domain record
type DNSRecord struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority *int   `json:"priority,omitempty"`
	Port     *int   `json:"port,omitempty"`
	TTL      int    `json:"ttl"`
	Weight   *int   `json:"weight,omitempty"`
	Flags    *int   `json:"flags,omitempty"`
	Tag      string `json:"tag,omitempty"`
}

// CreateDNSRecordRequest is the payload for creating a DNS record
type CreateDNSRecordRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority *int   `json:"priority,omitempty"`
	Port     *int   `json:"port,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
	Flags    *int   `json:"flags,omitempty"`
	Tag      string `json:"tag,omitempty"`
}

// CreateDNSRecord creates a DNS record for a domain
func (c *Client) CreateDNSRecord(domain string, req CreateDNSRecordRequest) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/domains/%s/records", c.BaseURL, domain)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		DomainRecord DNSRecord `json:"domain_record"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.DomainRecord, nil
}

// DeleteDNSRecord deletes a DNS record
func (c *Client) DeleteDNSRecord(domain string, recordID int) error {
	url := fmt.Sprintf("%s/domains/%s/records/%d", c.BaseURL, domain, recordID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// ListDNSRecords lists all DNS records for a domain
func (c *Client) ListDNSRecords(domain string) ([]DNSRecord, error) {
	url := fmt.Sprintf("%s/domains/%s/records?per_page=200", c.BaseURL, domain)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		DomainRecords []DNSRecord `json:"domain_records"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.DomainRecords, nil
}

// UpdateDNSRecordRequest is the payload for updating a DNS record
type UpdateDNSRecordRequest struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Data     string `json:"data,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Port     *int   `json:"port,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
	Flags    *int   `json:"flags,omitempty"`
	Tag      string `json:"tag,omitempty"`
}

// UpdateDNSRecord updates a DNS record
func (c *Client) UpdateDNSRecord(domain string, recordID int, req UpdateDNSRecordRequest) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/domains/%s/records/%d", c.BaseURL, domain, recordID)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		DomainRecord DNSRecord `json:"domain_record"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.DomainRecord, nil
}

// LoadBalancer represents a DigitalOcean load balancer
type LoadBalancer struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	IP                  string           `json:"ip"`
	Status              string           `json:"status"`
	Region              DropletRegion    `json:"region"`
	Algorithm           string           `json:"algorithm"`
	ForwardingRules     []ForwardingRule `json:"forwarding_rules"`
	HealthCheck         *HealthCheck     `json:"health_check,omitempty"`
	StickySessions      *StickySessions  `json:"sticky_sessions,omitempty"`
	DropletIDs          []int            `json:"droplet_ids"`
	RedirectHttpToHttps bool             `json:"redirect_http_to_https"`
}

// ForwardingRule represents a load balancer forwarding rule
type ForwardingRule struct {
	EntryProtocol  string `json:"entry_protocol"`
	EntryPort      int    `json:"entry_port"`
	TargetProtocol string `json:"target_protocol"`
	TargetPort     int    `json:"target_port"`
}

// HealthCheck represents a load balancer health check
type HealthCheck struct {
	Protocol               string `json:"protocol"`
	Port                   int    `json:"port"`
	Path                   string `json:"path"`
	CheckIntervalSeconds   int    `json:"check_interval_seconds"`
	ResponseTimeoutSeconds int    `json:"response_timeout_seconds"`
	UnhealthyThreshold     int    `json:"unhealthy_threshold"`
	HealthyThreshold       int    `json:"healthy_threshold"`
}

// StickySessions represents a load balancer sticky sessions configuration
type StickySessions struct {
	Type             string `json:"type"`
	CookieName       string `json:"cookie_name,omitempty"`
	CookieTTLSeconds int    `json:"cookie_ttl_seconds,omitempty"`
}

// CreateLoadBalancerRequest is the payload for creating a load balancer
type CreateLoadBalancerRequest struct {
	Name            string           `json:"name"`
	Region          string           `json:"region"`
	Algorithm       string           `json:"algorithm,omitempty"`
	ForwardingRules []ForwardingRule `json:"forwarding_rules"`
	HealthCheck     *HealthCheck     `json:"health_check,omitempty"`
	DropletIDs      []int            `json:"droplet_ids,omitempty"`
}

// CreateLoadBalancer creates a new load balancer
func (c *Client) CreateLoadBalancer(req CreateLoadBalancerRequest) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/load_balancers", c.BaseURL)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		LoadBalancer LoadBalancer `json:"load_balancer"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.LoadBalancer, nil
}

// DeleteLoadBalancer deletes a load balancer by ID
func (c *Client) DeleteLoadBalancer(loadBalancerID string) error {
	url := fmt.Sprintf("%s/load_balancers/%s", c.BaseURL, loadBalancerID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// ListDroplets retrieves all droplets
func (c *Client) ListDroplets() ([]Droplet, error) {
	url := fmt.Sprintf("%s/droplets?per_page=200", c.BaseURL)
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

// ListSnapshots retrieves all snapshots
func (c *Client) ListSnapshots() ([]Snapshot, error) {
	url := fmt.Sprintf("%s/snapshots?per_page=200&resource_type=droplet", c.BaseURL)
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

// Domain represents a DigitalOcean domain
type Domain struct {
	Name string `json:"name"`
}

// ListDomains retrieves all domains
func (c *Client) ListDomains() ([]Domain, error) {
	url := fmt.Sprintf("%s/domains?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Domains []Domain `json:"domains"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Domains, nil
}

// ListLoadBalancers retrieves all load balancers
func (c *Client) ListLoadBalancers() ([]LoadBalancer, error) {
	url := fmt.Sprintf("%s/load_balancers?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		LoadBalancers []LoadBalancer `json:"load_balancers"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.LoadBalancers, nil
}

// ReservedIPAction represents a reserved IP action request
type ReservedIPAction struct {
	Type      string `json:"type"`
	DropletID int    `json:"droplet_id,omitempty"`
}

// AssignReservedIP assigns a reserved IP to a droplet
func (c *Client) AssignReservedIP(reservedIP string, dropletID int) (*DOAction, error) {
	url := fmt.Sprintf("%s/reserved_ips/%s/actions", c.BaseURL, reservedIP)
	body, err := json.Marshal(ReservedIPAction{Type: "assign", DropletID: dropletID})
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

// UnassignReservedIP unassigns a reserved IP from a droplet
func (c *Client) UnassignReservedIP(reservedIP string) (*DOAction, error) {
	url := fmt.Sprintf("%s/reserved_ips/%s/actions", c.BaseURL, reservedIP)
	body, err := json.Marshal(ReservedIPAction{Type: "unassign"})
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
