package cloudsmith

import (
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
	IsQuarantined  bool `json:"is_quarantined"`
	PolicyViolated bool `json:"policy_violated"`

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

// GetPackage fetches a single package identified by namespace, repository slug, and package identifier.
func (c *Client) GetPackage(owner, repo, identifier string) (*Package, error) {
	requestURL := fmt.Sprintf("%s/packages/%s/%s/%s/", c.BaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(identifier))
	responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
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

	var existing PackageNodeMetadata
	if decodeErr := mapstructure.Decode(ctx.Metadata.Get(), &existing); decodeErr == nil &&
		existing.RepositoryID == repositoryID &&
		existing.PackageID == packageSlugPerm &&
		existing.PackageName != "" {
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
