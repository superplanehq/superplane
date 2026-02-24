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
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	s3ServiceName = "s3"
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

func (c *Client) endpoint(bucket string) string {
	if bucket == "" {
		return fmt.Sprintf("https://s3.%s.amazonaws.com", c.region)
	}
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s", c.region, bucket)
}

func (c *Client) objectURL(bucket, key string) string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", c.region, bucket, url.PathEscape(key))
}

func (c *Client) CreateBucket(bucket string) error {
	body := []byte{}
	if c.region != "us-east-1" {
		config := createBucketConfiguration{
			XMLNS:              "http://s3.amazonaws.com/doc/2006-03-01/",
			LocationConstraint: c.region,
		}
		var err error
		body, err = xml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal create bucket configuration: %w", err)
		}
	}

	req, err := http.NewRequest(http.MethodPut, c.endpoint(bucket), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build create bucket request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/xml")
	}

	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("create bucket request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read create bucket response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("S3 CreateBucket failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return nil
}

func (c *Client) DeleteBucket(bucket string) error {
	req, err := http.NewRequest(http.MethodDelete, c.endpoint(bucket), nil)
	if err != nil {
		return fmt.Errorf("failed to build delete bucket request: %w", err)
	}

	if err := c.signRequest(req, []byte{}); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("delete bucket request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read delete bucket response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("S3 DeleteBucket failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return nil
}

func (c *Client) HeadBucket(bucket string) (*HeadBucketResult, error) {
	req, err := http.NewRequest(http.MethodHead, c.endpoint(bucket), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build head bucket request: %w", err)
	}

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("head bucket request failed: %w", err)
	}
	defer res.Body.Close()
	io.ReadAll(res.Body) //nolint:errcheck

	if res.StatusCode == http.StatusNotFound {
		return &HeadBucketResult{Exists: false}, nil
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 HeadBucket failed with %d", res.StatusCode)
	}

	return &HeadBucketResult{
		Exists: true,
		Region: res.Header.Get("x-amz-bucket-region"),
	}, nil
}

func (c *Client) HeadObject(bucket, key string) (*HeadObjectResult, error) {
	req, err := http.NewRequest(http.MethodHead, c.objectURL(bucket, key), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build head object request: %w", err)
	}

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("head object request failed: %w", err)
	}
	defer res.Body.Close()
	io.ReadAll(res.Body) //nolint:errcheck

	if res.StatusCode == http.StatusNotFound {
		return nil, &common.Error{Code: "NotFound", Message: "object not found"}
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("S3 HeadObject failed with %d", res.StatusCode)
	}

	return &HeadObjectResult{
		ContentLength: res.Header.Get("Content-Length"),
		ContentType:   res.Header.Get("Content-Type"),
		ETag:          res.Header.Get("ETag"),
		LastModified:  res.Header.Get("Last-Modified"),
	}, nil
}

func (c *Client) CopyObject(sourceBucket, sourceKey, destBucket, destKey string) (*CopyObjectResult, error) {
	copySource := fmt.Sprintf("/%s/%s", sourceBucket, url.PathEscape(sourceKey))

	req, err := http.NewRequest(http.MethodPut, c.objectURL(destBucket, destKey), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build copy object request: %w", err)
	}

	req.Header.Set("x-amz-copy-source", copySource)

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("copy object request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read copy object response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("S3 CopyObject failed with %d: %s", res.StatusCode, string(responseBody))
	}

	var result copyObjectResponse
	if err := xml.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode copy object response: %w", err)
	}

	return &CopyObjectResult{
		ETag:         result.ETag,
		LastModified: result.LastModified,
	}, nil
}

func (c *Client) DeleteObject(bucket, key string) error {
	req, err := http.NewRequest(http.MethodDelete, c.objectURL(bucket, key), nil)
	if err != nil {
		return fmt.Errorf("failed to build delete object request: %w", err)
	}

	if err := c.signRequest(req, []byte{}); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("delete object request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read delete object response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("S3 DeleteObject failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return nil
}

func (c *Client) PutObject(bucket, key, contentType string, body []byte) (*PutObjectResult, error) {
	req, err := http.NewRequest(http.MethodPut, c.objectURL(bucket, key), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build put object request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if err := c.signRequest(req, body); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("put object request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read put object response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("S3 PutObject failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return &PutObjectResult{
		ETag: res.Header.Get("ETag"),
	}, nil
}

func (c *Client) GetObjectAttributes(bucket, key string, attributes []string) (*GetObjectAttributesResult, error) {
	req, err := http.NewRequest(http.MethodHead, c.objectURL(bucket, key)+"?attributes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build get object attributes request: %w", err)
	}

	req.Header.Set("x-amz-object-attributes", strings.Join(attributes, ","))

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get object attributes request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read get object attributes response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("S3 GetObjectAttributes failed with %d: %s", res.StatusCode, string(responseBody))
	}

	var result getObjectAttributesResponse
	if err := xml.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode get object attributes response: %w", err)
	}

	return &GetObjectAttributesResult{
		ETag:         result.ETag,
		ObjectSize:   result.ObjectSize,
		StorageClass: result.StorageClass,
		LastModified: res.Header.Get("Last-Modified"),
	}, nil
}

func (c *Client) ListObjectsV2(bucket, continuationToken string) (*ListObjectsResult, error) {
	queryURL := fmt.Sprintf("%s?list-type=2&max-keys=1000", c.endpoint(bucket))
	if continuationToken != "" {
		queryURL += "&continuation-token=" + url.QueryEscape(continuationToken)
	}

	req, err := http.NewRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build list objects request: %w", err)
	}

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list objects request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read list objects response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("S3 ListObjectsV2 failed with %d: %s", res.StatusCode, string(responseBody))
	}

	var result listObjectsV2Response
	if err := xml.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode list objects response: %w", err)
	}

	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		keys = append(keys, obj.Key)
	}

	return &ListObjectsResult{
		Keys:                  keys,
		IsTruncated:           result.IsTruncated,
		NextContinuationToken: result.NextContinuationToken,
	}, nil
}

func (c *Client) DeleteObjects(bucket string, keys []string) error {
	objects := make([]deleteObjectEntry, 0, len(keys))
	for _, key := range keys {
		objects = append(objects, deleteObjectEntry{Key: key})
	}

	deleteReq := deleteObjectsRequest{
		Quiet:   true,
		Objects: objects,
	}

	body, err := xml.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal delete objects request: %w", err)
	}

	reqURL := c.endpoint(bucket) + "?delete"
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build delete objects request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")

	hash := sha256.Sum256(body)
	req.Header.Set("x-amz-content-sha256", hex.EncodeToString(hash[:]))

	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("delete objects request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read delete objects response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("S3 DeleteObjects failed with %d: %s", res.StatusCode, string(responseBody))
	}

	return nil
}

func (c *Client) ListBuckets() ([]BucketInfo, error) {
	req, err := http.NewRequest(http.MethodGet, c.endpoint(""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build list buckets request: %w", err)
	}

	if err := c.signRequest(req, []byte{}); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list buckets request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read list buckets response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := parseS3Error(responseBody); awsErr != nil {
			return nil, awsErr
		}
		return nil, fmt.Errorf("S3 ListBuckets failed with %d: %s", res.StatusCode, string(responseBody))
	}

	var result listBucketsResponse
	if err := xml.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode list buckets response: %w", err)
	}

	buckets := make([]BucketInfo, 0, len(result.Buckets))
	for _, b := range result.Buckets {
		buckets = append(buckets, BucketInfo{
			Name:         b.Name,
			CreationDate: b.CreationDate,
		})
	}

	return buckets, nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, s3ServiceName, c.region, time.Now())
}

func parseS3Error(body []byte) *common.Error {
	var errResp s3ErrorResponse
	if err := xml.Unmarshal(body, &errResp); err != nil {
		return nil
	}

	code := strings.TrimSpace(errResp.Code)
	message := strings.TrimSpace(errResp.Message)
	if code == "" && message == "" {
		return nil
	}

	return &common.Error{Code: code, Message: message}
}

// XML types

type createBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	XMLNS              string   `xml:"xmlns,attr"`
	LocationConstraint string   `xml:"LocationConstraint"`
}

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

type copyObjectResponse struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	ETag         string   `xml:"ETag"`
	LastModified string   `xml:"LastModified"`
}

type getObjectAttributesResponse struct {
	XMLName      xml.Name `xml:"GetObjectAttributesResponse"`
	ETag         string   `xml:"ETag"`
	ObjectSize   int64    `xml:"ObjectSize"`
	StorageClass string   `xml:"StorageClass"`
}

type listObjectsV2Response struct {
	XMLName               xml.Name        `xml:"ListBucketResult"`
	Contents              []objectContent `xml:"Contents"`
	IsTruncated           bool            `xml:"IsTruncated"`
	NextContinuationToken string          `xml:"NextContinuationToken"`
}

type objectContent struct {
	Key string `xml:"Key"`
}

type deleteObjectsRequest struct {
	XMLName xml.Name            `xml:"Delete"`
	Quiet   bool                `xml:"Quiet"`
	Objects []deleteObjectEntry `xml:"Object"`
}

type deleteObjectEntry struct {
	Key string `xml:"Key"`
}

type listBucketsResponse struct {
	XMLName xml.Name     `xml:"ListAllMyBucketsResult"`
	Buckets []bucketInfo `xml:"Buckets>Bucket"`
}

type bucketInfo struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

// Result types

type HeadBucketResult struct {
	Exists bool   `json:"exists"`
	Region string `json:"region,omitempty"`
}

type HeadObjectResult struct {
	ContentLength string `json:"contentLength"`
	ContentType   string `json:"contentType"`
	ETag          string `json:"etag"`
	LastModified  string `json:"lastModified"`
}

type CopyObjectResult struct {
	ETag         string `json:"etag"`
	LastModified string `json:"lastModified"`
}

type PutObjectResult struct {
	ETag string `json:"etag"`
}

type GetObjectAttributesResult struct {
	ETag         string `json:"etag"`
	ObjectSize   int64  `json:"objectSize"`
	StorageClass string `json:"storageClass"`
	LastModified string `json:"lastModified"`
}

type ListObjectsResult struct {
	Keys                  []string
	IsTruncated           bool
	NextContinuationToken string
}

type BucketInfo struct {
	Name         string `json:"name"`
	CreationDate string `json:"creationDate"`
}
