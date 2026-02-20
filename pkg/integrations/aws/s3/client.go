package s3

import (
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
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
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

func (c *Client) CreateBucket(name string) (*Bucket, error) {
	var body io.Reader
	var bodyBytes []byte

	if c.region != "us-east-1" {
		config := createBucketConfiguration{LocationConstraint: c.region}
		data, err := xml.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("s3 client: failed to marshal CreateBucketConfiguration: %w", err)
		}
		bodyBytes = data
		body = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(http.MethodPut, c.bucketURL(name), body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build CreateBucket request: %w", err)
	}

	if err := c.signRequest(req, bodyBytes); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign CreateBucket request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: CreateBucket request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read CreateBucket response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, parseS3ErrorOrDefault("CreateBucket", res.StatusCode, responseBody)
	}

	location := strings.TrimSpace(res.Header.Get("Location"))
	return &Bucket{
		Name:     name,
		Region:   c.region,
		Location: location,
	}, nil
}

func (c *Client) DeleteBucket(name string) error {
	req, err := http.NewRequest(http.MethodDelete, c.bucketURL(name), nil)
	if err != nil {
		return fmt.Errorf("s3 client: failed to build DeleteBucket request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return fmt.Errorf("s3 client: failed to sign DeleteBucket request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("s3 client: DeleteBucket request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("s3 client: failed to read DeleteBucket response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return parseS3ErrorOrDefault("DeleteBucket", res.StatusCode, responseBody)
	}

	return nil
}

func (c *Client) HeadBucket(name string) (*HeadBucketResult, error) {
	req, err := http.NewRequest(http.MethodHead, c.bucketURL(name), nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build HeadBucket request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign HeadBucket request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: HeadBucket request failed: %w", err)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("s3 client: HeadBucket request failed with status %d", res.StatusCode)
	}

	return &HeadBucketResult{
		BucketName:    name,
		Region:        strings.TrimSpace(res.Header.Get("X-Amz-Bucket-Region")),
		AccessPointID: strings.TrimSpace(res.Header.Get("X-Amz-Access-Point-Alias")),
	}, nil
}

func (c *Client) PutObject(bucket, key string, body []byte, contentType string) (*PutObjectResult, error) {
	objectURL := c.objectURL(bucket, key)
	req, err := http.NewRequest(http.MethodPut, objectURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build PutObject request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if err := c.signRequest(req, body); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign PutObject request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: PutObject request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read PutObject response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, parseS3ErrorOrDefault("PutObject", res.StatusCode, responseBody)
	}

	return &PutObjectResult{
		Bucket: bucket,
		Key:    key,
		ETag:   strings.Trim(strings.TrimSpace(res.Header.Get("ETag")), "\""),
	}, nil
}

func (c *Client) CopyObject(sourceBucket, sourceKey, destBucket, destKey string) (*CopyObjectResult, error) {
	objectURL := c.objectURL(destBucket, destKey)
	req, err := http.NewRequest(http.MethodPut, objectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build CopyObject request: %w", err)
	}

	req.Header.Set("X-Amz-Copy-Source", fmt.Sprintf("/%s/%s", sourceBucket, sourceKey))

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign CopyObject request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: CopyObject request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read CopyObject response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, parseS3ErrorOrDefault("CopyObject", res.StatusCode, responseBody)
	}

	var copyResult copyObjectResponse
	if err := xml.Unmarshal(responseBody, &copyResult); err != nil {
		return nil, fmt.Errorf("s3 client: failed to decode CopyObject response: %w", err)
	}

	return &CopyObjectResult{
		SourceBucket: sourceBucket,
		SourceKey:    sourceKey,
		Bucket:       destBucket,
		Key:          destKey,
		ETag:         strings.Trim(strings.TrimSpace(copyResult.ETag), "\""),
		LastModified: strings.TrimSpace(copyResult.LastModified),
	}, nil
}

func (c *Client) DeleteObject(bucket, key string) error {
	objectURL := c.objectURL(bucket, key)
	req, err := http.NewRequest(http.MethodDelete, objectURL, nil)
	if err != nil {
		return fmt.Errorf("s3 client: failed to build DeleteObject request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return fmt.Errorf("s3 client: failed to sign DeleteObject request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("s3 client: DeleteObject request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("s3 client: failed to read DeleteObject response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return parseS3ErrorOrDefault("DeleteObject", res.StatusCode, responseBody)
	}

	return nil
}

func (c *Client) HeadObject(bucket, key string) (*HeadObjectResult, error) {
	objectURL := c.objectURL(bucket, key)
	req, err := http.NewRequest(http.MethodHead, objectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build HeadObject request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign HeadObject request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: HeadObject request failed: %w", err)
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("s3 client: HeadObject request failed with status %d", res.StatusCode)
	}

	contentLength := strings.TrimSpace(res.Header.Get("Content-Length"))
	return &HeadObjectResult{
		Bucket:        bucket,
		Key:           key,
		ContentLength: contentLength,
		ContentType:   strings.TrimSpace(res.Header.Get("Content-Type")),
		ETag:          strings.Trim(strings.TrimSpace(res.Header.Get("ETag")), "\""),
		LastModified:  strings.TrimSpace(res.Header.Get("Last-Modified")),
		StorageClass:  strings.TrimSpace(res.Header.Get("X-Amz-Storage-Class")),
	}, nil
}

func (c *Client) GetObjectAttributes(bucket, key string, attributes []string) (*ObjectAttributes, error) {
	objectURL := c.objectURL(bucket, key) + "?attributes"
	req, err := http.NewRequest(http.MethodGet, objectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build GetObjectAttributes request: %w", err)
	}

	req.Header.Set("X-Amz-Object-Attributes", strings.Join(attributes, ","))

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign GetObjectAttributes request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: GetObjectAttributes request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read GetObjectAttributes response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, parseS3ErrorOrDefault("GetObjectAttributes", res.StatusCode, responseBody)
	}

	var result getObjectAttributesResponse
	if err := xml.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("s3 client: failed to decode GetObjectAttributes response: %w", err)
	}

	return &ObjectAttributes{
		Bucket:       bucket,
		Key:          key,
		ETag:         strings.Trim(strings.TrimSpace(result.ETag), "\""),
		StorageClass: strings.TrimSpace(result.StorageClass),
		ObjectSize:   result.ObjectSize,
	}, nil
}

func (c *Client) ListObjects(bucket string) ([]ObjectSummary, error) {
	objects := []ObjectSummary{}
	continuationToken := ""

	for {
		listURL := c.bucketURL(bucket) + "?list-type=2&max-keys=1000"
		if continuationToken != "" {
			listURL += "&continuation-token=" + continuationToken
		}

		req, err := http.NewRequest(http.MethodGet, listURL, nil)
		if err != nil {
			return nil, fmt.Errorf("s3 client: failed to build ListObjectsV2 request: %w", err)
		}

		if err := c.signRequest(req, nil); err != nil {
			return nil, fmt.Errorf("s3 client: failed to sign ListObjectsV2 request: %w", err)
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("s3 client: ListObjectsV2 request failed: %w", err)
		}

		responseBody, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("s3 client: failed to read ListObjectsV2 response: %w", err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, parseS3ErrorOrDefault("ListObjectsV2", res.StatusCode, responseBody)
		}

		var result listObjectsV2Response
		if err := xml.Unmarshal(responseBody, &result); err != nil {
			return nil, fmt.Errorf("s3 client: failed to decode ListObjectsV2 response: %w", err)
		}

		for _, item := range result.Contents {
			objects = append(objects, ObjectSummary{
				Key:  strings.TrimSpace(item.Key),
				ETag: strings.Trim(strings.TrimSpace(item.ETag), "\""),
				Size: item.Size,
			})
		}

		if !result.IsTruncated {
			break
		}

		continuationToken = strings.TrimSpace(result.NextContinuationToken)
		if continuationToken == "" {
			break
		}
	}

	return objects, nil
}

func (c *Client) DeleteObjects(bucket string, keys []string) error {
	type deleteKey struct {
		Key string `xml:"Key"`
	}
	type deletePayload struct {
		XMLName xml.Name    `xml:"Delete"`
		Objects []deleteKey `xml:"Object"`
		Quiet   bool        `xml:"Quiet"`
	}

	payload := deletePayload{Quiet: true}
	for _, key := range keys {
		payload.Objects = append(payload.Objects, deleteKey{Key: key})
	}

	body, err := xml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("s3 client: failed to marshal DeleteObjects payload: %w", err)
	}

	deleteURL := c.bucketURL(bucket) + "?delete"
	req, err := http.NewRequest(http.MethodPost, deleteURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("s3 client: failed to build DeleteObjects request: %w", err)
	}

	hash := sha256.Sum256(body)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(hash[:]))

	if err := c.signRequest(req, body); err != nil {
		return fmt.Errorf("s3 client: failed to sign DeleteObjects request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("s3 client: DeleteObjects request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("s3 client: failed to read DeleteObjects response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return parseS3ErrorOrDefault("DeleteObjects", res.StatusCode, responseBody)
	}

	return nil
}

func (c *Client) ListBuckets() ([]BucketSummary, error) {
	req, err := http.NewRequest(http.MethodGet, c.serviceURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to build ListBuckets request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("s3 client: failed to sign ListBuckets request: %w", err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 client: ListBuckets request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 client: failed to read ListBuckets response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, parseS3ErrorOrDefault("ListBuckets", res.StatusCode, responseBody)
	}

	var result listBucketsResponse
	if err := xml.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("s3 client: failed to decode ListBuckets response: %w", err)
	}

	buckets := make([]BucketSummary, 0, len(result.Buckets))
	for _, b := range result.Buckets {
		buckets = append(buckets, BucketSummary{
			Name:         strings.TrimSpace(b.Name),
			CreationDate: strings.TrimSpace(b.CreationDate),
		})
	}

	return buckets, nil
}

func (c *Client) bucketURL(bucket string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, c.region)
}

func (c *Client) objectURL(bucket, key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, c.region, key)
}

func (c *Client) serviceURL() string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com", c.region)
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, s3ServiceName, c.region, time.Now())
}

func parseS3ErrorOrDefault(action string, statusCode int, body []byte) error {
	if awsErr := parseS3Error(body); awsErr != nil {
		return fmt.Errorf("s3 client: %s request failed: %w", action, awsErr)
	}
	return fmt.Errorf("s3 client: %s request failed with status %d: %s", action, statusCode, string(body))
}

func parseS3Error(body []byte) *common.Error {
	var payload s3ErrorPayload
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
