package oci

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	coreServicesHostTemplate = "iaas.%s.oraclecloud.com"
	identityHostTemplate     = "identity.%s.oraclecloud.com"
	coreServicesAPIVersion   = "20160918"
)

// Client is an OCI REST API client that signs requests using OCI API Key authentication.
type Client struct {
	tenancyOCID string
	userOCID    string
	fingerprint string
	privateKey  *rsa.PrivateKey
	region      string
	http        core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	tenancyOCID, err := integration.GetConfig("tenancyOcid")
	if err != nil {
		return nil, fmt.Errorf("failed to get tenancyOcid: %w", err)
	}
	userOCID, err := integration.GetConfig("userOcid")
	if err != nil {
		return nil, fmt.Errorf("failed to get userOcid: %w", err)
	}
	fingerprint, err := integration.GetConfig("fingerprint")
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprint: %w", err)
	}
	privateKeyPEM, err := integration.GetConfig("privateKey")
	if err != nil {
		return nil, fmt.Errorf("failed to get privateKey: %w", err)
	}
	region, err := integration.GetConfig("region")
	if err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}

	privateKey, err := parsePrivateKey(string(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &Client{
		tenancyOCID: string(tenancyOCID),
		userOCID:    string(userOCID),
		fingerprint: string(fingerprint),
		privateKey:  privateKey,
		region:      string(region),
		http:        httpCtx,
	}, nil
}

// ValidateCredentials validates the OCI credentials by fetching the current user.
func (c *Client) ValidateCredentials() error {
	host := fmt.Sprintf(identityHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/20160918/users/%s", host, c.userOCID)
	_, err := c.doRequest(http.MethodGet, host, url, nil)
	return err
}

// LaunchInstance starts a new OCI Compute instance.
func (c *Client) LaunchInstance(req LaunchInstanceRequest) (*Instance, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/instances", host, coreServicesAPIVersion)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal launch instance request: %w", err)
	}

	respBody, err := c.doRequest(http.MethodPost, host, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var instance Instance
	if err := json.Unmarshal(respBody, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance response: %w", err)
	}

	return &instance, nil
}

// GetInstance retrieves a Compute instance by OCID.
func (c *Client) GetInstance(instanceID string) (*Instance, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/instances/%s", host, coreServicesAPIVersion, instanceID)

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var instance Instance
	if err := json.Unmarshal(respBody, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance response: %w", err)
	}

	return &instance, nil
}

// ListInstances lists Compute instances in a compartment.
func (c *Client) ListInstances(compartmentID string) ([]Instance, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf(
		"https://%s/%s/instances?compartmentId=%s&limit=100",
		host,
		coreServicesAPIVersion,
		neturl.QueryEscape(compartmentID),
	)

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var instances []Instance
	if err := json.Unmarshal(respBody, &instances); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instances: %w", err)
	}

	return instances, nil
}

// ListVNICAttachments lists VNIC attachments for an instance, used to find IP addresses.
func (c *Client) ListVNICAttachments(compartmentID, instanceID string) ([]VNICAttachment, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/vnicAttachments?compartmentId=%s&instanceId=%s",
		host, coreServicesAPIVersion, neturl.QueryEscape(compartmentID), neturl.QueryEscape(instanceID))

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var attachments []VNICAttachment
	if err := json.Unmarshal(respBody, &attachments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VNIC attachments: %w", err)
	}

	return attachments, nil
}

// GetVNIC retrieves a VNIC by OCID to obtain IP addresses.
func (c *Client) GetVNIC(vnicID string) (*VNIC, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/vnics/%s", host, coreServicesAPIVersion, vnicID)

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var vnic VNIC
	if err := json.Unmarshal(respBody, &vnic); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VNIC: %w", err)
	}

	return &vnic, nil
}

func (c *Client) ListCompartments() ([]Compartment, error) {
	host := fmt.Sprintf(identityHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/compartments?compartmentId=%s&limit=1000", host, coreServicesAPIVersion, neturl.QueryEscape(c.tenancyOCID))

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var compartments []Compartment
	if err := json.Unmarshal(respBody, &compartments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compartments: %w", err)
	}

	return compartments, nil
}

func (c *Client) ListAvailabilityDomains(compartmentID string) ([]AvailabilityDomain, error) {
	host := fmt.Sprintf(identityHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/availabilityDomains?compartmentId=%s", host, coreServicesAPIVersion, neturl.QueryEscape(compartmentID))

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var ads []AvailabilityDomain
	if err := json.Unmarshal(respBody, &ads); err != nil {
		return nil, fmt.Errorf("failed to unmarshal availability domains: %w", err)
	}

	return ads, nil
}

func (c *Client) ListShapes(compartmentID string) ([]Shape, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/shapes?compartmentId=%s&limit=100", host, coreServicesAPIVersion, neturl.QueryEscape(compartmentID))

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var shapes []Shape
	if err := json.Unmarshal(respBody, &shapes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shapes: %w", err)
	}

	return shapes, nil
}

func (c *Client) ListImages(compartmentID, operatingSystem string) ([]Image, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/images?compartmentId=%s&limit=100&sortBy=DISPLAYNAME&sortOrder=ASC",
		host, coreServicesAPIVersion, neturl.QueryEscape(compartmentID))
	if operatingSystem != "" {
		url += "&operatingSystem=" + neturl.QueryEscape(operatingSystem)
	}

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var images []Image
	if err := json.Unmarshal(respBody, &images); err != nil {
		return nil, fmt.Errorf("failed to unmarshal images: %w", err)
	}

	return images, nil
}

func (c *Client) ListSubnets(compartmentID string) ([]Subnet, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/subnets?compartmentId=%s&limit=100", host, coreServicesAPIVersion, neturl.QueryEscape(compartmentID))

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var subnets []Subnet
	if err := json.Unmarshal(respBody, &subnets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subnets: %w", err)
	}

	return subnets, nil
}

func (c *Client) ListBlockVolumes(compartmentID string) ([]BlockVolume, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/volumes?compartmentId=%s&limit=100", host, coreServicesAPIVersion, neturl.QueryEscape(compartmentID))

	respBody, err := c.doRequest(http.MethodGet, host, url, nil)
	if err != nil {
		return nil, err
	}

	var volumes []BlockVolume
	if err := json.Unmarshal(respBody, &volumes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block volumes: %w", err)
	}

	return volumes, nil
}

// doRequest signs and executes an HTTP request against the OCI API.
func (c *Client) doRequest(method, host, url string, body io.Reader) ([]byte, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("Host", host)

	if len(bodyBytes) > 0 {
		hash := sha256.Sum256(bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
		req.Header.Set("x-content-sha256", base64.StdEncoding.EncodeToString(hash[:]))
	}

	if err := c.signRequest(req, len(bodyBytes) > 0); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OCI API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// signRequest adds the OCI HTTP Signature Authorization header.
// See: https://docs.oracle.com/en-us/iaas/Content/API/Concepts/signingrequests.htm
func (c *Client) signRequest(req *http.Request, hasBody bool) error {
	var headerNames []string
	if hasBody {
		headerNames = []string{"date", "(request-target)", "host", "content-length", "content-type", "x-content-sha256"}
	} else {
		headerNames = []string{"date", "(request-target)", "host"}
	}

	signingString := c.buildSigningString(req, headerNames)

	h := sha256.New()
	h.Write([]byte(signingString))
	digest := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, digest)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	encodedSig := base64.StdEncoding.EncodeToString(signature)
	keyID := fmt.Sprintf("%s/%s/%s", c.tenancyOCID, c.userOCID, c.fingerprint)
	authHeader := fmt.Sprintf(
		`Signature version="1",keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`,
		keyID,
		strings.Join(headerNames, " "),
		encodedSig,
	)

	req.Header.Set("Authorization", authHeader)
	return nil
}

func (c *Client) buildSigningString(req *http.Request, headerNames []string) string {
	var parts []string
	for _, name := range headerNames {
		switch name {
		case "(request-target)":
			target := strings.ToLower(req.Method) + " " + req.URL.RequestURI()
			parts = append(parts, "(request-target): "+target)
		default:
			parts = append(parts, name+": "+req.Header.Get(name))
		}
	}
	return strings.Join(parts, "\n")
}

func parsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}

	// Try PKCS8 first, then PKCS1.
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("parsed key is not an RSA private key")
		}
		return rsaKey, nil
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key (tried PKCS8 and PKCS1): %w", err)
	}

	return rsaKey, nil
}

// OCI API types

type LaunchInstanceRequest struct {
	CompartmentID               string                       `json:"compartmentId"`
	AvailabilityDomain          string                       `json:"availabilityDomain"`
	DisplayName                 string                       `json:"displayName,omitempty"`
	Shape                       string                       `json:"shape"`
	SourceDetails               InstanceSourceDetails        `json:"sourceDetails"`
	CreateVnicDetails           *CreateVnicDetails           `json:"createVnicDetails,omitempty"`
	Metadata                    map[string]string            `json:"metadata,omitempty"`
	ShapeConfig                 *InstanceShapeConfig         `json:"shapeConfig,omitempty"`
	ShieldedInstanceConfig      *ShieldedInstanceConfig      `json:"shieldedInstanceConfig,omitempty"`
	ConfidentialInstanceOptions *ConfidentialInstanceOptions `json:"confidentialInstanceOptions,omitempty"`
}

type InstanceSourceDetails struct {
	SourceType          string   `json:"sourceType"`
	ImageID             string   `json:"imageId"`
	BootVolumeSizeInGBs *float64 `json:"bootVolumeSizeInGBs,omitempty"`
	BootVolumeVpusPerGB *float64 `json:"bootVolumeVpusPerGB,omitempty"`
}

type CreateVnicDetails struct {
	SubnetID string `json:"subnetId"`
}

type InstanceShapeConfig struct {
	OCPUs       *float64 `json:"ocpus,omitempty"`
	MemoryInGBs *float64 `json:"memoryInGBs,omitempty"`
}

type UpdateInstanceRequest struct {
	DisplayName *string              `json:"displayName,omitempty"`
	ShapeConfig *InstanceShapeConfig `json:"shapeConfig,omitempty"`
}

type ShieldedInstanceConfig struct {
	IsSecureBootEnabled            bool `json:"isSecureBootEnabled"`
	IsMeasuredBootEnabled          bool `json:"isMeasuredBootEnabled"`
	IsTrustedPlatformModuleEnabled bool `json:"isTrustedPlatformModuleEnabled"`
}

type ConfidentialInstanceOptions struct {
	IsEnabled bool `json:"isEnabled"`
}

type Instance struct {
	ID                 string `json:"id"`
	DisplayName        string `json:"displayName"`
	LifecycleState     string `json:"lifecycleState"`
	Shape              string `json:"shape"`
	AvailabilityDomain string `json:"availabilityDomain"`
	CompartmentID      string `json:"compartmentId"`
	Region             string `json:"region"`
	TimeCreated        string `json:"timeCreated"`
}

type VNICAttachment struct {
	VNICID         string `json:"vnicId"`
	SubnetID       string `json:"subnetId"`
	LifecycleState string `json:"lifecycleState"`
}

type VNIC struct {
	ID        string `json:"id"`
	PublicIP  string `json:"publicIp"`
	PrivateIP string `json:"privateIp"`
}

type Compartment struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	LifecycleState string `json:"lifecycleState"`
}

type AvailabilityDomain struct {
	Name          string `json:"name"`
	CompartmentID string `json:"compartmentId"`
}

type Shape struct {
	Shape string `json:"shape"`
}

type Image struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	LifecycleState string `json:"lifecycleState"`
}

type Subnet struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	CIDRBlock      string `json:"cidrBlock"`
	LifecycleState string `json:"lifecycleState"`
}

type BlockVolume struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	LifecycleState string `json:"lifecycleState"`
	SizeInGBs      int    `json:"sizeInGBs"`
}

type VolumeAttachment struct {
	ID             string `json:"id"`
	InstanceID     string `json:"instanceId"`
	VolumeID       string `json:"volumeId"`
	LifecycleState string `json:"lifecycleState"`
	Device         string `json:"device"`
}

// AttachVolume attaches an existing block volume to a running instance using paravirtualised attachment.
func (c *Client) AttachVolume(instanceID, volumeID string) (*VolumeAttachment, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/volumeAttachments", host, coreServicesAPIVersion)

	body := map[string]any{
		"instanceId": instanceID,
		"volumeId":   volumeID,
		"type":       "paravirtualized",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attach volume request: %w", err)
	}

	respBody, err := c.doRequest(http.MethodPost, host, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var attachment VolumeAttachment
	if err := json.Unmarshal(respBody, &attachment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal volume attachment response: %w", err)
	}

	return &attachment, nil
}

// UpdateInstance updates mutable Compute instance attributes.
func (c *Client) UpdateInstance(instanceID string, req UpdateInstanceRequest) (*Instance, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/instances/%s", host, coreServicesAPIVersion, instanceID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update instance request: %w", err)
	}

	respBody, err := c.doRequest(http.MethodPut, host, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var instance Instance
	if err := json.Unmarshal(respBody, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance response: %w", err)
	}

	return &instance, nil
}

// InstanceAction performs a power lifecycle action on a Compute instance.
func (c *Client) InstanceAction(instanceID, action string) (*Instance, error) {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf("https://%s/%s/instances/%s?action=%s", host, coreServicesAPIVersion, instanceID, neturl.QueryEscape(action))

	respBody, err := c.doRequest(http.MethodPost, host, url, nil)
	if err != nil {
		return nil, err
	}

	var instance Instance
	if err := json.Unmarshal(respBody, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance response: %w", err)
	}

	return &instance, nil
}

// TerminateInstance terminates a Compute instance, optionally preserving the boot volume.
func (c *Client) TerminateInstance(instanceID string, preserveBootVolume bool) error {
	host := fmt.Sprintf(coreServicesHostTemplate, c.region)
	url := fmt.Sprintf(
		"https://%s/%s/instances/%s?preserveBootVolume=%t",
		host,
		coreServicesAPIVersion,
		instanceID,
		preserveBootVolume,
	)

	_, err := c.doRequest(http.MethodDelete, host, url, nil)
	return err
}

// ONS (Notifications) types

type ONSSubscription struct {
	ID             string `json:"id"`
	TopicID        string `json:"topicId"`
	Protocol       string `json:"protocol"`
	Endpoint       string `json:"endpoint"`
	LifecycleState string `json:"lifecycleState"`
}

// CreateONSSubscription creates an HTTPS subscription on an OCI Notifications topic.
func (c *Client) CreateONSSubscription(compartmentID, topicID, endpoint string) (*ONSSubscription, error) {
	host := fmt.Sprintf("notification.%s.oraclecloud.com", c.region)
	url := fmt.Sprintf("https://%s/20181201/subscriptions", host)

	body := map[string]string{
		"compartmentId": compartmentID,
		"topicId":       topicID,
		"protocol":      "CUSTOM_HTTPS",
		"endpoint":      endpoint,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription request: %w", err)
	}

	respBody, err := c.doRequest(http.MethodPost, host, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var sub ONSSubscription
	if err := json.Unmarshal(respBody, &sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ONS subscription response: %w (body: %s)", err, string(respBody))
	}

	if sub.ID == "" {
		return nil, fmt.Errorf("OCI returned an unexpected subscription response with no OCID (body: %s)", string(respBody))
	}

	return &sub, nil
}

// DeleteONSSubscription deletes an OCI Notifications subscription by its OCID.
func (c *Client) DeleteONSSubscription(subscriptionID string) error {
	host := fmt.Sprintf("notification.%s.oraclecloud.com", c.region)
	url := fmt.Sprintf("https://%s/20181201/subscriptions/%s", host, subscriptionID)

	_, err := c.doRequest(http.MethodDelete, host, url, nil)
	return err
}

// ONS topic types and methods

type ONSTopic struct {
	TopicID        string `json:"topicId"`
	Name           string `json:"name"`
	LifecycleState string `json:"lifecycleState"`
}

// CreateONSTopic creates a new OCI Notifications topic.
func (c *Client) CreateONSTopic(compartmentID, name string) (*ONSTopic, error) {
	host := fmt.Sprintf("notification.%s.oraclecloud.com", c.region)
	url := fmt.Sprintf("https://%s/20181201/topics", host)

	body := map[string]string{
		"compartmentId": compartmentID,
		"name":          name,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal topic request: %w", err)
	}

	respBody, err := c.doRequest(http.MethodPost, host, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var topic ONSTopic
	if err := json.Unmarshal(respBody, &topic); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ONS topic: %w", err)
	}

	return &topic, nil
}

// DeleteONSTopic deletes an OCI Notifications topic by its OCID.
// Deleting the topic also removes all its subscriptions.
func (c *Client) DeleteONSTopic(topicID string) error {
	host := fmt.Sprintf("notification.%s.oraclecloud.com", c.region)
	url := fmt.Sprintf("https://%s/20181201/topics/%s", host, topicID)

	_, err := c.doRequest(http.MethodDelete, host, url, nil)
	return err
}

// OCI Events rule types and methods

type EventsRule struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	LifecycleState string `json:"lifecycleState"`
}

type eventsRuleAction struct {
	ActionType string `json:"actionType"`
	TopicID    string `json:"topicId"`
	IsEnabled  bool   `json:"isEnabled"`
}

type eventsRuleActions struct {
	Actions []eventsRuleAction `json:"actions"`
}

type createEventsRuleRequest struct {
	CompartmentID string            `json:"compartmentId"`
	DisplayName   string            `json:"displayName"`
	Description   string            `json:"description"`
	IsEnabled     bool              `json:"isEnabled"`
	Condition     string            `json:"condition"`
	Actions       eventsRuleActions `json:"actions"`
}

// CreateEventsRule creates an OCI Events rule that routes matching events to an ONS topic.
func (c *Client) CreateEventsRule(compartmentID, displayName, condition, topicID string) (*EventsRule, error) {
	host := fmt.Sprintf("events.%s.oraclecloud.com", c.region)
	url := fmt.Sprintf("https://%s/20181201/rules", host)

	req := createEventsRuleRequest{
		CompartmentID: compartmentID,
		DisplayName:   displayName,
		Description:   "Managed by SuperPlane",
		IsEnabled:     true,
		Condition:     condition,
		Actions: eventsRuleActions{
			Actions: []eventsRuleAction{
				{ActionType: "ONS", TopicID: topicID, IsEnabled: true},
			},
		},
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal events rule request: %w", err)
	}

	respBody, err := c.doRequest(http.MethodPost, host, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var rule EventsRule
	if err := json.Unmarshal(respBody, &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events rule: %w", err)
	}

	return &rule, nil
}

// DeleteEventsRule deletes an OCI Events rule by its OCID.
func (c *Client) DeleteEventsRule(ruleID string) error {
	host := fmt.Sprintf("events.%s.oraclecloud.com", c.region)
	url := fmt.Sprintf("https://%s/20181201/rules/%s", host, ruleID)

	_, err := c.doRequest(http.MethodDelete, host, url, nil)
	return err
}
