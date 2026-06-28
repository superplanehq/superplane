package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
)

// emptyPayloadHash is the SHA-256 of an empty body, required when signing
// S3 requests that carry no payload (GET/DELETE/HEAD).
const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// defaultRegion is the region whose buckets report an empty LocationConstraint.
const defaultRegion = "us-east-1"

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      strings.TrimSpace(region),
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

type Bucket struct {
	Name         string
	CreationDate string
}

type listAllMyBucketsResult struct {
	XMLName xml.Name        `xml:"ListAllMyBucketsResult"`
	Buckets []bucketElement `xml:"Buckets>Bucket"`
}

type bucketElement struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type locationConstraint struct {
	XMLName xml.Name `xml:"LocationConstraint"`
	Value   string   `xml:",chardata"`
}

// bucketARN returns the canonical S3 bucket ARN. S3 bucket ARNs omit region
// and account ID.
func bucketARN(name string) string {
	return fmt.Sprintf("arn:aws:s3:::%s", strings.TrimSpace(name))
}

// endpoint returns the regional path-style S3 endpoint.
func (c *Client) endpoint() string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com", c.region)
}

func (c *Client) bucketURL(bucket string) string {
	return fmt.Sprintf("%s/%s", c.endpoint(), url.PathEscape(strings.TrimSpace(bucket)))
}

// CreateBucket creates a new bucket in the client's region. Regions other than
// us-east-1 require a CreateBucketConfiguration body with the LocationConstraint.
func (c *Client) CreateBucket(name string) error {
	var body []byte
	if c.region != "" && c.region != defaultRegion {
		body = []byte(fmt.Sprintf(
			`<CreateBucketConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><LocationConstraint>%s</LocationConstraint></CreateBucketConfiguration>`,
			c.region,
		))
	}

	_, err := c.do(http.MethodPut, c.bucketURL(name), body)
	return err
}

// GetBucketLocation returns the region a bucket resides in. An empty
// LocationConstraint maps to us-east-1.
func (c *Client) GetBucketLocation(name string) (string, error) {
	body, err := c.do(http.MethodGet, c.bucketURL(name)+"?location=", nil)
	if err != nil {
		return "", err
	}

	var location locationConstraint
	if err := xml.Unmarshal(body, &location); err != nil {
		return "", fmt.Errorf("failed to decode GetBucketLocation response: %w", err)
	}

	region := strings.TrimSpace(location.Value)
	if region == "" {
		return defaultRegion, nil
	}

	return region, nil
}

// DeleteBucket deletes an (empty) bucket.
func (c *Client) DeleteBucket(name string) error {
	_, err := c.do(http.MethodDelete, c.bucketURL(name), nil)
	return err
}

// ListBuckets returns every bucket owned by the authenticated account.
func (c *Client) ListBuckets() ([]Bucket, error) {
	body, err := c.do(http.MethodGet, c.endpoint()+"/", nil)
	if err != nil {
		return nil, err
	}

	var result listAllMyBucketsResult
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode ListBuckets response: %w", err)
	}

	buckets := make([]Bucket, 0, len(result.Buckets))
	for _, b := range result.Buckets {
		buckets = append(buckets, Bucket{
			Name:         b.Name,
			CreationDate: b.CreationDate,
		})
	}

	return buckets, nil
}

func (c *Client) do(method, requestURL string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, requestURL, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to build S3 request: %w", err)
	}

	if err := c.signRequest(req, body); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("S3 request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	payloadHash := emptyPayloadHash
	if payload != nil {
		hash := sha256.Sum256(payload)
		payloadHash = hex.EncodeToString(hash[:])
	}

	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "s3", c.region, time.Now())
}
