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
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type listBucketsResponse struct {
	XMLName xml.Name       `xml:"ListAllMyBucketsResult"`
	Buckets listOfBuckets  `xml:"Buckets"`
	Owner   bucketOwnerXML `xml:"Owner"`
}

type listOfBuckets struct {
	Bucket []Bucket `xml:"Bucket"`
}

type bucketOwnerXML struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type listObjectsV2Response struct {
	XMLName               xml.Name `xml:"ListBucketResult"`
	Name                  string   `xml:"Name"`
	IsTruncated           bool     `xml:"IsTruncated"`
	Contents              []Object `xml:"Contents"`
	NextContinuationToken string   `xml:"NextContinuationToken"`
}

type copyObjectResultXML struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	ETag         string   `xml:"ETag"`
	LastModified string   `xml:"LastModified"`
}

type getObjectAttributesResponse struct {
	XMLName      xml.Name `xml:"GetObjectAttributesResponse"`
	ETag         string   `xml:"ETag"`
	StorageClass string   `xml:"StorageClass"`
	ObjectSize   int64    `xml:"ObjectSize"`
	LastModified string
}

func (c *Client) endpoint(bucket string) string {
	if bucket == "" {
		return fmt.Sprintf("https://s3.%s.amazonaws.com/", c.region)
	}
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s", c.region, bucket)
}

func (c *Client) objectEndpoint(bucket, key string) string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", c.region, bucket, key)
}

func (c *Client) CreateBucket(bucket string) error {
	endpoint := c.endpoint(bucket)

	var body []byte
	if c.region != "us-east-1" {
		locationConstraint := fmt.Sprintf(
			`<CreateBucketConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><LocationConstraint>%s</LocationConstraint></CreateBucketConfiguration>`,
			c.region,
		)
		body = []byte(locationConstraint)
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build CreateBucket request: %w", err)
	}

	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("CreateBucket request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read CreateBucket response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("S3 CreateBucket failed with %d: %s", res.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) DeleteBucket(bucket string) error {
	endpoint := c.endpoint(bucket)
	return c.doSimpleRequest(http.MethodDelete, endpoint, "DeleteBucket")
}

func (c *Client) HeadBucket(bucket string) (http.Header, error) {
	endpoint := c.endpoint(bucket)
	req, err := http.NewRequest(http.MethodHead, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build HeadBucket request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HeadBucket request failed: %w", err)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 HeadBucket failed with %d", res.StatusCode)
	}

	return res.Header, nil
}

func (c *Client) HeadObject(bucket, key string) (http.Header, error) {
	endpoint := c.objectEndpoint(bucket, key)
	req, err := http.NewRequest(http.MethodHead, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build HeadObject request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HeadObject request failed: %w", err)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 HeadObject failed with %d", res.StatusCode)
	}

	return res.Header, nil
}

func (c *Client) CopyObject(bucket, key, copySource string) (*copyObjectResultXML, error) {
	endpoint := c.objectEndpoint(bucket, key)
	req, err := http.NewRequest(http.MethodPut, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build CopyObject request: %w", err)
	}

	req.Header.Set("x-amz-copy-source", copySource)
	if err := c.signRequest(req, nil); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CopyObject request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CopyObject response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 CopyObject failed with %d: %s", res.StatusCode, string(body))
	}

	var result copyObjectResultXML
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode CopyObject response: %w", err)
	}

	return &result, nil
}

func (c *Client) DeleteObject(bucket, key string) error {
	endpoint := c.objectEndpoint(bucket, key)
	return c.doSimpleRequest(http.MethodDelete, endpoint, "DeleteObject")
}

func (c *Client) GetObjectAttributes(bucket, key string, attributes []string) (*getObjectAttributesResponse, http.Header, error) {
	endpoint := c.objectEndpoint(bucket, key) + "?attributes"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build GetObjectAttributes request: %w", err)
	}

	req.Header.Set("x-amz-object-attributes", strings.Join(attributes, ","))
	if err := c.signRequest(req, nil); err != nil {
		return nil, nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("GetObjectAttributes request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read GetObjectAttributes response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("S3 GetObjectAttributes failed with %d: %s", res.StatusCode, string(body))
	}

	var result getObjectAttributesResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode GetObjectAttributes response: %w", err)
	}

	result.LastModified = res.Header.Get("Last-Modified")

	return &result, res.Header, nil
}

func (c *Client) PutObject(bucket, key string, data []byte, contentType string) error {
	endpoint := c.objectEndpoint(bucket, key)
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to build PutObject request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if err := c.signRequest(req, data); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("PutObject request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read PutObject response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("S3 PutObject failed with %d: %s", res.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) ListBuckets() ([]Bucket, error) {
	endpoint := c.endpoint("")
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build ListBuckets request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, err
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
		return nil, fmt.Errorf("S3 ListBuckets failed with %d: %s", res.StatusCode, string(body))
	}

	var resp listBucketsResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode ListBuckets response: %w", err)
	}

	return resp.Buckets.Bucket, nil
}

func (c *Client) ListObjects(bucket string) ([]Object, error) {
	var allObjects []Object
	continuationToken := ""

	for {
		endpoint := c.endpoint(bucket) + "?list-type=2"
		if continuationToken != "" {
			endpoint += "&continuation-token=" + continuationToken
		}

		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build ListObjects request: %w", err)
		}

		if err := c.signRequest(req, nil); err != nil {
			return nil, err
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ListObjects request failed: %w", err)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read ListObjects response: %w", err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("S3 ListObjects failed with %d: %s", res.StatusCode, string(body))
		}

		var resp listObjectsV2Response
		if err := xml.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to decode ListObjects response: %w", err)
		}

		allObjects = append(allObjects, resp.Contents...)

		if !resp.IsTruncated {
			break
		}
		continuationToken = resp.NextContinuationToken
	}

	return allObjects, nil
}

func (c *Client) doSimpleRequest(method, endpoint, operation string) error {
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to build %s request: %w", operation, err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", operation, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read %s response: %w", operation, err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("S3 %s failed with %d: %s", operation, res.StatusCode, string(body))
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	if payload == nil {
		payload = []byte{}
	}
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "s3", c.region, time.Now())
}
