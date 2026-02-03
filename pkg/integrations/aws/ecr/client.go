package ecr

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const targetPrefix = "AmazonEC2ContainerRegistry_V20150921."

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) DescribeRepository(name string) (*Repository, error) {
	payload := map[string]any{
		"repositoryNames": []string{name},
	}

	var response struct {
		Repositories []Repository `json:"repositories"`
	}

	if err := c.postJSON("DescribeRepositories", payload, &response); err != nil {
		return nil, err
	}

	if len(response.Repositories) == 0 {
		return nil, fmt.Errorf("repository not found")
	}

	return &response.Repositories[0], nil
}

func (c *Client) ListRepositories() ([]Repository, error) {
	repositories := []Repository{}
	nextToken := ""

	for {
		payload := map[string]any{
			"maxResults": 100,
		}
		if nextToken != "" {
			payload["nextToken"] = nextToken
		}

		var response struct {
			Repositories []Repository `json:"repositories"`
			NextToken    string       `json:"nextToken"`
		}

		if err := c.postJSON("DescribeRepositories", payload, &response); err != nil {
			return nil, err
		}

		repositories = append(repositories, response.Repositories...)
		if response.NextToken == "" {
			break
		}
		nextToken = response.NextToken
	}

	return repositories, nil
}

type DescribeImageResponse struct {
	ImageDetails []ImageDetail `json:"imageDetails"`
}

type ImageDetail struct {
	RegistryID             string   `json:"registryId"`
	RepositoryName         string   `json:"repositoryName"`
	ImageDigest            string   `json:"imageDigest"`
	ImageTags              []string `json:"imageTags"`
	ImageSizeInBytes       int64    `json:"imageSizeInBytes"`
	ImageManifestMediaType string   `json:"imageManifestMediaType"`
	ArtifactMediaType      string   `json:"artifactMediaType"`
}

func (c *Client) DescribeImage(repositoryName string, imageDigest string, imageTag string) (*ImageDetail, error) {
	imageID := map[string]any{}
	if strings.TrimSpace(imageDigest) != "" {
		imageID["imageDigest"] = strings.TrimSpace(imageDigest)
	}
	if strings.TrimSpace(imageTag) != "" {
		imageID["imageTag"] = strings.TrimSpace(imageTag)
	}
	if len(imageID) == 0 {
		return nil, errors.New("image digest or image tag is required")
	}

	payload := map[string]any{
		"repositoryName": repositoryName,
		"imageIds":       []map[string]any{imageID},
	}

	response := DescribeImageResponse{}
	if err := c.postJSON("DescribeImages", payload, &response); err != nil {
		return nil, err
	}

	if len(response.ImageDetails) == 0 {
		return nil, errors.New("image not found")
	}

	if len(response.ImageDetails) > 1 {
		return nil, errors.New("multiple images found")
	}

	return &response.ImageDetails[0], nil
}

type DescribeImageScanFindingsResponse struct {
	ImageScanFindings ImageScanFindings `json:"imageScanFindings"`
	RegistryID        string            `json:"registryId"`
	RepositoryName    string            `json:"repositoryName"`
	ImageID           ImageIdentifier   `json:"imageId"`
	ImageScanStatus   ImageScanStatus   `json:"imageScanStatus"`
}

type ImageIdentifier struct {
	ImageDigest string `json:"imageDigest"`
	ImageTag    string `json:"imageTag"`
}

type ImageScanStatus struct {
	Status      string `json:"status"`
	Description string `json:"description"`
}

type ImageScanFindings struct {
	Findings              []ImageScanFinding `json:"findings"`
	FindingSeverityCounts map[string]int     `json:"findingSeverityCounts"`
}

type ImageScanFinding struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	URI         string                      `json:"uri"`
	Severity    string                      `json:"severity"`
	Attributes  []ImageScanFindingAttribute `json:"attributes"`
}

type ImageScanFindingAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c *Client) DescribeImageScanFindings(repositoryName string, imageDigest string, imageTag string) (*DescribeImageScanFindingsResponse, error) {
	imageID := map[string]any{}
	if strings.TrimSpace(imageDigest) != "" {
		imageID["imageDigest"] = strings.TrimSpace(imageDigest)
	}
	if strings.TrimSpace(imageTag) != "" {
		imageID["imageTag"] = strings.TrimSpace(imageTag)
	}
	if len(imageID) == 0 {
		return nil, errors.New("image digest or image tag is required")
	}

	payload := map[string]any{
		"repositoryName": repositoryName,
		"imageId":        imageID,
	}

	response := DescribeImageScanFindingsResponse{}
	if err := c.postJSON("DescribeImageScanFindings", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

type ScanImageResponse struct {
	ImageIdentifier ImageIdentifier `json:"imageId"`
	ScanStatus      ImageScanStatus `json:"scanStatus"`
	RepositoryName  string          `json:"repositoryName"`
	RegistryID      string          `json:"registryId"`
}

func (c *Client) ScanImage(repositoryName string, imageDigest string, imageTag string) (*ScanImageResponse, error) {
	imageID := map[string]any{}
	if strings.TrimSpace(imageDigest) != "" {
		imageID["imageDigest"] = strings.TrimSpace(imageDigest)
	}
	if strings.TrimSpace(imageTag) != "" {
		imageID["imageTag"] = strings.TrimSpace(imageTag)
	}

	payload := map[string]any{
		"repositoryName": repositoryName,
		"imageId":        imageID,
	}

	response := ScanImageResponse{}
	if err := c.postJSON("StartImageScan", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) postJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.ecr.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", targetPrefix+action)

	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("ECR API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "ecr", c.region, time.Now())
}
