package codeartifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type PackageVersionDescription struct {
	DisplayName          string                `json:"displayName"`
	Format               string                `json:"format"`
	HomePage             string                `json:"homePage"`
	Licenses             []PackageLicense      `json:"licenses"`
	Namespace            string                `json:"namespace"`
	Origin               *PackageVersionOrigin `json:"origin"`
	PackageName          string                `json:"packageName"`
	PublishedTime        common.FloatTime      `json:"publishedTime,omitempty"`
	Revision             string                `json:"revision"`
	SourceCodeRepository string                `json:"sourceCodeRepository"`
	Status               string                `json:"status"`
	Summary              string                `json:"summary"`
	Version              string                `json:"version"`
}

type PackageVersionAsset struct {
	Hashes map[string]string `json:"hashes"`
	Name   string            `json:"name"`
	Size   int64             `json:"size"`
}

type PackageLicense struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type PackageVersionOrigin struct {
	DomainEntryPoint *PackageVersionDomainEntryPoint `json:"domainEntryPoint"`
	OriginType       string                          `json:"originType"`
}

type PackageVersionDomainEntryPoint struct {
	ExternalConnectionName string `json:"externalConnectionName"`
	RepositoryName         string `json:"repositoryName"`
}

type DescribePackageVersionResponse struct {
	PackageVersion PackageVersionDescription `json:"packageVersion"`
}

type DescribePackageVersionInput struct {
	Domain         string
	Repository     string
	Format         string
	Namespace      string
	Package        string
	PackageVersion string
}

type ListPackageVersionAssetsInput struct {
	Domain         string
	Repository     string
	Format         string
	Namespace      string
	Package        string
	PackageVersion string
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

type ListDomainsResponse struct {
	Domains []Domain `json:"domains"`
}

type Domain struct {
	Name   string `json:"name"`
	Arn    string `json:"arn"`
	Status string `json:"status"`
}

func (c *Client) ListDomains() ([]Domain, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/domains", c.region)
	domains := []Domain{}
	nextToken := ""

	for {
		payload := map[string]any{
			"maxResults": 100,
		}
		if strings.TrimSpace(nextToken) != "" {
			payload["nextToken"] = strings.TrimSpace(nextToken)
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to encode list domains request: %w", err)
		}

		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to build list domains request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		if err := c.signRequest(req, bodyBytes); err != nil {
			return nil, err
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list domains request failed: %w", err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read list domains response: %w", err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("list domains failed with %d: %s", res.StatusCode, string(body))
		}

		var response struct {
			Domains   []Domain `json:"domains"`
			NextToken string   `json:"nextToken"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode list domains response: %w", err)
		}

		domains = append(domains, response.Domains...)
		if strings.TrimSpace(response.NextToken) == "" {
			break
		}
		nextToken = response.NextToken
	}

	return domains, nil
}

type ListRepositoriesResponse struct {
	Repositories []Repository `json:"repositories"`
}

type Repository struct {
	Name       string `json:"name"`
	Arn        string `json:"arn"`
	DomainName string `json:"domainName"`
}

// RepositoryDescription is returned by CreateRepository and DeleteRepository.
type RepositoryDescription struct {
	AdministratorAccount string                         `json:"administratorAccount"`
	Arn                  string                         `json:"arn"`
	CreatedTime          float64                        `json:"createdTime"`
	Description          string                         `json:"description"`
	DomainName           string                         `json:"domainName"`
	DomainOwner          string                         `json:"domainOwner"`
	ExternalConnections  []RepositoryExternalConnection `json:"externalConnections"`
	Name                 string                         `json:"name"`
	Upstreams            []UpstreamRepositoryInfo       `json:"upstreams"`
}

type RepositoryExternalConnection struct {
	ExternalConnectionName string `json:"externalConnectionName"`
	PackageFormat          string `json:"packageFormat"`
	Status                 string `json:"status"`
}

type UpstreamRepositoryInfo struct {
	RepositoryName string `json:"repositoryName"`
}

func (c *Client) ListRepositories(domain string) ([]Repository, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/repositories", c.region)
	repositories := []Repository{}
	nextToken := ""

	for {
		payload := map[string]any{
			"maxResults": 100,
		}

		if domain != "" {
			payload["domain"] = domain
		}

		if strings.TrimSpace(nextToken) != "" {
			payload["nextToken"] = strings.TrimSpace(nextToken)
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to encode list repositories request: %w", err)
		}

		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to build list repositories request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		if err := c.signRequest(req, bodyBytes); err != nil {
			return nil, err
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list repositories request failed: %w", err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read list repositories response: %w", err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("list repositories failed with %d: %s", res.StatusCode, string(body))
		}

		var response struct {
			Repositories []Repository `json:"repositories"`
			NextToken    string       `json:"nextToken"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode list repositories response: %w", err)
		}

		repositories = append(repositories, response.Repositories...)
		if strings.TrimSpace(response.NextToken) == "" {
			break
		}
		nextToken = response.NextToken
	}

	return repositories, nil
}

// CreateRepositoryInput is the input for CreateRepository.
type CreateRepositoryInput struct {
	Domain      string
	Repository  string
	Description string
}

// CreateRepository creates a repository in the given domain.
func (c *Client) CreateRepository(input CreateRepositoryInput) (*RepositoryDescription, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/repository", c.region)
	payload := map[string]any{}
	if strings.TrimSpace(input.Description) != "" {
		payload["description"] = strings.TrimSpace(input.Description)
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode create repository request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build create repository request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	query.Set("repository", input.Repository)
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Content-Type", "application/json")

	if err := c.signRequest(req, bodyBytes); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create repository request failed: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read create repository response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("create repository failed with %d: %s", res.StatusCode, string(body))
	}

	var response struct {
		Repository RepositoryDescription `json:"repository"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode create repository response: %w", err)
	}

	return &response.Repository, nil
}

// DeleteRepositoryInput is the input for DeleteRepository.
type DeleteRepositoryInput struct {
	Domain     string
	Repository string
}

// DeleteRepository deletes a repository from the given domain.
func (c *Client) DeleteRepository(input DeleteRepositoryInput) (*RepositoryDescription, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/repository", c.region)
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build delete repository request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	query.Set("repository", input.Repository)
	req.URL.RawQuery = query.Encode()

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("delete repository request failed: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read delete repository response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("delete repository failed with %d: %s", res.StatusCode, string(body))
	}

	var response struct {
		Repository RepositoryDescription `json:"repository"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode delete repository response: %w", err)
	}

	return &response.Repository, nil
}

// SuccessfulPackageVersionInfo is returned for each successfully updated or copied package version.
type SuccessfulPackageVersionInfo struct {
	Revision string `json:"revision"`
	Status   string `json:"status"`
}

// PackageVersionError is returned for each failed package version update or copy.
type PackageVersionError struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// UpdatePackageVersionsStatusInput is the input for UpdatePackageVersionsStatus.
type UpdatePackageVersionsStatusInput struct {
	Domain           string
	Repository       string
	Format           string
	Namespace        string
	Package          string
	Versions         []string          // required
	VersionRevisions map[string]string // optional; use Versions or VersionRevisions, not both
	TargetStatus     string            // Archived | Published | Unlisted
	ExpectedStatus   string            // optional
}

// UpdatePackageVersionsStatusResponse is the response from UpdatePackageVersionsStatus.
type UpdatePackageVersionsStatusResponse struct {
	SuccessfulVersions map[string]SuccessfulPackageVersionInfo `json:"successfulVersions"`
	FailedVersions     map[string]PackageVersionError          `json:"failedVersions"`
}

// UpdatePackageVersionsStatus updates the status of one or more package versions.
func (c *Client) UpdatePackageVersionsStatus(input UpdatePackageVersionsStatusInput) (*UpdatePackageVersionsStatusResponse, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/package/versions/update_status", c.region)
	payload := map[string]any{"targetStatus": input.TargetStatus}
	if len(input.VersionRevisions) > 0 {
		payload["versionRevisions"] = input.VersionRevisions
	} else {
		payload["versions"] = input.Versions
	}
	if strings.TrimSpace(input.ExpectedStatus) != "" {
		payload["expectedStatus"] = strings.TrimSpace(input.ExpectedStatus)
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode update package versions status request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build update package versions status request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	query.Set("format", input.Format)
	query.Set("package", input.Package)
	query.Set("repository", input.Repository)
	if strings.TrimSpace(input.Namespace) != "" {
		query.Set("namespace", strings.TrimSpace(input.Namespace))
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Content-Type", "application/json")

	if err := c.signRequest(req, bodyBytes); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update package versions status request failed: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read update package versions status response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("update package versions status failed with %d: %s", res.StatusCode, string(body))
	}

	var response UpdatePackageVersionsStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode update package versions status response: %w", err)
	}
	return &response, nil
}

// CopyPackageVersionsInput is the input for CopyPackageVersions.
type CopyPackageVersionsInput struct {
	Domain                string
	SourceRepository      string
	DestinationRepository string
	Format                string
	Namespace             string
	Package               string
	Versions              []string          // use Versions or VersionRevisions, not both
	VersionRevisions      map[string]string // optional
	AllowOverwrite        bool
	IncludeFromUpstream   bool
}

// CopyPackageVersionsResponse is the response from CopyPackageVersions.
type CopyPackageVersionsResponse struct {
	SuccessfulVersions map[string]SuccessfulPackageVersionInfo `json:"successfulVersions"`
	FailedVersions     map[string]PackageVersionError          `json:"failedVersions"`
}

// CopyPackageVersions copies package versions from one repository to another in the same domain.
func (c *Client) CopyPackageVersions(input CopyPackageVersionsInput) (*CopyPackageVersionsResponse, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/package/versions/copy", c.region)
	payload := map[string]any{
		"allowOverwrite":      input.AllowOverwrite,
		"includeFromUpstream": input.IncludeFromUpstream,
	}
	if len(input.VersionRevisions) > 0 {
		payload["versionRevisions"] = input.VersionRevisions
	} else {
		payload["versions"] = input.Versions
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode copy package versions request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build copy package versions request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	query.Set("format", input.Format)
	query.Set("package", input.Package)
	query.Set("source-repository", input.SourceRepository)
	query.Set("destination-repository", input.DestinationRepository)
	if strings.TrimSpace(input.Namespace) != "" {
		query.Set("namespace", strings.TrimSpace(input.Namespace))
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Content-Type", "application/json")

	if err := c.signRequest(req, bodyBytes); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("copy package versions request failed: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read copy package versions response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("copy package versions failed with %d: %s", res.StatusCode, string(body))
	}

	var response CopyPackageVersionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode copy package versions response: %w", err)
	}
	return &response, nil
}

// DeletePackageVersionsInput is the input for DeletePackageVersions.
type DeletePackageVersionsInput struct {
	Domain         string
	Repository     string
	Format         string
	Namespace      string
	Package        string
	Versions       []string
	ExpectedStatus string // optional
}

// DeletePackageVersionsResponse is the response from DeletePackageVersions.
type DeletePackageVersionsResponse struct {
	SuccessfulVersions map[string]SuccessfulPackageVersionInfo `json:"successfulVersions"`
	FailedVersions     map[string]PackageVersionError          `json:"failedVersions"`
}

// DeletePackageVersions permanently deletes one or more package versions.
func (c *Client) DeletePackageVersions(input DeletePackageVersionsInput) (*DeletePackageVersionsResponse, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/package/versions/delete", c.region)
	payload := map[string]any{"versions": input.Versions}
	if strings.TrimSpace(input.ExpectedStatus) != "" {
		payload["expectedStatus"] = strings.TrimSpace(input.ExpectedStatus)
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode delete package versions request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build delete package versions request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	query.Set("format", input.Format)
	query.Set("package", input.Package)
	query.Set("repository", input.Repository)
	if strings.TrimSpace(input.Namespace) != "" {
		query.Set("namespace", strings.TrimSpace(input.Namespace))
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Content-Type", "application/json")

	if err := c.signRequest(req, bodyBytes); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("delete package versions request failed: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read delete package versions response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("delete package versions failed with %d: %s", res.StatusCode, string(body))
	}

	var response DeletePackageVersionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode delete package versions response: %w", err)
	}
	return &response, nil
}

// DisposePackageVersionsInput is the input for DisposePackageVersions.
type DisposePackageVersionsInput struct {
	Domain           string
	Repository       string
	Format           string
	Namespace        string
	Package          string
	Versions         []string
	VersionRevisions map[string]string // optional; use Versions or VersionRevisions, not both
	ExpectedStatus   string            // optional
}

// DisposePackageVersionsResponse is the response from DisposePackageVersions.
type DisposePackageVersionsResponse struct {
	SuccessfulVersions map[string]SuccessfulPackageVersionInfo `json:"successfulVersions"`
	FailedVersions     map[string]PackageVersionError          `json:"failedVersions"`
}

// DisposePackageVersions deletes assets and sets package version status to Disposed.
func (c *Client) DisposePackageVersions(input DisposePackageVersionsInput) (*DisposePackageVersionsResponse, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/package/versions/dispose", c.region)
	payload := map[string]any{}
	if len(input.VersionRevisions) > 0 {
		payload["versionRevisions"] = input.VersionRevisions
	} else {
		payload["versions"] = input.Versions
	}
	if strings.TrimSpace(input.ExpectedStatus) != "" {
		payload["expectedStatus"] = strings.TrimSpace(input.ExpectedStatus)
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode dispose package versions request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build dispose package versions request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	query.Set("format", input.Format)
	query.Set("package", input.Package)
	query.Set("repository", input.Repository)
	if strings.TrimSpace(input.Namespace) != "" {
		query.Set("namespace", strings.TrimSpace(input.Namespace))
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Content-Type", "application/json")

	if err := c.signRequest(req, bodyBytes); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dispose package versions request failed: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read dispose package versions response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("dispose package versions failed with %d: %s", res.StatusCode, string(body))
	}

	var response DisposePackageVersionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode dispose package versions response: %w", err)
	}
	return &response, nil
}

func (c *Client) DescribePackageVersion(input DescribePackageVersionInput) (*PackageVersionDescription, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/package/version", c.region)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build describe package version request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	query.Set("format", input.Format)
	query.Set("package", input.Package)
	query.Set("repository", input.Repository)
	query.Set("version", input.PackageVersion)

	if strings.TrimSpace(input.Namespace) != "" {
		query.Set("namespace", strings.TrimSpace(input.Namespace))
	}

	req.URL.RawQuery = query.Encode()

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("describe package version request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read describe package version response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(body); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("describe package version failed with %d: %s", res.StatusCode, string(body))
	}

	response := DescribePackageVersionResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode describe package version response: %w", err)
	}

	return &response.PackageVersion, nil
}

type ListPackageVersionAssetsResponse struct {
	Assets    []PackageVersionAsset `json:"assets"`
	NextToken string                `json:"nextToken"`
}

func (c *Client) ListPackageVersionAssets(input ListPackageVersionAssetsInput) ([]PackageVersionAsset, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/package/version/assets", c.region)
	assets := []PackageVersionAsset{}
	nextToken := ""

	for {
		req, err := http.NewRequest(http.MethodPost, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build list package version assets request: %w", err)
		}

		query := url.Values{}
		query.Set("domain", input.Domain)
		query.Set("format", input.Format)
		query.Set("package", input.Package)
		query.Set("repository", input.Repository)
		query.Set("version", input.PackageVersion)
		query.Set("max-results", "1000")

		if strings.TrimSpace(input.Namespace) != "" {
			query.Set("namespace", strings.TrimSpace(input.Namespace))
		}
		if strings.TrimSpace(nextToken) != "" {
			query.Set("next-token", strings.TrimSpace(nextToken))
		}

		req.URL.RawQuery = query.Encode()

		if err := c.signRequest(req, []byte{}); err != nil {
			return nil, err
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list package version assets request failed: %w", err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read list package version assets response: %w", err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			if awsErr := common.ParseError(body); awsErr != nil {
				return nil, awsErr
			}
			return nil, fmt.Errorf("list package version assets failed with %d: %s", res.StatusCode, string(body))
		}

		response := ListPackageVersionAssetsResponse{}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode list package version assets response: %w", err)
		}

		assets = append(assets, response.Assets...)
		if strings.TrimSpace(response.NextToken) == "" {
			break
		}
		nextToken = response.NextToken
	}

	return assets, nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "codeartifact", c.region, time.Now())
}
