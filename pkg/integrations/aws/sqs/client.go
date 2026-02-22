package sqs

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

type Queue struct {
	URL  string
	Name string
}

type listQueuesResponse struct {
	XMLName xml.Name         `xml:"ListQueuesResponse"`
	Result  listQueuesResult `xml:"ListQueuesResult"`
}

type listQueuesResult struct {
	QueueURLs []string `xml:"QueueUrl"`
}

type getQueueAttributesResponse struct {
	XMLName xml.Name                 `xml:"GetQueueAttributesResponse"`
	Result  getQueueAttributesResult `xml:"GetQueueAttributesResult"`
}

type getQueueAttributesResult struct {
	Attributes []queueAttribute `xml:"Attribute"`
}

type queueAttribute struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

type sendMessageResponse struct {
	XMLName xml.Name          `xml:"SendMessageResponse"`
	Result  sendMessageResult `xml:"SendMessageResult"`
}

type sendMessageResult struct {
	MessageID string `xml:"MessageId"`
}

type createQueueResponse struct {
	XMLName xml.Name          `xml:"CreateQueueResponse"`
	Result  createQueueResult `xml:"CreateQueueResult"`
}

type createQueueResult struct {
	QueueURL string `xml:"QueueUrl"`
}

func (c *Client) endpoint() string {
	return fmt.Sprintf("https://sqs.%s.amazonaws.com/", c.region)
}

func (c *Client) ListQueues(prefix string) ([]Queue, error) {
	params := url.Values{}
	params.Set("Action", "ListQueues")
	params.Set("Version", "2012-11-05")
	if strings.TrimSpace(prefix) != "" {
		params.Set("QueueNamePrefix", strings.TrimSpace(prefix))
	}

	body, err := c.postForm(c.endpoint(), params)
	if err != nil {
		return nil, err
	}

	var resp listQueuesResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode ListQueues response: %w", err)
	}

	queues := make([]Queue, 0, len(resp.Result.QueueURLs))
	for _, queueURL := range resp.Result.QueueURLs {
		name := queueNameFromURL(queueURL)
		queues = append(queues, Queue{
			URL:  queueURL,
			Name: name,
		})
	}

	return queues, nil
}

func (c *Client) GetQueueAttributes(queueURL string) (map[string]string, error) {
	queueURL = strings.TrimSpace(queueURL)
	params := url.Values{}
	params.Set("Action", "GetQueueAttributes")
	params.Set("Version", "2012-11-05")
	params.Set("AttributeName", "All")

	body, err := c.postForm(queueURL, params)
	if err != nil {
		return nil, err
	}

	var resp getQueueAttributesResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode GetQueueAttributes response: %w", err)
	}

	attributes := map[string]string{}
	for _, attr := range resp.Result.Attributes {
		attributes[attr.Name] = attr.Value
	}

	return attributes, nil
}

func (c *Client) SendMessage(queueURL string, messageBody string) (string, error) {
	queueURL = strings.TrimSpace(queueURL)
	params := url.Values{}
	params.Set("Action", "SendMessage")
	params.Set("Version", "2012-11-05")
	params.Set("MessageBody", messageBody)

	body, err := c.postForm(queueURL, params)
	if err != nil {
		return "", err
	}

	var resp sendMessageResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to decode SendMessage response: %w", err)
	}

	return strings.TrimSpace(resp.Result.MessageID), nil
}

func (c *Client) CreateQueue(name string, attributes map[string]string) (string, error) {
	params := url.Values{}
	params.Set("Action", "CreateQueue")
	params.Set("Version", "2012-11-05")
	params.Set("QueueName", strings.TrimSpace(name))

	i := 1
	for key, value := range attributes {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		params.Set(fmt.Sprintf("Attribute.%d.Name", i), key)
		params.Set(fmt.Sprintf("Attribute.%d.Value", i), value)
		i++
	}

	body, err := c.postForm(c.endpoint(), params)
	if err != nil {
		return "", err
	}

	var resp createQueueResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to decode CreateQueue response: %w", err)
	}

	return strings.TrimSpace(resp.Result.QueueURL), nil
}

func (c *Client) DeleteQueue(queueURL string) error {
	queueURL = strings.TrimSpace(queueURL)
	params := url.Values{}
	params.Set("Action", "DeleteQueue")
	params.Set("Version", "2012-11-05")

	_, err := c.postForm(queueURL, params)
	return err
}

func (c *Client) PurgeQueue(queueURL string) error {
	queueURL = strings.TrimSpace(queueURL)
	params := url.Values{}
	params.Set("Action", "PurgeQueue")
	params.Set("Version", "2012-11-05")

	_, err := c.postForm(queueURL, params)
	return err
}

func (c *Client) postForm(endpoint string, params url.Values) ([]byte, error) {
	bodyBytes := []byte(params.Encode())

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build SQS request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err := c.signRequest(req, bodyBytes); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SQS request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SQS response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("SQS API request failed with %d: %s", res.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "sqs", c.region, time.Now())
}

func queueNameFromURL(queueURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(queueURL))
	if err != nil {
		return strings.TrimSpace(queueURL)
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 {
		return strings.TrimSpace(queueURL)
	}

	return parts[len(parts)-1]
}
