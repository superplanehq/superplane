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
	PublishedTime        float64               `json:"publishedTime"`
	Revision             string                `json:"revision"`
	SourceCodeRepository string                `json:"sourceCodeRepository"`
	Status               string                `json:"status"`
	Summary              string                `json:"summary"`
	Version              string                `json:"version"`
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
	DomainOwner    string
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

func (c *Client) ListRepositories() ([]Repository, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/repositories", c.region)
	repositories := []Repository{}
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

func (c *Client) DescribePackageVersion(input DescribePackageVersionInput) (*PackageVersionDescription, error) {
	endpoint := fmt.Sprintf("https://codeartifact.%s.amazonaws.com/v1/package/version", c.region)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build describe package version request: %w", err)
	}

	query := url.Values{}
	query.Set("domain", input.Domain)
	if strings.TrimSpace(input.DomainOwner) != "" {
		query.Set("domain-owner", strings.TrimSpace(input.DomainOwner))
	}
	query.Set("format", input.Format)
	if strings.TrimSpace(input.Namespace) != "" {
		query.Set("namespace", strings.TrimSpace(input.Namespace))
	}
	query.Set("package", input.Package)
	query.Set("repository", input.Repository)
	query.Set("version", input.PackageVersion)
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

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "codeartifact", c.region, time.Now())
}
