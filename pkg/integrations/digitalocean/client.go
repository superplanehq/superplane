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

// SSHKey represents a DigitalOcean SSH key
type SSHKey struct {
	ID          int    `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
}

// ListSSHKeys retrieves all SSH keys on the account
func (c *Client) ListSSHKeys() ([]SSHKey, error) {
	url := fmt.Sprintf("%s/account/keys?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		SSHKeys []SSHKey `json:"ssh_keys"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.SSHKeys, nil
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
	Name       string   `json:"name"`
	Region     string   `json:"region"`
	Size       string   `json:"size"`
	Image      string   `json:"image"`
	SSHKeys    []string `json:"ssh_keys,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	UserData   string   `json:"user_data,omitempty"`
	Backups    bool     `json:"backups,omitempty"`
	IPv6       bool     `json:"ipv6,omitempty"`
	Monitoring bool     `json:"monitoring,omitempty"`
	VpcUUID    string   `json:"vpc_uuid,omitempty"`
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
	Features []string        `json:"features"`
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

// Domain represents a DigitalOcean domain
type Domain struct {
	Name string `json:"name"`
}

// ListDomains retrieves all domains in the account
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

// DNSRecord represents a DigitalOcean DNS record
type DNSRecord struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority *int   `json:"priority"`
	Port     *int   `json:"port"`
	TTL      int    `json:"ttl"`
	Weight   *int   `json:"weight"`
}

// DNSRecordRequest is the payload for creating or updating a DNS record
type DNSRecordRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	TTL      int    `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Port     *int   `json:"port,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
}

// CreateDNSRecord creates a new DNS record for a domain
func (c *Client) CreateDNSRecord(domain string, req DNSRecordRequest) (*DNSRecord, error) {
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

// DeleteDNSRecord deletes a DNS record by its ID
func (c *Client) DeleteDNSRecord(domain string, recordID int) error {
	url := fmt.Sprintf("%s/domains/%s/records/%d", c.BaseURL, domain, recordID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// ListDNSRecords retrieves all DNS records for a domain
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

// UpdateDNSRecord updates an existing DNS record
func (c *Client) UpdateDNSRecord(domain string, recordID int, req DNSRecordRequest) (*DNSRecord, error) {
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
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	IP              string           `json:"ip"`
	Status          string           `json:"status"`
	Algorithm       string           `json:"algorithm"`
	Region          DropletRegion    `json:"region"`
	ForwardingRules []ForwardingRule `json:"forwarding_rules"`
	DropletIDs      []int            `json:"droplet_ids"`
	Tag             string           `json:"tag"`
	CreatedAt       string           `json:"created_at"`
}

// ForwardingRule defines a load balancer forwarding rule
type ForwardingRule struct {
	EntryProtocol  string `json:"entry_protocol"`
	EntryPort      int    `json:"entry_port"`
	TargetProtocol string `json:"target_protocol"`
	TargetPort     int    `json:"target_port"`
	TLSPassthrough bool   `json:"tls_passthrough,omitempty"`
}

// CreateLoadBalancerRequest is the payload for creating a load balancer
type CreateLoadBalancerRequest struct {
	Name            string           `json:"name"`
	Region          string           `json:"region"`
	ForwardingRules []ForwardingRule `json:"forwarding_rules"`
	DropletIDs      []int            `json:"droplet_ids,omitempty"`
	Tag             string           `json:"tag,omitempty"`
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

// GetLoadBalancer retrieves a load balancer by its ID
func (c *Client) GetLoadBalancer(lbID string) (*LoadBalancer, error) {
	url := fmt.Sprintf("%s/load_balancers/%s", c.BaseURL, lbID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
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

// DeleteLoadBalancer deletes a load balancer by its ID
func (c *Client) DeleteLoadBalancer(lbID string) error {
	url := fmt.Sprintf("%s/load_balancers/%s", c.BaseURL, lbID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// ListLoadBalancers retrieves all load balancers in the account
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

// ReservedIP represents a DigitalOcean reserved IP
type ReservedIP struct {
	IP         string `json:"ip"`
	RegionSlug string `json:"region_slug"`
	Locked     bool   `json:"locked"`
}

// ListReservedIPs retrieves all reserved IPs in the account
func (c *Client) ListReservedIPs() ([]ReservedIP, error) {
	url := fmt.Sprintf("%s/reserved_ips?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		ReservedIPs []ReservedIP `json:"reserved_ips"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.ReservedIPs, nil
}

// PostReservedIPAction initiates an assign or unassign action for a reserved IP
func (c *Client) PostReservedIPAction(reservedIP, actionType string, dropletID *int) (*DOAction, error) {
	url := fmt.Sprintf("%s/reserved_ips/%s/actions", c.BaseURL, reservedIP)

	payload := map[string]any{"type": actionType}
	if dropletID != nil {
		payload["droplet_id"] = *dropletID
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

// LBNodeMetadata stores metadata about a load balancer for display in the UI
type LBNodeMetadata struct {
	LBID   string `json:"lbId" mapstructure:"lbId"`
	LBName string `json:"lbName" mapstructure:"lbName"`
}

// resolveLBMetadata fetches the load balancer name from the API and stores it in metadata
func resolveLBMetadata(ctx core.SetupContext, lbID string) error {
	if strings.Contains(lbID, "{{") {
		return ctx.Metadata.Set(LBNodeMetadata{
			LBName: lbID,
		})
	}

	var existing LBNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.LBID == lbID && existing.LBName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client for metadata resolution: %w", err)
	}

	lb, err := client.GetLoadBalancer(lbID)
	if err != nil {
		return fmt.Errorf("failed to fetch load balancer %s for metadata: %w", lbID, err)
	}

	return ctx.Metadata.Set(LBNodeMetadata{
		LBID:   lbID,
		LBName: lb.Name,
	})
}

// DropletNodeMetadata stores metadata about a droplet for display in the UI
type DropletNodeMetadata struct {
	DropletID   int    `json:"dropletId" mapstructure:"dropletId"`
	DropletName string `json:"dropletName" mapstructure:"dropletName"`
}

// DNSRecordNodeMetadata stores metadata about a DNS record for display in the UI
type DNSRecordNodeMetadata struct {
	RecordID   int    `json:"recordId" mapstructure:"recordId"`
	RecordName string `json:"recordName" mapstructure:"recordName"`
}

// resolveDNSRecordMetadata fetches the DNS record name from the API and stores it in metadata
// This allows the UI to display the record name instead of just the ID
func resolveDNSRecordMetadata(ctx core.SetupContext, domain, recordIDStr string) error {
	// If the record ID is an expression placeholder, skip metadata resolution
	if strings.Contains(recordIDStr, "{{") {
		return ctx.Metadata.Set(DNSRecordNodeMetadata{
			RecordName: recordIDStr,
		})
	}

	recordID, err := strconv.Atoi(recordIDStr)
	if err != nil {
		return fmt.Errorf("invalid record ID %q: must be a number", recordIDStr)
	}

	// If metadata is already set for the same record, skip the API call
	var existing DNSRecordNodeMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.RecordID == recordID && existing.RecordName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client for metadata resolution: %w", err)
	}

	records, err := client.ListDNSRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to list DNS records for metadata: %w", err)
	}

	for _, record := range records {
		if record.ID == recordID {
			return ctx.Metadata.Set(DNSRecordNodeMetadata{
				RecordID:   recordID,
				RecordName: fmt.Sprintf("%s (%s)", record.Name, record.Type),
			})
		}
	}

	// Record not found — store the ID as the name as a fallback
	return ctx.Metadata.Set(DNSRecordNodeMetadata{
		RecordID:   recordID,
		RecordName: recordIDStr,
	})
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

// AlertPolicySlackDetails represents a Slack notification channel for an alert policy
type AlertPolicySlackDetails struct {
	URL     string `json:"url"`
	Channel string `json:"channel"`
}

// AlertPolicyAlerts represents the notification channels configured on an alert policy
type AlertPolicyAlerts struct {
	Slack []AlertPolicySlackDetails `json:"slack,omitempty"`
	Email []string                  `json:"email,omitempty"`
}

// AlertPolicy represents a DigitalOcean monitoring alert policy
type AlertPolicy struct {
	UUID        string            `json:"uuid"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Compare     string            `json:"compare"`
	Value       float64           `json:"value"`
	Window      string            `json:"window"`
	Entities    []string          `json:"entities"`
	Tags        []string          `json:"tags"`
	Alerts      AlertPolicyAlerts `json:"alerts"`
	Enabled     bool              `json:"enabled"`
}

// CreateAlertPolicyRequest is the payload for creating a monitoring alert policy
type CreateAlertPolicyRequest struct {
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Compare     string            `json:"compare"`
	Value       float64           `json:"value"`
	Window      string            `json:"window"`
	Entities    []string          `json:"entities,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Alerts      AlertPolicyAlerts `json:"alerts"`
	Enabled     bool              `json:"enabled"`
}

// UpdateAlertPolicyRequest is the payload for updating a monitoring alert policy
type UpdateAlertPolicyRequest struct {
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Compare     string            `json:"compare"`
	Value       float64           `json:"value"`
	Window      string            `json:"window"`
	Entities    []string          `json:"entities,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Alerts      AlertPolicyAlerts `json:"alerts"`
	Enabled     bool              `json:"enabled"`
}

// CreateAlertPolicy creates a new monitoring alert policy
func (c *Client) CreateAlertPolicy(req CreateAlertPolicyRequest) (*AlertPolicy, error) {
	url := fmt.Sprintf("%s/monitoring/alerts", c.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Policy AlertPolicy `json:"policy"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Policy, nil
}

// UpdateAlertPolicy updates an existing monitoring alert policy
func (c *Client) UpdateAlertPolicy(policyID string, req UpdateAlertPolicyRequest) (*AlertPolicy, error) {
	url := fmt.Sprintf("%s/monitoring/alerts/%s", c.BaseURL, policyID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Policy AlertPolicy `json:"policy"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Policy, nil
}

// GetAlertPolicy retrieves a monitoring alert policy by its UUID
func (c *Client) GetAlertPolicy(policyID string) (*AlertPolicy, error) {
	url := fmt.Sprintf("%s/monitoring/alerts/%s", c.BaseURL, policyID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Policy AlertPolicy `json:"policy"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Policy, nil
}

// DeleteAlertPolicy deletes a monitoring alert policy by its UUID
func (c *Client) DeleteAlertPolicy(policyID string) error {
	url := fmt.Sprintf("%s/monitoring/alerts/%s", c.BaseURL, policyID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// ListAlertPolicies retrieves all monitoring alert policies in the account
func (c *Client) ListAlertPolicies() ([]AlertPolicy, error) {
	url := fmt.Sprintf("%s/monitoring/alerts?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Policies []AlertPolicy `json:"policies"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Policies, nil
}

// AlertPolicyNodeMetadata stores metadata about an alert policy for display in the UI
type AlertPolicyNodeMetadata struct {
	PolicyID       string                             `json:"policyUuid" mapstructure:"policyUuid"`
	PolicyDesc     string                             `json:"policyDesc" mapstructure:"policyDesc"`
	ScopedDroplets []AlertPolicyScopedDropletMetadata `json:"scopedDroplets,omitempty" mapstructure:"scopedDroplets"`
}

// resolveAlertPolicyMetadata fetches the alert policy description from the API and stores it in metadata
func resolveAlertPolicyMetadata(ctx core.SetupContext, policyID string) error {
	var existing AlertPolicyNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if strings.Contains(policyID, "{{") {
		existing.PolicyID = ""
		existing.PolicyDesc = policyID
		return ctx.Metadata.Set(existing)
	}
	if err == nil && existing.PolicyID == policyID && existing.PolicyDesc != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client for metadata resolution: %w", err)
	}

	policy, err := client.GetAlertPolicy(policyID)
	if err != nil {
		return fmt.Errorf("failed to fetch alert policy %s for metadata: %w", policyID, err)
	}

	existing.PolicyID = policyID
	existing.PolicyDesc = policy.Description
	return ctx.Metadata.Set(existing)
}

// AlertPolicyScopedDropletMetadata stores the selected scope droplets for alert policy components.
type AlertPolicyScopedDropletMetadata struct {
	DropletID   string `json:"dropletId" mapstructure:"dropletId"`
	DropletName string `json:"dropletName" mapstructure:"dropletName"`
}

// resolveAlertPolicyEntitiesMetadata validates configured droplets and stores their labels in metadata.
func resolveAlertPolicyEntitiesMetadata(ctx core.SetupContext, dropletIDs []string) error {
	var existing AlertPolicyNodeMetadata
	_ = mapstructure.Decode(ctx.Metadata.Get(), &existing)

	scopedDroplets := make([]AlertPolicyScopedDropletMetadata, 0, len(dropletIDs))

	var client *Client
	for _, dropletID := range dropletIDs {
		if strings.Contains(dropletID, "{{") {
			scopedDroplets = append(scopedDroplets, AlertPolicyScopedDropletMetadata{
				DropletID:   dropletID,
				DropletName: dropletID,
			})
			continue
		}

		id, err := strconv.Atoi(dropletID)
		if err != nil {
			return fmt.Errorf("invalid droplet ID %q: must be a number", dropletID)
		}

		if client == nil {
			client, err = NewClient(ctx.HTTP, ctx.Integration)
			if err != nil {
				return fmt.Errorf("failed to create client for metadata resolution: %w", err)
			}
		}

		droplet, err := client.GetDroplet(id)
		if err != nil {
			return fmt.Errorf("failed to fetch droplet %s for metadata: %w", dropletID, err)
		}

		scopedDroplets = append(scopedDroplets, AlertPolicyScopedDropletMetadata{
			DropletID:   dropletID,
			DropletName: droplet.Name,
		})
	}

	existing.ScopedDroplets = scopedDroplets
	return ctx.Metadata.Set(existing)
}

// MetricsValue represents a single data point in a metric series: [unix_timestamp, string_value]
type MetricsValue []any

// MetricsResult represents a single labeled metric time series
type MetricsResult struct {
	Metric map[string]string `json:"metric"`
	Values []MetricsValue    `json:"values"`
}

// MetricsData is the data envelope returned by the monitoring metrics API
type MetricsData struct {
	ResultType string          `json:"resultType"`
	Result     []MetricsResult `json:"result"`
}

// MetricsResponse is the top-level response from the monitoring metrics API
type MetricsResponse struct {
	Status string      `json:"status"`
	Data   MetricsData `json:"data"`
}

// GetDropletCPUMetrics fetches CPU usage percentage metrics for a droplet
func (c *Client) GetDropletCPUMetrics(dropletID string, start, end int64) (*MetricsResponse, error) {
	url := fmt.Sprintf("%s/monitoring/metrics/droplet/cpu?host_id=%s&start=%d&end=%d", c.BaseURL, dropletID, start, end)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response MetricsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response, nil
}

// GetDropletMemoryAvailableMetrics fetches available memory metrics for a droplet.
// Available memory includes free memory and reclaimable cache, matching what
// DigitalOcean's dashboard reports as memory utilization.
func (c *Client) GetDropletMemoryAvailableMetrics(dropletID string, start, end int64) (*MetricsResponse, error) {
	url := fmt.Sprintf("%s/monitoring/metrics/droplet/memory_available?host_id=%s&start=%d&end=%d", c.BaseURL, dropletID, start, end)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response MetricsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response, nil
}

// GetDropletMemoryTotalMetrics fetches total memory metrics for a droplet
func (c *Client) GetDropletMemoryTotalMetrics(dropletID string, start, end int64) (*MetricsResponse, error) {
	url := fmt.Sprintf("%s/monitoring/metrics/droplet/memory_total?host_id=%s&start=%d&end=%d", c.BaseURL, dropletID, start, end)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response MetricsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response, nil
}

// GetDropletBandwidthMetrics fetches network bandwidth metrics for a droplet.
func (c *Client) GetDropletBandwidthMetrics(dropletID, iface, direction string, start, end int64) (*MetricsResponse, error) {
	url := fmt.Sprintf("%s/monitoring/metrics/droplet/bandwidth?host_id=%s&interface=%s&direction=%s&start=%d&end=%d", c.BaseURL, dropletID, iface, direction, start, end)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response MetricsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response, nil
}

// VPC represents a DigitalOcean VPC
type VPC struct {
	ID          string `json:"id"`
	URN         string `json:"urn"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RegionSlug  string `json:"region"`
	IPRange     string `json:"ip_range"`
	CreatedAt   string `json:"created_at"`
	Default     bool   `json:"default"`
}

// ListVPCs retrieves all VPCs in the account
func (c *Client) ListVPCs() ([]VPC, error) {
	url := fmt.Sprintf("%s/vpcs?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		VPCs []VPC `json:"vpcs"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.VPCs, nil
}
