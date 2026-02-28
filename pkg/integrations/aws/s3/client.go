package s3

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const s3ServiceName = "s3"

// Client provides lightweight S3 API operations through signed HTTP requests.
type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

// NewClient creates a region-scoped S3 client.
func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      strings.TrimSpace(region),
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) bucketURL(bucket string) string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s", c.region, bucket)
}

func (c *Client) objectURL(bucket, key string) string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", c.region, bucket, encodeKey(key))
}

// CreateBucket creates an S3 bucket in the client's region.
func (c *Client) CreateBucket(name string) (*Bucket, error) {
	var body []byte
	if c.region != "us-east-1" {
		config := createBucketConfiguration{LocationConstraint: c.region}
		var err error
		body, err = xml.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("s3 client: failed to marshal CreateBucketConfiguration: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPut, c.bucketURL(name), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build CreateBucket request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/xml")
	}

	if err := c.signRequest(req, body); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign CreateBucket request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: CreateBucket request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, "CreateBucket"); err != nil {
		return nil, err
	}

	location := strings.TrimSpace(resp.Header.Get("Location"))
	return &Bucket{
		Name:     name,
		Location: location,
	}, nil
}

// HeadBucket checks whether a bucket exists and is accessible.
func (c *Client) HeadBucket(name string) (*BucketInfo, error) {
	req, err := http.NewRequest(http.MethodHead, c.bucketURL(name), nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build HeadBucket request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign HeadBucket request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: HeadBucket request failed: %w", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return &BucketInfo{
			Name:   name,
			Region: c.region,
			Exists: false,
		}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("s3 client: HeadBucket request failed with status %d", resp.StatusCode)
	}

	region := strings.TrimSpace(resp.Header.Get("X-Amz-Bucket-Region"))
	if region == "" {
		region = c.region
	}

	return &BucketInfo{
		Name:   name,
		Region: region,
		Exists: true,
	}, nil
}

// DeleteBucket deletes an empty S3 bucket.
func (c *Client) DeleteBucket(name string) error {
	req, err := http.NewRequest(http.MethodDelete, c.bucketURL(name), nil)
	if err != nil {
		return fmt.Errorf("s3 client: failed to build DeleteBucket request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return fmt.Errorf("s3 client: failed to sign DeleteBucket request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("s3 client: DeleteBucket request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.checkResponse(resp, "DeleteBucket")
}

// PutObject uploads an object to an S3 bucket.
func (c *Client) PutObject(bucket, key string, body []byte, contentType string) (*PutObjectResult, error) {
	req, err := http.NewRequest(http.MethodPut, c.objectURL(bucket, key), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build PutObject request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if err := c.signRequest(req, body); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign PutObject request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: PutObject request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp, "PutObject"); err != nil {
		return nil, err
	}

	return &PutObjectResult{
		Bucket: bucket,
		Key:    key,
		ETag:   cleanETag(resp.Header.Get("ETag")),
	}, nil
}

// CopyObject copies an object within S3.
func (c *Client) CopyObject(sourceBucket, sourceKey, destBucket, destKey string) (*CopyObjectResult, error) {
	copySource := fmt.Sprintf("/%s/%s", sourceBucket, encodeKey(sourceKey))

	req, err := http.NewRequest(http.MethodPut, c.objectURL(destBucket, destKey), nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build CopyObject request: %w", err)
	}

	req.Header.Set("x-amz-copy-source", copySource)

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign CopyObject request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: CopyObject request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read CopyObject response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if awsErr := parseS3Error(respBody); awsErr != nil {
			return nil, fmt.Errorf("s3 client: CopyObject request failed: %w", awsErr)
		}
		return nil, fmt.Errorf("s3 client: CopyObject request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result copyObjectResponse
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("s3 client: failed to decode CopyObject response: %w", err)
	}

	return &CopyObjectResult{
		SourceBucket: sourceBucket,
		SourceKey:    sourceKey,
		Bucket:       destBucket,
		Key:          destKey,
		ETag:         cleanETag(result.ETag),
		LastModified: strings.TrimSpace(result.LastModified),
	}, nil
}

// DeleteObject deletes an object from an S3 bucket.
func (c *Client) DeleteObject(bucket, key string) error {
	req, err := http.NewRequest(http.MethodDelete, c.objectURL(bucket, key), nil)
	if err != nil {
		return fmt.Errorf("s3 client: failed to build DeleteObject request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return fmt.Errorf("s3 client: failed to sign DeleteObject request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("s3 client: DeleteObject request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.checkResponse(resp, "DeleteObject")
}

// HeadObject retrieves metadata for an S3 object.
func (c *Client) HeadObject(bucket, key string) (*ObjectMetadata, error) {
	req, err := http.NewRequest(http.MethodHead, c.objectURL(bucket, key), nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build HeadObject request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign HeadObject request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: HeadObject request failed: %w", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("s3 client: HeadObject request failed with status %d", resp.StatusCode)
	}

	contentLength, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	return &ObjectMetadata{
		Key:           key,
		Bucket:        bucket,
		ContentLength: contentLength,
		ContentType:   strings.TrimSpace(resp.Header.Get("Content-Type")),
		ETag:          cleanETag(resp.Header.Get("ETag")),
		LastModified:  strings.TrimSpace(resp.Header.Get("Last-Modified")),
		StorageClass:  strings.TrimSpace(resp.Header.Get("x-amz-storage-class")),
	}, nil
}

// GetObjectAttributes retrieves attributes for an S3 object.
func (c *Client) GetObjectAttributes(bucket, key string, attributes []string) (*ObjectAttributes, error) {
	objectURL := c.objectURL(bucket, key) + "?attributes"

	req, err := http.NewRequest(http.MethodGet, objectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build GetObjectAttributes request: %w", err)
	}

	req.Header.Set("x-amz-object-attributes", strings.Join(attributes, ","))

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign GetObjectAttributes request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: GetObjectAttributes request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read GetObjectAttributes response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if awsErr := parseS3Error(respBody); awsErr != nil {
			return nil, fmt.Errorf("s3 client: GetObjectAttributes request failed: %w", awsErr)
		}
		return nil, fmt.Errorf("s3 client: GetObjectAttributes request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result getObjectAttributesResponse
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("s3 client: failed to decode GetObjectAttributes response: %w", err)
	}

	return &ObjectAttributes{
		Bucket:       bucket,
		Key:          key,
		ETag:         cleanETag(result.ETag),
		StorageClass: strings.TrimSpace(result.StorageClass),
		ObjectSize:   result.ObjectSize,
	}, nil
}

// ListObjects returns all objects in a bucket (handles pagination).
func (c *Client) ListObjects(bucket string) ([]Object, error) {
	var objects []Object
	continuationToken := ""

	for {
		query := "?list-type=2&max-keys=1000"
		if continuationToken != "" {
			query += "&continuation-token=" + url.QueryEscape(continuationToken)
		}

		req, err := http.NewRequest(http.MethodGet, c.bucketURL(bucket)+query, nil)
		if err != nil {
			return nil, fmt.Errorf("s3 client: failed to build ListObjects request: %w", err)
		}

		if err := c.signRequest(req, nil); err != nil {
			return nil, fmt.Errorf("s3 client: failed to sign ListObjects request: %w", err)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("s3 client: ListObjects request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("s3 client: failed to read ListObjects response: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if awsErr := parseS3Error(respBody); awsErr != nil {
				return nil, fmt.Errorf("s3 client: ListObjects request failed: %w", awsErr)
			}
			return nil, fmt.Errorf("s3 client: ListObjects request failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		var result listObjectsV2Response
		if err := xml.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("s3 client: failed to decode ListObjects response: %w", err)
		}

		for _, entry := range result.Contents {
			objects = append(objects, Object{
				Key:          strings.TrimSpace(entry.Key),
				Size:         entry.Size,
				ETag:         cleanETag(entry.ETag),
				StorageClass: strings.TrimSpace(entry.StorageClass),
				LastModified: strings.TrimSpace(entry.LastModified),
			})
		}

		if !result.IsTruncated {
			return objects, nil
		}

		continuationToken = strings.TrimSpace(result.NextContinuationToken)
		if continuationToken == "" {
			return objects, nil
		}
	}
}

// DeleteObjects deletes multiple objects from a bucket in a single request.
func (c *Client) DeleteObjects(bucket string, keys []string) (*deleteObjectsResponse, error) {
	identifiers := make([]deleteObjectIdentifier, len(keys))
	for i, key := range keys {
		identifiers[i] = deleteObjectIdentifier{Key: key}
	}

	payload := deleteRequest{
		Quiet:   true,
		Objects: identifiers,
	}

	body, err := xml.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to marshal DeleteObjects request: %w", err)
	}

	reqURL := c.bucketURL(bucket) + "?delete"
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build DeleteObjects request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	hash := md5.Sum(body)
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(hash[:]))

	if err := c.signRequest(req, body); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign DeleteObjects request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: DeleteObjects request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read DeleteObjects response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if awsErr := parseS3Error(respBody); awsErr != nil {
			return nil, fmt.Errorf("s3 client: DeleteObjects request failed: %w", awsErr)
		}
		return nil, fmt.Errorf("s3 client: DeleteObjects request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result deleteObjectsResponse
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("s3 client: failed to decode DeleteObjects response: %w", err)
	}

	return &result, nil
}

// ListBuckets returns all S3 buckets accessible to the account.
func (c *Client) ListBuckets() ([]Bucket, error) {
	endpoint := fmt.Sprintf("https://s3.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build ListBuckets request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign ListBuckets request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: ListBuckets request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read ListBuckets response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if awsErr := parseS3Error(respBody); awsErr != nil {
			return nil, fmt.Errorf("s3 client: ListBuckets request failed: %w", awsErr)
		}
		return nil, fmt.Errorf("s3 client: ListBuckets request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result listBucketsResponse
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("s3 client: failed to decode ListBuckets response: %w", err)
	}

	buckets := make([]Bucket, 0, len(result.Buckets))
	for _, entry := range result.Buckets {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		buckets = append(buckets, Bucket{Name: name})
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

func (c *Client) checkResponse(resp *http.Response, action string) error {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("s3 client: failed to read %s response: %w", action, err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if awsErr := parseS3Error(respBody); awsErr != nil {
		return fmt.Errorf("s3 client: %s request failed: %w", action, awsErr)
	}

	return fmt.Errorf("s3 client: %s request failed with status %d: %s", action, resp.StatusCode, string(respBody))
}

func parseS3Error(body []byte) *common.Error {
	var payload s3ErrorResponse
	if err := xml.Unmarshal(body, &payload); err != nil {
		return nil
	}

	code := strings.TrimSpace(payload.Code)
	message := strings.TrimSpace(payload.Message)
	if code == "" && message == "" {
		return nil
	}

	return &common.Error{Code: code, Message: message}
}

func encodeKey(key string) string {
	segments := strings.Split(key, "/")
	encoded := make([]string, len(segments))
	for i, s := range segments {
		encoded[i] = url.PathEscape(s)
	}
	return strings.Join(encoded, "/")
}

func cleanETag(etag string) string {
	return strings.Trim(strings.TrimSpace(etag), "\"")
}
