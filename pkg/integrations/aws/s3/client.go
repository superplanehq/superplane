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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
)

const s3ServiceName = "s3"

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

type createBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	LocationConstraint string   `xml:"LocationConstraint"`
}

type listAllMyBucketsResult struct {
	XMLName xml.Name       `xml:"ListAllMyBucketsResult"`
	Buckets bucketListWrap `xml:"Buckets"`
}

type bucketListWrap struct {
	Bucket []bucketEntry `xml:"Bucket"`
}

type bucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

func (c *Client) endpoint() string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com", c.region)
}

func (c *Client) CreateBucket(name string) (string, error) {
	name = strings.TrimSpace(name)
	url := fmt.Sprintf("%s/%s", c.endpoint(), name)

	var body []byte
	if c.region != "us-east-1" {
		config := createBucketConfiguration{
			LocationConstraint: c.region,
		}

		var err error
		body, err = xml.Marshal(config)
		if err != nil {
			return "", fmt.Errorf("failed to marshal CreateBucketConfiguration: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build CreateBucket request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/xml")
	}

	if err := c.signRequest(req, body); err != nil {
		return "", fmt.Errorf("failed to sign CreateBucket request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("CreateBucket request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read CreateBucket response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("CreateBucket failed with %d: %s", res.StatusCode, string(respBody))
	}

	location := res.Header.Get("Location")
	if location == "" {
		location = fmt.Sprintf("/%s", name)
	}

	return location, nil
}

func (c *Client) ListBuckets() ([]Bucket, error) {
	req, err := http.NewRequest(http.MethodGet, c.endpoint()+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build ListBuckets request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("failed to sign ListBuckets request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ListBuckets request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ListBuckets response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("ListBuckets failed with %d: %s", res.StatusCode, string(body))
	}

	var resp listAllMyBucketsResult
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode ListBuckets response: %w", err)
	}

	buckets := make([]Bucket, 0, len(resp.Buckets.Bucket))
	for _, b := range resp.Buckets.Bucket {
		buckets = append(buckets, Bucket{
			Name:         strings.TrimSpace(b.Name),
			CreationDate: strings.TrimSpace(b.CreationDate),
		})
	}

	return buckets, nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	if payload == nil {
		payload = []byte{}
	}

	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, s3ServiceName, c.region, time.Now())
}
