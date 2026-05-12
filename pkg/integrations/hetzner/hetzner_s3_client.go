package hetzner

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

const s3Service = "s3"

// emptyBodyHash is the SHA-256 hash of an empty body, used when there is no request body.
const emptyBodyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

type HetznerS3Client struct {
	accessKeyID     string
	secretAccessKey string
	region          string
	endpoint        string
	http            core.HTTPContext
	signer          *v4.Signer
}

func NewHetznerS3Client(httpCtx core.HTTPContext, integration core.IntegrationContext) (*HetznerS3Client, error) {
	accessKeyID, err := integration.GetConfig("s3AccessKeyId")
	if err != nil || strings.TrimSpace(string(accessKeyID)) == "" {
		return nil, fmt.Errorf("s3AccessKeyId is required")
	}
	secretKey, err := integration.GetConfig("s3SecretAccessKey")
	if err != nil || strings.TrimSpace(string(secretKey)) == "" {
		return nil, fmt.Errorf("s3SecretAccessKey is required")
	}
	region, err := integration.GetConfig("s3Region")
	if err != nil || strings.TrimSpace(string(region)) == "" {
		return nil, fmt.Errorf("s3Region is required")
	}
	regionStr := strings.TrimSpace(string(region))
	return &HetznerS3Client{
		accessKeyID:     strings.TrimSpace(string(accessKeyID)),
		secretAccessKey: strings.TrimSpace(string(secretKey)),
		region:          regionStr,
		endpoint:        fmt.Sprintf("https://%s.your-objectstorage.com", regionStr),
		http:            httpCtx,
		signer:          v4.NewSigner(),
	}, nil
}

func (c *HetznerS3Client) credentials() aws.Credentials {
	return aws.Credentials{
		AccessKeyID:     c.accessKeyID,
		SecretAccessKey: c.secretAccessKey,
	}
}

func (c *HetznerS3Client) buildURL(bucket, key string, query url.Values) string {
	path := "/"
	if bucket != "" {
		path += url.PathEscape(bucket)
		if key != "" {
			path += "/" + encodeKeyPath(strings.TrimPrefix(key, "/"))
		}
	}
	rawURL := c.endpoint + path
	if len(query) > 0 {
		rawURL += "?" + query.Encode()
	}
	return rawURL
}

// encodeKeyPath URL-encodes each path segment of an S3 key individually,
// preserving the / separators so that multi-part keys remain valid URL paths.
func encodeKeyPath(key string) string {
	segments := strings.Split(key, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

func bodyHash(body []byte) string {
	if body == nil {
		return emptyBodyHash
	}
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

func (c *HetznerS3Client) doRequest(method, bucket, key string, headers map[string]string, body []byte, query url.Values) (*http.Response, error) {
	rawURL := c.buildURL(bucket, key, query)
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// Ceph (used by Hetzner Object Storage) requires X-Amz-Content-SHA256 to be
	// present as a signed header. Set it before signing so the signer includes it.
	hash := bodyHash(body)
	req.Header.Set("X-Amz-Content-SHA256", hash)
	creds := c.credentials()
	if err := c.signer.SignHTTP(context.Background(), creds, req, hash, s3Service, c.region, time.Now()); err != nil {
		return nil, fmt.Errorf("sign S3 request: %w", err)
	}
	return c.http.Do(req)
}

func (c *HetznerS3Client) parseS3Error(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var s3Err struct {
		XMLName xml.Name `xml:"Error"`
		Code    string   `xml:"Code"`
		Message string   `xml:"Message"`
	}
	if xmlErr := xml.Unmarshal(body, &s3Err); xmlErr == nil && s3Err.Message != "" {
		return fmt.Errorf("S3 error %d (%s): %s", resp.StatusCode, s3Err.Code, s3Err.Message)
	}
	return fmt.Errorf("S3 error %d: %s", resp.StatusCode, string(body))
}

func (c *HetznerS3Client) PresignURL(bucket, key, method string, expiresIn time.Duration) (string, error) {
	rawURL := c.buildURL(bucket, key, nil)
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return "", err
	}

	// Set X-Amz-Expires so the signer embeds the expiry in the presigned URL.
	q := req.URL.Query()
	q.Set("X-Amz-Expires", fmt.Sprintf("%d", int(expiresIn.Seconds())))
	req.URL.RawQuery = q.Encode()

	creds := c.credentials()
	signedURL, _, err := c.signer.PresignHTTP(context.Background(), creds, req, "UNSIGNED-PAYLOAD", s3Service, c.region, time.Now())
	if err != nil {
		return "", fmt.Errorf("presign URL: %w", err)
	}
	return signedURL, nil
}

// Bucket operations

func (c *HetznerS3Client) CreateBucket(bucket string) error {
	resp, err := c.doRequest(http.MethodPut, bucket, "", nil, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseS3Error(resp)
	}
	return nil
}

func (c *HetznerS3Client) DeleteBucket(bucket string) error {
	resp, err := c.doRequest(http.MethodDelete, bucket, "", nil, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseS3Error(resp)
	}
	return nil
}

type HetznerS3Bucket struct {
	Name string `xml:"Name"`
}

func (c *HetznerS3Client) ListBuckets() ([]HetznerS3Bucket, error) {
	resp, err := c.doRequest(http.MethodGet, "", "", nil, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseS3Error(resp)
	}
	var out struct {
		XMLName xml.Name          `xml:"ListAllMyBucketsResult"`
		Buckets []HetznerS3Bucket `xml:"Buckets>Bucket"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode list buckets response: %w", err)
	}
	return out.Buckets, nil
}

// Object operations

func (c *HetznerS3Client) PutObject(bucket, key, contentType string, body []byte) (etag string, err error) {
	headers := map[string]string{}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	resp, err := c.doRequest(http.MethodPut, bucket, key, headers, body, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", c.parseS3Error(resp)
	}
	return resp.Header.Get("ETag"), nil
}

type S3Object struct {
	Body        []byte
	ContentType string
	Size        int64
}

func (c *HetznerS3Client) GetObject(bucket, key string) (*S3Object, error) {
	resp, err := c.doRequest(http.MethodGet, bucket, key, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseS3Error(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read object body: %w", err)
	}
	return &S3Object{
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
		Size:        resp.ContentLength,
	}, nil
}

func (c *HetznerS3Client) DeleteObject(bucket, key string) error {
	resp, err := c.doRequest(http.MethodDelete, bucket, key, nil, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseS3Error(resp)
	}
	return nil
}

type S3ListItem struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
}

func (c *HetznerS3Client) ListObjects(bucket, prefix string, maxKeys int) ([]S3ListItem, error) {
	q := url.Values{}
	q.Set("list-type", "2")
	if prefix != "" {
		q.Set("prefix", prefix)
	}
	if maxKeys > 0 {
		q.Set("max-keys", fmt.Sprintf("%d", maxKeys))
	}
	resp, err := c.doRequest(http.MethodGet, bucket, "", nil, nil, q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseS3Error(resp)
	}
	var out struct {
		XMLName  xml.Name     `xml:"ListBucketResult"`
		Contents []S3ListItem `xml:"Contents"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode list objects response: %w", err)
	}
	return out.Contents, nil
}
