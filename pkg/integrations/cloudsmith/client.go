package cloudsmith

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.cloudsmith.io/v1"

// Client is a thin wrapper around the Cloudsmith v1 HTTP API.
type Client struct {
	APIKey  string
	http    core.HTTPContext
	BaseURL string
}

type APIError struct {
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.StatusCode, string(e.Body))
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error finding API key: %v", err)
	}

	return &Client{
		APIKey:  string(apiKey),
		http:    http,
		BaseURL: baseURL,
	}, nil
}

func (c *Client) execRequest(method, requestURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.APIKey)

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
		return nil, &APIError{
			StatusCode: res.StatusCode,
			Body:       responseBody,
		}
	}

	return responseBody, nil
}

type User struct {
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
}

// GetSelf validates the API key by fetching the authenticated user.
func (c *Client) GetSelf() (*User, error) {
	requestURL := fmt.Sprintf("%s/user/self/", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(responseBody, &user); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &user, nil
}

// Repository holds the Cloudsmith repository fields most useful for workflows.
type Repository struct {
	Name                      string `json:"name"`
	Slug                      string `json:"slug"`
	SlugPerm                  string `json:"slug_perm"`
	Namespace                 string `json:"namespace"`
	NamespaceURL              string `json:"namespace_url"`
	Description               string `json:"description"`
	RepositoryType            string `json:"repository_type_str"`
	ContentKind               string `json:"content_kind"`
	StorageRegion             string `json:"storage_region"`
	CDNUrl                    string `json:"cdn_url"`
	SelfURL                   string `json:"self_url"`
	SelfHTMLUrl               string `json:"self_html_url"`
	SelfWebappURL             string `json:"self_webapp_url"`
	IsPrivate                 bool   `json:"is_private"`
	IsPublic                  bool   `json:"is_public"`
	IsOpenSource              bool   `json:"is_open_source"`
	Size                      int64  `json:"size"`
	SizeStr                   string `json:"size_str"`
	PackageCount              int64  `json:"package_count"`
	PackageGroupCount         int64  `json:"package_group_count"`
	NumDownloads              int64  `json:"num_downloads"`
	NumQuarantinedPackages    int64  `json:"num_quarantined_packages"`
	NumPolicyViolatedPackages int64  `json:"num_policy_violated_packages"`
	CreatedAt                 string `json:"created_at"`
}

// GetRepository fetches a single repository identified by namespace (owner) and slug.
func (c *Client) GetRepository(owner, identifier string) (*Repository, error) {
	requestURL := fmt.Sprintf("%s/repos/%s/%s/", c.BaseURL, url.PathEscape(owner), url.PathEscape(identifier))
	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if err := json.Unmarshal(responseBody, &repository); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &repository, nil
}

const repositoryPageSize = 100

// ListRepositories returns every repository the authenticated user can access,
// following all pages until the API returns a page smaller than repositoryPageSize.
func (c *Client) ListRepositories() ([]Repository, error) {
	var all []Repository

	for page := 1; ; page++ {
		requestURL := fmt.Sprintf("%s/repos/?page=%d&page_size=%d", c.BaseURL, page, repositoryPageSize)
		responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}

		var repositories []Repository
		if err := json.Unmarshal(responseBody, &repositories); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		all = append(all, repositories...)

		if len(repositories) < repositoryPageSize {
			break
		}
	}

	return all, nil
}

// Package represents a Cloudsmith package with its metadata.
type Package struct {
	// Identity
	Slug        string `json:"slug"`
	SlugPerm    string `json:"slug_perm"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Version     string `json:"version"`
	Filename    string `json:"filename"`
	Format      string `json:"format"`
	Repository  string `json:"repository"`
	Namespace   string `json:"namespace"`
	UploadedAt  string `json:"uploaded_at"`
	Uploader    string `json:"uploader"`

	// Status
	Status          int    `json:"status"`
	StatusStr       string `json:"status_str"`
	StatusReason    string `json:"status_reason"`
	StatusUpdatedAt string `json:"status_updated_at"`

	// Stage / sync
	Stage            int    `json:"stage"`
	StageStr         string `json:"stage_str"`
	StageUpdatedAt   string `json:"stage_updated_at"`
	IsSyncAwaiting   bool   `json:"is_sync_awaiting"`
	IsSyncCompleted  bool   `json:"is_sync_completed"`
	IsSyncFailed     bool   `json:"is_sync_failed"`
	IsSyncInFlight   bool   `json:"is_sync_in_flight"`
	IsSyncInProgress bool   `json:"is_sync_in_progress"`
	SyncFinishedAt   string `json:"sync_finished_at"`
	SyncProgress     int    `json:"sync_progress"`

	// Quarantine / policy
	IsQuarantined  bool   `json:"is_quarantined"`
	PolicyViolated bool   `json:"policy_violated"`
	License        string `json:"license"`

	// Security scanning
	SecurityScanStatus      string `json:"security_scan_status"`
	SecurityScanStartedAt   string `json:"security_scan_started_at"`
	SecurityScanCompletedAt string `json:"security_scan_completed_at"`
	VulnerabilityResultsURL string `json:"vulnerability_scan_results_url"`

	// Checksums
	ChecksumMD5    string `json:"checksum_md5"`
	ChecksumSHA1   string `json:"checksum_sha1"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	ChecksumSHA512 string `json:"checksum_sha512"`

	// URLs
	SelfURL       string `json:"self_url"`
	SelfHTMLURL   string `json:"self_html_url"`
	SelfWebappURL string `json:"self_webapp_url"`
	CDNURL        string `json:"cdn_url"`

	// Size / metadata
	Size        int64  `json:"size"`
	SizeStr     string `json:"size_str"`
	Description string `json:"description"`
	Summary     string `json:"summary"`

	// Tags
	Tags          map[string]any `json:"tags"`
	TagsImmutable map[string]any `json:"tags_immutable"`
}

// PackageTagRequest is the request body for the Cloudsmith tag endpoint.
type PackageTagRequest struct {
	Action      string   `json:"action,omitempty"`
	IsImmutable bool     `json:"is_immutable,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func packageURL(baseURL, owner, repository, identifier string) string {
	return fmt.Sprintf(
		"%s/packages/%s/%s/%s/",
		baseURL,
		url.PathEscape(owner),
		url.PathEscape(repository),
		url.PathEscape(identifier),
	)
}

// GetPackage fetches a single package identified by namespace, repository slug, and package identifier.
func (c *Client) GetPackage(owner, repo, identifier string) (*Package, error) {
	responseBody, err := c.execRequest(http.MethodGet, packageURL(c.BaseURL, owner, repo, identifier), nil)
	if err != nil {
		return nil, err
	}

	var pkg Package
	if err := json.Unmarshal(responseBody, &pkg); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &pkg, nil
}

func (c *Client) ResyncPackage(owner, repository, identifier string) (*Package, error) {
	requestURL := packageURL(c.BaseURL, owner, repository, identifier) + "resync/"
	return c.execPackageRequest(http.MethodPost, requestURL, nil)
}

func (c *Client) TagPackage(owner, repository, identifier string, request PackageTagRequest) (*Package, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %v", err)
	}

	requestURL := packageURL(c.BaseURL, owner, repository, identifier) + "tag/"
	return c.execPackageRequest(http.MethodPost, requestURL, bytes.NewReader(body))
}

func (c *Client) DeletePackage(owner, repository, identifier string) error {
	_, err := c.execRequest(http.MethodDelete, packageURL(c.BaseURL, owner, repository, identifier), nil)
	return err
}

func (c *Client) execPackageRequest(method, requestURL string, body io.Reader) (*Package, error) {
	responseBody, err := c.execRequest(method, requestURL, body)
	if err != nil {
		return nil, err
	}

	if len(responseBody) == 0 {
		return nil, nil
	}

	var pkg Package
	if err := json.Unmarshal(responseBody, &pkg); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &pkg, nil
}

const packagePageSize = 100

// ListPackages returns all packages in the given repository, following pagination.
func (c *Client) ListPackages(owner, repo string) ([]Package, error) {
	var all []Package

	for page := 1; ; page++ {
		requestURL := fmt.Sprintf("%s/packages/%s/%s/?page=%d&page_size=%d", c.BaseURL, url.PathEscape(owner), url.PathEscape(repo), page, packagePageSize)
		responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}

		var packages []Package
		if err := json.Unmarshal(responseBody, &packages); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		all = append(all, packages...)

		if len(packages) < packagePageSize {
			break
		}
	}

	return all, nil
}

var errInvalidRepositoryID = errors.New("must be in the form 'owner/repository'")

// parseRepositoryID splits a Cloudsmith repository identifier of the form
// "owner/repository" into its namespace (owner) and slug (identifier) parts.
func parseRepositoryID(raw string) (owner string, identifier string, err error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", errInvalidRepositoryID
	}

	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return "", "", errInvalidRepositoryID
	}

	owner = strings.TrimSpace(parts[0])
	identifier = strings.TrimSpace(parts[1])
	if owner == "" || identifier == "" {
		return "", "", errInvalidRepositoryID
	}

	return owner, identifier, nil
}

// RepositoryNodeMetadata caches the human-readable repository name so the UI can
// display it without re-fetching from the API on every render.
type RepositoryNodeMetadata struct {
	RepositoryID        string `json:"repositoryId" mapstructure:"repositoryId"`
	RepositoryName      string `json:"repositoryName" mapstructure:"repositoryName"`
	RepositoryNamespace string `json:"repositoryNamespace" mapstructure:"repositoryNamespace"`
	RepositorySlug      string `json:"repositorySlug" mapstructure:"repositorySlug"`
}

// PackageNodeMetadata caches both the repository and package display names so the
// UI can show human-readable labels without extra API calls.
type PackageNodeMetadata struct {
	RepositoryID        string `json:"repositoryId" mapstructure:"repositoryId"`
	RepositoryName      string `json:"repositoryName" mapstructure:"repositoryName"`
	RepositoryNamespace string `json:"repositoryNamespace" mapstructure:"repositoryNamespace"`
	RepositorySlug      string `json:"repositorySlug" mapstructure:"repositorySlug"`
	PackageID           string `json:"packageId" mapstructure:"packageId"`
	PackageName         string `json:"packageName" mapstructure:"packageName"`
}

// resolveRepositoryMetadata stores display metadata for the selected repository.
// Expressions are stored verbatim because they can only be resolved at execution time.
func resolveRepositoryMetadata(ctx core.SetupContext, repositoryID string) error {
	if strings.Contains(repositoryID, "{{") {
		return ctx.Metadata.Set(RepositoryNodeMetadata{
			RepositoryID:   repositoryID,
			RepositoryName: repositoryID,
		})
	}

	owner, identifier, err := parseRepositoryID(repositoryID)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", repositoryID, err)
	}

	var existing RepositoryNodeMetadata
	decodeErr := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if decodeErr == nil && existing.RepositoryID == repositoryID && existing.RepositoryName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	repository, err := client.GetRepository(owner, identifier)
	if err != nil {
		return fmt.Errorf("failed to fetch repository %q: %w", repositoryID, err)
	}

	name := repository.Name
	if name == "" {
		name = identifier
	}

	return ctx.Metadata.Set(RepositoryNodeMetadata{
		RepositoryID:        repositoryID,
		RepositoryName:      name,
		RepositoryNamespace: owner,
		RepositorySlug:      identifier,
	})
}

// resolvePackageMetadata fetches and caches both the repository and package names
// for display in the canvas node. Expressions are stored verbatim.
func resolvePackageMetadata(ctx core.SetupContext, repositoryID, packageSlugPerm string) error {
	repoIsExpr := strings.Contains(repositoryID, "{{")
	pkgIsExpr := strings.Contains(packageSlugPerm, "{{")

	// If the repository itself is an expression we cannot resolve either field at setup time.
	if repoIsExpr {
		return ctx.Metadata.Set(PackageNodeMetadata{
			RepositoryID:   repositoryID,
			RepositoryName: repositoryID,
			PackageID:      packageSlugPerm,
			PackageName:    packageSlugPerm,
		})
	}

	owner, identifier, err := parseRepositoryID(repositoryID)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", repositoryID, err)
	}

	// Skip API calls when cached metadata matches and the stored name was already
	// computed from pkg.Name+version (i.e. differs from the raw slug_perm value).
	// A stored name equal to packageSlugPerm means the old slug format was cached.
	var existing PackageNodeMetadata
	if decodeErr := mapstructure.Decode(ctx.Metadata.Get(), &existing); decodeErr == nil &&
		existing.RepositoryID == repositoryID &&
		existing.PackageID == packageSlugPerm &&
		existing.PackageName != "" &&
		existing.PackageName != packageSlugPerm {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	repository, err := client.GetRepository(owner, identifier)
	if err != nil {
		return fmt.Errorf("failed to fetch repository %q: %w", repositoryID, err)
	}

	repoName := repository.Name
	if repoName == "" {
		repoName = identifier
	}

	// Package is an expression — store the resolved repository name but leave
	// the package fields verbatim until the expression is evaluated at runtime.
	if pkgIsExpr {
		return ctx.Metadata.Set(PackageNodeMetadata{
			RepositoryID:        repositoryID,
			RepositoryName:      repoName,
			RepositoryNamespace: owner,
			RepositorySlug:      identifier,
			PackageID:           packageSlugPerm,
			PackageName:         packageSlugPerm,
		})
	}

	pkg, err := client.GetPackage(owner, identifier, packageSlugPerm)
	if err != nil {
		return fmt.Errorf("failed to fetch package %q: %w", packageSlugPerm, err)
	}

	packageName := pkg.Name
	if packageName == "" {
		packageName = packageSlugPerm
	}
	if pkg.Version != "" {
		packageName = fmt.Sprintf("%s %s", packageName, pkg.Version)
	}

	return ctx.Metadata.Set(PackageNodeMetadata{
		RepositoryID:        repositoryID,
		RepositoryName:      repoName,
		RepositoryNamespace: owner,
		RepositorySlug:      identifier,
		PackageID:           packageSlugPerm,
		PackageName:         packageName,
	})
}

// VulnerabilityScan is a package's security-scan summary, delivered under
// context.vulnerability_scan_results on a package.security_scanned webhook.
type VulnerabilityScan struct {
	Identifier         string `json:"identifier"`
	CreatedAt          string `json:"created_at"`
	HasVulnerabilities bool   `json:"has_vulnerabilities"`
	MaxSeverity        string `json:"max_severity"`
	NumVulnerabilities int    `json:"num_vulnerabilities"`
}

// Webhook is the subset of a Cloudsmith webhook this integration manages.
type Webhook struct {
	SlugPerm  string   `json:"slug_perm"`
	TargetURL string   `json:"target_url"`
	Events    []string `json:"events"`
	IsActive  bool     `json:"is_active"`
}

// requestBodyFormatJSONObject is the Cloudsmith webhook payload format that
// delivers the package as a JSON object body (application/json).
const requestBodyFormatJSONObject = 0

// webhookPayload builds the create/update request body for a managed webhook:
// the desired target URL, subscribed events, active flag, and (when set) the
// signing key. Cloudsmith treats PATCH the same as POST for these fields, so the
// two operations share one body.
func webhookPayload(targetURL, signatureKey string, events []string) ([]byte, error) {
	templates := make([]map[string]string, 0, len(events))
	for _, event := range events {
		templates = append(templates, map[string]string{"event": event, "template": ""})
	}

	body := map[string]any{
		"target_url":          targetURL,
		"events":              events,
		"request_body_format": requestBodyFormatJSONObject,
		"templates":           templates,
		"is_active":           true,
	}
	if signatureKey != "" {
		body["signature_key"] = signatureKey
	}

	return json.Marshal(body)
}

// CreateWebhook registers a webhook on a repository (owner/repository) that
// posts the given events to targetURL as a JSON object. When signatureKey is
// non-empty, Cloudsmith signs each delivery with HMAC-SHA1 of the body using
// that key (sent in the X-Cloudsmith-Signature header).
func (c *Client) CreateWebhook(owner, repository, targetURL, signatureKey string, events []string) (*Webhook, error) {
	payload, err := webhookPayload(targetURL, signatureKey, events)
	if err != nil {
		return nil, fmt.Errorf("error encoding webhook: %v", err)
	}

	requestURL := fmt.Sprintf("%s/webhooks/%s/%s/", c.BaseURL, url.PathEscape(owner), url.PathEscape(repository))
	responseBody, err := c.execRequest(http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := json.Unmarshal(responseBody, &webhook); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &webhook, nil
}

// GetWebhook fetches a webhook by its permanent slug. The error (including
// not-found) lets callers detect a webhook that was removed at Cloudsmith.
func (c *Client) GetWebhook(owner, repository, slugPerm string) (*Webhook, error) {
	requestURL := fmt.Sprintf("%s/webhooks/%s/%s/%s/",
		c.BaseURL, url.PathEscape(owner), url.PathEscape(repository), url.PathEscape(slugPerm))
	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := json.Unmarshal(responseBody, &webhook); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &webhook, nil
}

// UpdateWebhook re-asserts an existing webhook's desired state (PATCH): target
// URL, subscribed events, active flag, and signing key. A webhook can be
// disabled or have its events edited out of band at Cloudsmith, and the
// signature_key is only set on write and never returned — re-applying all of
// these on reconcile keeps deliveries flowing and verifiable without recreating
// the webhook.
func (c *Client) UpdateWebhook(owner, repository, slugPerm, targetURL, signatureKey string, events []string) (*Webhook, error) {
	payload, err := webhookPayload(targetURL, signatureKey, events)
	if err != nil {
		return nil, fmt.Errorf("error encoding webhook: %v", err)
	}

	requestURL := fmt.Sprintf("%s/webhooks/%s/%s/%s/",
		c.BaseURL, url.PathEscape(owner), url.PathEscape(repository), url.PathEscape(slugPerm))
	responseBody, err := c.execRequest(http.MethodPatch, requestURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := json.Unmarshal(responseBody, &webhook); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &webhook, nil
}

// DeleteWebhook removes a webhook from a repository by its permanent slug.
func (c *Client) DeleteWebhook(owner, repository, slugPerm string) error {
	requestURL := fmt.Sprintf("%s/webhooks/%s/%s/%s/",
		c.BaseURL, url.PathEscape(owner), url.PathEscape(repository), url.PathEscape(slugPerm))
	_, err := c.execRequest(http.MethodDelete, requestURL, nil)
	return err
}

// Organization is the subset of a Cloudsmith organization used to populate the
// organization resource selector.
type Organization struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	SlugPerm string `json:"slug_perm"`
}

const organizationPageSize = 100

// ListOrganizations returns every organization the authenticated user belongs to,
// following all pages until the API returns a page smaller than organizationPageSize.
func (c *Client) ListOrganizations() ([]Organization, error) {
	var all []Organization

	for page := 1; ; page++ {
		requestURL := fmt.Sprintf("%s/orgs/?page=%d&page_size=%d", c.BaseURL, page, organizationPageSize)
		responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}

		var organizations []Organization
		if err := json.Unmarshal(responseBody, &organizations); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		all = append(all, organizations...)

		if len(organizations) < organizationPageSize {
			break
		}
	}

	return all, nil
}

// GetOrganization fetches a single organization by its slug.
func (c *Client) GetOrganization(org string) (*Organization, error) {
	requestURL := fmt.Sprintf("%s/orgs/%s/", c.BaseURL, url.PathEscape(org))
	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var organization Organization
	if err := json.Unmarshal(responseBody, &organization); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &organization, nil
}

// VulnerabilityPolicy is an organization-scoped package vulnerability policy.
type VulnerabilityPolicy struct {
	Name                  string `json:"name"`
	Description           string `json:"description"`
	MinSeverity           string `json:"min_severity"`
	PackageQueryString    string `json:"package_query_string"`
	OnViolationQuarantine bool   `json:"on_violation_quarantine"`
	AllowUnknownSeverity  bool   `json:"allow_unknown_severity"`
	SlugPerm              string `json:"slug_perm"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

// VulnerabilityPolicyRequest is the create/update body for a vulnerability policy.
// Only name is required; min_severity defaults to "Critical" at Cloudsmith.
type VulnerabilityPolicyRequest struct {
	Name                  string `json:"name"`
	Description           string `json:"description,omitempty"`
	MinSeverity           string `json:"min_severity,omitempty"`
	PackageQueryString    string `json:"package_query_string,omitempty"`
	OnViolationQuarantine bool   `json:"on_violation_quarantine"`
	AllowUnknownSeverity  bool   `json:"allow_unknown_severity"`
}

// CreateVulnerabilityPolicy creates a new vulnerability policy in an organization.
func (c *Client) CreateVulnerabilityPolicy(org string, req VulnerabilityPolicyRequest) (*VulnerabilityPolicy, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %v", err)
	}

	requestURL := fmt.Sprintf("%s/orgs/%s/vulnerability-policy/", c.BaseURL, url.PathEscape(org))
	responseBody, err := c.execRequest(http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var policy VulnerabilityPolicy
	if err := json.Unmarshal(responseBody, &policy); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &policy, nil
}

// GetVulnerabilityPolicy fetches a single vulnerability policy by its permanent slug.
func (c *Client) GetVulnerabilityPolicy(org, slugPerm string) (*VulnerabilityPolicy, error) {
	requestURL := fmt.Sprintf("%s/orgs/%s/vulnerability-policy/%s/",
		c.BaseURL, url.PathEscape(org), url.PathEscape(slugPerm))
	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var policy VulnerabilityPolicy
	if err := json.Unmarshal(responseBody, &policy); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &policy, nil
}

// DeleteVulnerabilityPolicy removes a vulnerability policy by its permanent slug.
func (c *Client) DeleteVulnerabilityPolicy(org, slugPerm string) error {
	requestURL := fmt.Sprintf("%s/orgs/%s/vulnerability-policy/%s/",
		c.BaseURL, url.PathEscape(org), url.PathEscape(slugPerm))
	_, err := c.execRequest(http.MethodDelete, requestURL, nil)
	return err
}

const vulnerabilityPolicyPageSize = 100

// ListVulnerabilityPolicies returns all vulnerability policies in an organization,
// following pagination.
func (c *Client) ListVulnerabilityPolicies(org string) ([]VulnerabilityPolicy, error) {
	var all []VulnerabilityPolicy

	for page := 1; ; page++ {
		requestURL := fmt.Sprintf("%s/orgs/%s/vulnerability-policy/?page=%d&page_size=%d",
			c.BaseURL, url.PathEscape(org), page, vulnerabilityPolicyPageSize)
		responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}

		var policies []VulnerabilityPolicy
		if err := json.Unmarshal(responseBody, &policies); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		all = append(all, policies...)

		if len(policies) < vulnerabilityPolicyPageSize {
			break
		}
	}

	return all, nil
}

// VulnerabilityPolicyNodeMetadata caches human-readable organization and policy
// names so the UI can display them without re-fetching on every render.
type VulnerabilityPolicyNodeMetadata struct {
	OrganizationSlug string `json:"organizationSlug" mapstructure:"organizationSlug"`
	OrganizationName string `json:"organizationName" mapstructure:"organizationName"`
	PolicyID         string `json:"policyId" mapstructure:"policyId"`
	PolicyName       string `json:"policyName" mapstructure:"policyName"`
}

// resolveOrganizationMetadata stores display metadata for the selected organization.
// Expressions are stored verbatim because they can only be resolved at execution time.
func resolveOrganizationMetadata(ctx core.SetupContext, org string) error {
	if strings.Contains(org, "{{") {
		return ctx.Metadata.Set(VulnerabilityPolicyNodeMetadata{
			OrganizationSlug: org,
			OrganizationName: org,
		})
	}

	var existing VulnerabilityPolicyNodeMetadata
	if decodeErr := mapstructure.Decode(ctx.Metadata.Get(), &existing); decodeErr == nil &&
		existing.OrganizationSlug == org && existing.OrganizationName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	organization, err := client.GetOrganization(org)
	if err != nil {
		return fmt.Errorf("failed to fetch organization %q: %w", org, err)
	}

	name := organization.Name
	if name == "" {
		name = org
	}

	return ctx.Metadata.Set(VulnerabilityPolicyNodeMetadata{
		OrganizationSlug: org,
		OrganizationName: name,
	})
}

// resolvePolicyMetadata stores display metadata for the selected organization and
// policy. Expressions are stored verbatim until evaluated at runtime.
func resolvePolicyMetadata(ctx core.SetupContext, org, policySlugPerm string) error {
	orgIsExpr := strings.Contains(org, "{{")
	policyIsExpr := strings.Contains(policySlugPerm, "{{")

	if orgIsExpr {
		return ctx.Metadata.Set(VulnerabilityPolicyNodeMetadata{
			OrganizationSlug: org,
			OrganizationName: org,
			PolicyID:         policySlugPerm,
			PolicyName:       policySlugPerm,
		})
	}

	var existing VulnerabilityPolicyNodeMetadata
	if decodeErr := mapstructure.Decode(ctx.Metadata.Get(), &existing); decodeErr == nil &&
		existing.OrganizationSlug == org &&
		existing.PolicyID == policySlugPerm &&
		existing.PolicyName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	organization, err := client.GetOrganization(org)
	if err != nil {
		return fmt.Errorf("failed to fetch organization %q: %w", org, err)
	}

	orgName := organization.Name
	if orgName == "" {
		orgName = org
	}

	if policyIsExpr {
		return ctx.Metadata.Set(VulnerabilityPolicyNodeMetadata{
			OrganizationSlug: org,
			OrganizationName: orgName,
			PolicyID:         policySlugPerm,
			PolicyName:       policySlugPerm,
		})
	}

	policy, err := client.GetVulnerabilityPolicy(org, policySlugPerm)
	if err != nil {
		return fmt.Errorf("failed to fetch vulnerability policy %q: %w", policySlugPerm, err)
	}

	policyName := policy.Name
	if policyName == "" {
		policyName = policySlugPerm
	}

	return ctx.Metadata.Set(VulnerabilityPolicyNodeMetadata{
		OrganizationSlug: org,
		OrganizationName: orgName,
		PolicyID:         policySlugPerm,
		PolicyName:       policyName,
	})
}
