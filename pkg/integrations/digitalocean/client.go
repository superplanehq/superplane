package digitalocean

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// SpacesClient handles S3-compatible requests to DigitalOcean Spaces.
// It uses AWS Signature Version 4 for authentication.
type SpacesClient struct {
	AccessKey string
	SecretKey string
	Region    string
	http      core.HTTPContext
}

// NewSpacesClient creates a new SpacesClient using credentials from the integration configuration.
func NewSpacesClient(httpCtx core.HTTPContext, ctx core.IntegrationContext, region string) (*SpacesClient, error) {
	accessKey, err := ctx.GetConfig("spacesAccessKey")
	if err != nil || string(accessKey) == "" {
		return nil, fmt.Errorf("spaces access key is required — set it in the DigitalOcean integration configuration")
	}

	secretKey, err := ctx.GetConfig("spacesSecretKey")
	if err != nil || string(secretKey) == "" {
		return nil, fmt.Errorf("spaces secret key is required — set it in the DigitalOcean integration configuration")
	}

	return &SpacesClient{
		AccessKey: string(accessKey),
		SecretKey: string(secretKey),
		Region:    region,
		http:      httpCtx,
	}, nil
}

// execSpacesRequest signs and executes an HTTP request to the Spaces S3-compatible API.
// If bucket is empty, a region-level request is made (e.g. ListBuckets).
// queryString is already encoded, e.g. "" or "tagging=".
func (c *SpacesClient) execSpacesRequest(method, bucket, key, queryString string) (*http.Response, error) {
	var host, endpoint string
	if bucket == "" {
		host = fmt.Sprintf("%s.digitaloceanspaces.com", c.Region)
		endpoint = fmt.Sprintf("https://%s/", host)
	} else {
		host = fmt.Sprintf("%s.%s.digitaloceanspaces.com", bucket, c.Region)
		endpoint = fmt.Sprintf("https://%s/%s", host, key)
	}

	if queryString != "" {
		endpoint += "?" + queryString
	}

	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	payloadHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	path := "/"
	if key != "" {
		path = "/" + key
	}

	canonicalRequest := strings.Join([]string{
		method,
		path,
		queryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.Region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest))),
	}, "\n")

	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256(
				hmacSHA256([]byte("AWS4"+c.SecretKey), dateStamp),
				c.Region,
			),
			"s3",
		),
		"aws4_request",
	)

	signature := fmt.Sprintf("%x", hmacSHA256(signingKey, stringToSign))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s,SignedHeaders=%s,Signature=%s",
		c.AccessKey, credentialScope, signedHeaders, signature,
	)

	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("Authorization", authHeader)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	return res, nil
}

// ObjectResult holds the response from a GetObject call.
type ObjectResult struct {
	ContentType   string
	ContentLength int64
	LastModified  string
	ETag          string
	Metadata      map[string]string
	Body          []byte
	NotFound      bool
}

// GetObject retrieves an object (and optionally its body) from a Spaces bucket.
func (c *SpacesClient) GetObject(bucket, key string, includeBody bool) (*ObjectResult, error) {
	method := http.MethodGet
	if !includeBody {
		method = http.MethodHead
	}

	res, err := c.execSpacesRequest(method, bucket, key, "")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return &ObjectResult{NotFound: true}, nil
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(bodyBytes))
	}

	result := &ObjectResult{
		ContentType:  res.Header.Get("Content-Type"),
		LastModified: res.Header.Get("Last-Modified"),
		ETag:         strings.Trim(res.Header.Get("ETag"), `"`),
		Metadata:     map[string]string{},
	}

	if cl := res.Header.Get("Content-Length"); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
			result.ContentLength = n
		}
	}

	for k, v := range res.Header {
		lower := strings.ToLower(k)
		if strings.HasPrefix(lower, "x-amz-meta-") {
			metaKey := strings.TrimPrefix(lower, "x-amz-meta-")
			if len(v) > 0 {
				result.Metadata[metaKey] = v[0]
			}
		}
	}

	if includeBody && isReadableContentType(result.ContentType) {
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading body: %v", err)
		}
		result.Body = bodyBytes
	}

	return result, nil
}

// ListBuckets returns all Spaces buckets in the client's region.
func (c *SpacesClient) ListBuckets() ([]string, error) {
	res, err := c.execSpacesRequest(http.MethodGet, "", "", "")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading list buckets response: %v", err)
	}

	var result struct {
		Buckets struct {
			Buckets []struct {
				Name string `xml:"Name"`
			} `xml:"Bucket"`
		} `xml:"Buckets"`
	}

	if err := xml.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("error parsing list buckets response: %v", err)
	}

	names := make([]string, 0, len(result.Buckets.Buckets))
	for _, b := range result.Buckets.Buckets {
		names = append(names, b.Name)
	}

	return names, nil
}

// GetObjectTagging retrieves the tags applied to an object in a Spaces bucket.
func (c *SpacesClient) GetObjectTagging(bucket, key string) (map[string]string, error) {
	res, err := c.execSpacesRequest(http.MethodGet, bucket, key, "tagging=")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading tagging response: %v", err)
	}

	var tagging struct {
		TagSet struct {
			Tags []struct {
				Key   string `xml:"Key"`
				Value string `xml:"Value"`
			} `xml:"Tag"`
		} `xml:"TagSet"`
	}

	if err := xml.Unmarshal(bodyBytes, &tagging); err != nil {
		return nil, fmt.Errorf("error parsing tagging response: %v", err)
	}

	tags := make(map[string]string, len(tagging.TagSet.Tags))
	for _, tag := range tagging.TagSet.Tags {
		tags[tag.Key] = tag.Value
	}

	return tags, nil
}

// FormatSize converts a byte count to a human-readable string (KiB, MiB, GiB).
func FormatSize(bytes int64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
	)

	switch {
	case bytes >= gib:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/float64(gib))
	case bytes >= mib:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/float64(mib))
	case bytes >= kib:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/float64(kib))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// isReadableContentType returns true if the content type is human-readable text.
func isReadableContentType(contentType string) bool {
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	readable := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/yaml",
		"application/x-yaml",
		"application/javascript",
		"application/toml",
		"application/x-www-form-urlencoded",
	}
	for _, r := range readable {
		if strings.HasPrefix(ct, r) {
			return true
		}
	}
	return false
}

// hmacSHA256 computes HMAC-SHA256 of data with key.
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
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

// App represents a DigitalOcean App Platform application
type App struct {
	ID                      string            `json:"id"`
	Spec                    *AppSpec          `json:"spec"`
	DefaultIngress          string            `json:"default_ingress,omitempty"`
	LiveURL                 string            `json:"live_url,omitempty"`
	LiveURLBase             string            `json:"live_url_base,omitempty"`
	LiveDomain              string            `json:"live_domain,omitempty"`
	ActiveDeployment        *Deployment       `json:"active_deployment,omitempty"`
	InProgressDeployment    *Deployment       `json:"in_progress_deployment,omitempty"`
	LastDeploymentCreatedAt string            `json:"last_deployment_created_at,omitempty"`
	CreatedAt               string            `json:"created_at"`
	UpdatedAt               string            `json:"updated_at"`
	Region                  *AppRegion        `json:"region,omitempty"`
	PendingDeployment       PendingDeployment `json:"pending_deployment,omitempty"`
}

// AppSpec defines the specification for an App Platform application
type AppSpec struct {
	Name        string           `json:"name"`
	Region      string           `json:"region,omitempty"`
	Services    []*AppService    `json:"services,omitempty"`
	Workers     []*AppWorker     `json:"workers,omitempty"`
	Jobs        []*AppJob        `json:"jobs,omitempty"`
	StaticSites []*AppStaticSite `json:"static_sites,omitempty"`
	Databases   []*AppDatabase   `json:"databases,omitempty"`
	Domains     []*AppDomain     `json:"domains,omitempty"`
	Ingress     *AppIngress      `json:"ingress,omitempty"`
	VPC         *AppVPC          `json:"vpc,omitempty"`
}

// AppIngress defines ingress rules for an app
type AppIngress struct {
	Rules []*AppIngressRule `json:"rules,omitempty"`
}

// AppIngressRule defines a single ingress rule
type AppIngressRule struct {
	Match     *AppIngressRuleMatch     `json:"match,omitempty"`
	Component *AppIngressRuleComponent `json:"component,omitempty"`
	CORS      *AppCORS                 `json:"cors,omitempty"`
}

// AppIngressRuleMatch defines the match criteria for an ingress rule
type AppIngressRuleMatch struct {
	Path *AppIngressRuleMatchPath `json:"path,omitempty"`
}

// AppIngressRuleMatchPath defines path matching for an ingress rule
type AppIngressRuleMatchPath struct {
	Prefix string `json:"prefix,omitempty"`
}

// AppIngressRuleComponent references a component in an ingress rule
type AppIngressRuleComponent struct {
	Name               string `json:"name"`
	PreservePathPrefix bool   `json:"preserve_path_prefix,omitempty"`
}

// AppCORS defines Cross-Origin Resource Sharing configuration
type AppCORS struct {
	AllowOrigins     []*AppCORSAllowOrigin `json:"allow_origins,omitempty"`
	AllowMethods     []string              `json:"allow_methods,omitempty"`
	AllowHeaders     []string              `json:"allow_headers,omitempty"`
	ExposeHeaders    []string              `json:"expose_headers,omitempty"`
	MaxAge           string                `json:"max_age,omitempty"`
	AllowCredentials bool                  `json:"allow_credentials,omitempty"`
}

// AppCORSAllowOrigin defines an allowed origin for CORS
type AppCORSAllowOrigin struct {
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

// AppService represents a service component in an app
type AppService struct {
	Name             string           `json:"name"`
	GitHub           *GitHubSource    `json:"github,omitempty"`
	GitLab           *GitLabSource    `json:"gitlab,omitempty"`
	Bitbucket        *BitbucketSource `json:"bitbucket,omitempty"`
	Git              *GitSource       `json:"git,omitempty"`
	Image            *ImageSource     `json:"image,omitempty"`
	EnvironmentSlug  string           `json:"environment_slug,omitempty"`
	EnvVariables     []*AppEnvVar     `json:"envs,omitempty"`
	InstanceCount    int64            `json:"instance_count,omitempty"`
	InstanceSizeSlug string           `json:"instance_size_slug,omitempty"`
	Routes           []*AppRoute      `json:"routes,omitempty"`
	HealthCheck      *HealthCheck     `json:"health_check,omitempty"`
	HTTPPort         int64            `json:"http_port,omitempty"`
	RunCommand       string           `json:"run_command,omitempty"`
	BuildCommand     string           `json:"build_command,omitempty"`
	SourceDir        string           `json:"source_dir,omitempty"`
}

// AppWorker represents a worker component in an app
type AppWorker struct {
	Name             string           `json:"name"`
	GitHub           *GitHubSource    `json:"github,omitempty"`
	GitLab           *GitLabSource    `json:"gitlab,omitempty"`
	Bitbucket        *BitbucketSource `json:"bitbucket,omitempty"`
	Git              *GitSource       `json:"git,omitempty"`
	Image            *ImageSource     `json:"image,omitempty"`
	EnvironmentSlug  string           `json:"environment_slug,omitempty"`
	EnvVariables     []*AppEnvVar     `json:"envs,omitempty"`
	InstanceCount    int64            `json:"instance_count,omitempty"`
	InstanceSizeSlug string           `json:"instance_size_slug,omitempty"`
	RunCommand       string           `json:"run_command,omitempty"`
	BuildCommand     string           `json:"build_command,omitempty"`
	SourceDir        string           `json:"source_dir,omitempty"`
}

// AppJob represents a job component in an app
type AppJob struct {
	Name             string           `json:"name"`
	GitHub           *GitHubSource    `json:"github,omitempty"`
	GitLab           *GitLabSource    `json:"gitlab,omitempty"`
	Bitbucket        *BitbucketSource `json:"bitbucket,omitempty"`
	Git              *GitSource       `json:"git,omitempty"`
	Image            *ImageSource     `json:"image,omitempty"`
	EnvironmentSlug  string           `json:"environment_slug,omitempty"`
	EnvVariables     []*AppEnvVar     `json:"envs,omitempty"`
	InstanceCount    int64            `json:"instance_count,omitempty"`
	InstanceSizeSlug string           `json:"instance_size_slug,omitempty"`
	Kind             string           `json:"kind,omitempty"`
	RunCommand       string           `json:"run_command,omitempty"`
	BuildCommand     string           `json:"build_command,omitempty"`
	SourceDir        string           `json:"source_dir,omitempty"`
}

// AppStaticSite represents a static site component in an app
type AppStaticSite struct {
	Name             string           `json:"name"`
	GitHub           *GitHubSource    `json:"github,omitempty"`
	GitLab           *GitLabSource    `json:"gitlab,omitempty"`
	Bitbucket        *BitbucketSource `json:"bitbucket,omitempty"`
	Git              *GitSource       `json:"git,omitempty"`
	EnvironmentSlug  string           `json:"environment_slug,omitempty"`
	EnvVariables     []*AppEnvVar     `json:"envs,omitempty"`
	BuildCommand     string           `json:"build_command,omitempty"`
	OutputDir        string           `json:"output_dir,omitempty"`
	SourceDir        string           `json:"source_dir,omitempty"`
	IndexDocument    string           `json:"index_document,omitempty"`
	ErrorDocument    string           `json:"error_document,omitempty"`
	CatchallDocument string           `json:"catchall_document,omitempty"`
	Routes           []*AppRoute      `json:"routes,omitempty"`
}

// AppDatabase represents a database component in an app
type AppDatabase struct {
	Name        string `json:"name"`
	Engine      string `json:"engine,omitempty"`
	Version     string `json:"version,omitempty"`
	Production  bool   `json:"production,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	DBName      string `json:"db_name,omitempty"`
	DBUser      string `json:"db_user,omitempty"`
}

// AppDomain represents a custom domain for an app
type AppDomain struct {
	Domain   string `json:"domain"`
	Type     string `json:"type,omitempty"`
	Wildcard bool   `json:"wildcard,omitempty"`
	Zone     string `json:"zone,omitempty"`
}

// AppVPC represents a VPC configuration for an app
type AppVPC struct {
	ID string `json:"id"`
}

// GitHubSource represents a GitHub repository source
type GitHubSource struct {
	Repo         string `json:"repo"`
	Branch       string `json:"branch,omitempty"`
	DeployOnPush bool   `json:"deploy_on_push"`
}

// GitLabSource represents a GitLab repository source
type GitLabSource struct {
	Repo         string `json:"repo"`
	Branch       string `json:"branch,omitempty"`
	DeployOnPush bool   `json:"deploy_on_push"`
}

// BitbucketSource represents a Bitbucket repository source
type BitbucketSource struct {
	Repo         string `json:"repo"`
	Branch       string `json:"branch,omitempty"`
	DeployOnPush bool   `json:"deploy_on_push"`
}

// GitSource represents a generic Git repository source
type GitSource struct {
	RepoCloneURL string `json:"repo_clone_url"`
	Branch       string `json:"branch,omitempty"`
}

// ImageSource represents a container image source
type ImageSource struct {
	RegistryType string `json:"registry_type"`
	Registry     string `json:"registry,omitempty"`
	Repository   string `json:"repository"`
	Tag          string `json:"tag,omitempty"`
}

// AppEnvVar represents an environment variable
type AppEnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	Scope string `json:"scope,omitempty"`
	Type  string `json:"type,omitempty"`
}

// AppRoute represents a route configuration for a service
type AppRoute struct {
	Path               string `json:"path,omitempty"`
	PreservePathPrefix bool   `json:"preserve_path_prefix,omitempty"`
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	HTTPPath            string `json:"http_path,omitempty"`
	InitialDelaySeconds int32  `json:"initial_delay_seconds,omitempty"`
	PeriodSeconds       int32  `json:"period_seconds,omitempty"`
	TimeoutSeconds      int32  `json:"timeout_seconds,omitempty"`
	SuccessThreshold    int32  `json:"success_threshold,omitempty"`
	FailureThreshold    int32  `json:"failure_threshold,omitempty"`
}

// Deployment represents an app deployment
type Deployment struct {
	ID          string                  `json:"id"`
	Spec        *AppSpec                `json:"spec,omitempty"`
	Services    []*DeploymentService    `json:"services,omitempty"`
	Workers     []*DeploymentWorker     `json:"workers,omitempty"`
	Jobs        []*DeploymentJob        `json:"jobs,omitempty"`
	StaticSites []*DeploymentStaticSite `json:"static_sites,omitempty"`
	Phase       string                  `json:"phase,omitempty"`
	Progress    *DeploymentProgress     `json:"progress,omitempty"`
	CreatedAt   string                  `json:"created_at"`
	UpdatedAt   string                  `json:"updated_at"`
	Cause       string                  `json:"cause,omitempty"`
}

// DeploymentService represents a deployed service
type DeploymentService struct {
	Name             string `json:"name"`
	SourceCommitHash string `json:"source_commit_hash,omitempty"`
}

// DeploymentWorker represents a deployed worker
type DeploymentWorker struct {
	Name             string `json:"name"`
	SourceCommitHash string `json:"source_commit_hash,omitempty"`
}

// DeploymentJob represents a deployed job
type DeploymentJob struct {
	Name             string `json:"name"`
	SourceCommitHash string `json:"source_commit_hash,omitempty"`
}

// DeploymentStaticSite represents a deployed static site
type DeploymentStaticSite struct {
	Name             string `json:"name"`
	SourceCommitHash string `json:"source_commit_hash,omitempty"`
}

// DeploymentProgress tracks deployment progress
type DeploymentProgress struct {
	PendingSteps int32                     `json:"pending_steps,omitempty"`
	RunningSteps int32                     `json:"running_steps,omitempty"`
	SuccessSteps int32                     `json:"success_steps,omitempty"`
	ErrorSteps   int32                     `json:"error_steps,omitempty"`
	TotalSteps   int32                     `json:"total_steps,omitempty"`
	Steps        []*DeploymentProgressStep `json:"steps,omitempty"`
}

// DeploymentProgressStep represents a single deployment step
type DeploymentProgressStep struct {
	Name      string                    `json:"name"`
	Status    string                    `json:"status"`
	Steps     []*DeploymentProgressStep `json:"steps,omitempty"`
	StartedAt string                    `json:"started_at,omitempty"`
	EndedAt   string                    `json:"ended_at,omitempty"`
}

// AppRegion represents the region where an app is deployed
type AppRegion struct {
	Slug        string   `json:"slug"`
	Label       string   `json:"label"`
	Flag        string   `json:"flag"`
	Continent   string   `json:"continent"`
	DataCenters []string `json:"data_centers,omitempty"`
}

type PendingDeployment struct {
	ID string `json:"id"`
}

// CreateAppRequest is the payload for creating an app
type CreateAppRequest struct {
	Spec *AppSpec `json:"spec"`
}

// UpdateAppRequest is the payload for updating an app
type UpdateAppRequest struct {
	Spec *AppSpec `json:"spec"`
}

// CreateApp creates a new App Platform application
func (c *Client) CreateApp(req CreateAppRequest) (*App, error) {
	url := fmt.Sprintf("%s/apps", c.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		App App `json:"app"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.App, nil
}

// GetApp retrieves an app by its ID
func (c *Client) GetApp(appID string) (*App, error) {
	url := fmt.Sprintf("%s/apps/%s", c.BaseURL, appID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		App App `json:"app"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.App, nil
}

// UpdateApp updates an existing app
func (c *Client) UpdateApp(appID string, req UpdateAppRequest) (*App, error) {
	url := fmt.Sprintf("%s/apps/%s", c.BaseURL, appID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		App App `json:"app"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.App, nil
}

// DeleteApp deletes an app by its ID
func (c *Client) DeleteApp(appID string) error {
	url := fmt.Sprintf("%s/apps/%s", c.BaseURL, appID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// GetDeployment retrieves a specific deployment for an app
func (c *Client) GetDeployment(appID, deploymentID string) (*Deployment, error) {
	url := fmt.Sprintf("%s/apps/%s/deployments/%s", c.BaseURL, appID, deploymentID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Deployment Deployment `json:"deployment"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &response.Deployment, nil
}

// ListApps retrieves all apps in the account
func (c *Client) ListApps() ([]App, error) {
	url := fmt.Sprintf("%s/apps?per_page=200", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Apps []App `json:"apps"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Apps, nil
}

// AppNodeMetadata stores metadata about an app for display in the UI
type AppNodeMetadata struct {
	AppID   string `json:"appId" mapstructure:"appId"`
	AppName string `json:"appName" mapstructure:"appName"`
}

// resolveAppMetadata fetches the app name from the API and stores it in metadata
func resolveAppMetadata(ctx core.SetupContext, appID string) error {
	// Handle expression placeholders
	if strings.Contains(appID, "{{") {
		return ctx.Metadata.Set(AppNodeMetadata{
			AppName: appID,
		})
	}

	// Check if already cached
	var existing AppNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.AppID == appID && existing.AppName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	app, err := client.GetApp(appID)
	if err != nil {
		return fmt.Errorf("failed to fetch app %q: %w", appID, err)
	}

	appName := appID
	if app.Spec != nil {
		appName = app.Spec.Name
	}

	return ctx.Metadata.Set(AppNodeMetadata{
		AppID:   appID,
		AppName: appName,
	})
}
