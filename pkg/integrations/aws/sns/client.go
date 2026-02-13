package sns

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	snsServiceName = "sns"
	snsAPIVersion  = "2010-03-31"
	snsContentType = "application/x-www-form-urlencoded; charset=utf-8"
)

// Client provides lightweight SNS API operations through signed HTTP requests.
type Client struct {
	http        core.HTTPContext
	region      string
	endpoint    string
	credentials *aws.Credentials
	signer      *v4.Signer
}

// NewClient creates a region-scoped SNS client.
func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	normalizedRegion := strings.TrimSpace(region)
	return &Client{
		http:        httpCtx,
		region:      normalizedRegion,
		endpoint:    fmt.Sprintf("https://sns.%s.amazonaws.com/", normalizedRegion),
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

// GetTopic returns a topic with normalized attributes from GetTopicAttributes.
func (c *Client) GetTopic(topicArn string) (*Topic, error) {
	params := map[string]string{
		"TopicArn": topicArn,
	}

	var response getTopicAttributesResponse
	if err := c.postForm("GetTopicAttributes", params, &response); err != nil {
		return nil, fmt.Errorf("sns client: failed to get topic attributes for %q: %w", topicArn, err)
	}

	attributes := attributeEntriesToMap(response.Entries)
	return &Topic{
		TopicArn:                  topicArn,
		Name:                      topicNameFromArn(topicArn),
		DisplayName:               strings.TrimSpace(attributes["DisplayName"]),
		Owner:                     strings.TrimSpace(attributes["Owner"]),
		KmsMasterKeyID:            strings.TrimSpace(attributes["KmsMasterKeyId"]),
		FifoTopic:                 boolAttribute(attributes, "FifoTopic"),
		ContentBasedDeduplication: boolAttribute(attributes, "ContentBasedDeduplication"),
		Attributes:                attributes,
	}, nil
}

func (c *Client) CreateTopic(name string) (*Topic, error) {
	params := map[string]string{
		"Name": name,
	}

	var response createTopicResponse
	if err := c.postForm("CreateTopic", params, &response); err != nil {
		return nil, fmt.Errorf("sns client: failed to create topic %q: %w", name, err)
	}

	topic, err := c.GetTopic(strings.TrimSpace(response.TopicArn))
	if err != nil {
		return nil, fmt.Errorf("sns client: failed to load created topic %q: %w", name, err)
	}

	return topic, nil
}

// DeleteTopic deletes the topic associated with the provided ARN.
func (c *Client) DeleteTopic(topicArn string) error {
	if err := c.postForm("DeleteTopic", map[string]string{
		"TopicArn": topicArn,
	}, nil); err != nil {
		return fmt.Errorf("sns client: failed to delete topic %q: %w", topicArn, err)
	}

	return nil
}

// PublishMessage publishes a message to a topic and returns publish metadata.
func (c *Client) PublishMessage(parameters PublishMessageParameters) (*PublishResult, error) {
	params := map[string]string{
		"TopicArn": parameters.TopicArn,
		"Message":  parameters.Message,
	}

	if subject := strings.TrimSpace(parameters.Subject); subject != "" {
		params["Subject"] = subject
	}

	for index, key := range sortedKeys(parameters.MessageAttributes) {
		entry := strconv.Itoa(index + 1)
		value := parameters.MessageAttributes[key]
		params["MessageAttributes.entry."+entry+".Name"] = key
		params["MessageAttributes.entry."+entry+".Value.DataType"] = "String"
		params["MessageAttributes.entry."+entry+".Value.StringValue"] = value
	}

	var response publishResponse
	if err := c.postForm("Publish", params, &response); err != nil {
		return nil, fmt.Errorf("sns client: failed to publish message to topic %q: %w", parameters.TopicArn, err)
	}

	return &PublishResult{
		MessageID:      strings.TrimSpace(response.MessageID),
		SequenceNumber: strings.TrimSpace(response.SequenceNumber),
		TopicArn:       parameters.TopicArn,
	}, nil
}

// GetSubscription returns a subscription with normalized attributes.
func (c *Client) GetSubscription(subscriptionArn string) (*Subscription, error) {
	params := map[string]string{
		"SubscriptionArn": subscriptionArn,
	}

	var response getSubscriptionAttributesResponse
	if err := c.postForm("GetSubscriptionAttributes", params, &response); err != nil {
		return nil, fmt.Errorf("sns client: failed to get subscription attributes for %q: %w", subscriptionArn, err)
	}

	attributes := attributeEntriesToMap(response.Entries)
	return &Subscription{
		SubscriptionArn:     subscriptionArn,
		TopicArn:            strings.TrimSpace(attributes["TopicArn"]),
		Protocol:            strings.TrimSpace(attributes["Protocol"]),
		Endpoint:            strings.TrimSpace(attributes["Endpoint"]),
		Owner:               strings.TrimSpace(attributes["Owner"]),
		PendingConfirmation: boolAttribute(attributes, "PendingConfirmation"),
		RawMessageDelivery:  boolAttribute(attributes, "RawMessageDelivery"),
		Attributes:          attributes,
	}, nil
}

// Subscribe creates an SNS subscription and returns the resulting metadata.
func (c *Client) Subscribe(parameters SubscribeParameters) (*Subscription, error) {
	params := map[string]string{
		"TopicArn": parameters.TopicArn,
		"Protocol": parameters.Protocol,
		"Endpoint": parameters.Endpoint,
	}

	if parameters.ReturnSubscriptionARN {
		params["ReturnSubscriptionArn"] = "true"
	}

	for index, key := range sortedKeys(parameters.Attributes) {
		entry := strconv.Itoa(index + 1)
		params["Attributes.entry."+entry+".key"] = key
		params["Attributes.entry."+entry+".value"] = parameters.Attributes[key]
	}

	var response subscribeResponse
	if err := c.postForm("Subscribe", params, &response); err != nil {
		return nil, fmt.Errorf("sns client: failed to subscribe endpoint %q to topic %q: %w", parameters.Endpoint, parameters.TopicArn, err)
	}

	subscriptionArn := strings.TrimSpace(response.SubscriptionArn)
	if strings.EqualFold(subscriptionArn, "pending confirmation") {
		return &Subscription{
			SubscriptionArn:     subscriptionArn,
			TopicArn:            parameters.TopicArn,
			Protocol:            parameters.Protocol,
			Endpoint:            parameters.Endpoint,
			PendingConfirmation: true,
			Attributes:          map[string]string{},
		}, nil
	}

	subscription, err := c.GetSubscription(subscriptionArn)
	if err != nil {
		return nil, fmt.Errorf("sns client: failed to load subscription %q: %w", subscriptionArn, err)
	}

	return subscription, nil
}

// Unsubscribe removes a subscription identified by the provided ARN.
func (c *Client) Unsubscribe(subscriptionArn string) error {
	if err := c.postForm("Unsubscribe", map[string]string{
		"SubscriptionArn": subscriptionArn,
	}, nil); err != nil {
		return fmt.Errorf("sns client: failed to unsubscribe %q: %w", subscriptionArn, err)
	}

	return nil
}

// ListTopics returns all topics in the configured region.
func (c *Client) ListTopics() ([]Topic, error) {
	topics := []Topic{}
	nextToken := ""

	for {
		params := map[string]string{}
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		var response listTopicsResponse
		if err := c.postForm("ListTopics", params, &response); err != nil {
			return nil, fmt.Errorf("failed to list topics in region %q: %w", c.region, err)
		}

		for _, item := range response.Topics {
			topicArn := strings.TrimSpace(item.TopicArn)
			if topicArn == "" {
				continue
			}

			topics = append(topics, Topic{
				TopicArn: topicArn,
				Name:     topicNameFromArn(topicArn),
			})
		}

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			return topics, nil
		}
	}
}

func (c *Client) ListSubscriptionsByTopic(topicArn string) ([]Subscription, error) {
	subscriptions := []Subscription{}
	nextToken := ""
	baseParams := map[string]string{
		"TopicArn": topicArn,
	}

	for {
		params := map[string]string{}
		for key, value := range baseParams {
			params[key] = value
		}
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		var response listSubscriptionsResponse
		if err := c.postForm("ListSubscriptionsByTopic", params, &response); err != nil {
			return nil, fmt.Errorf("failed to list subscriptions: %w", err)
		}

		for _, item := range response.SubscriptionsTopic {
			subscriptions = append(subscriptions, Subscription{
				SubscriptionArn: item.SubscriptionArn,
				TopicArn:        topicArn,
			})
		}

		if response.NextToken == "" {
			return subscriptions, nil
		}
	}
}

// postForm sends a signed SNS query request and decodes XML responses.
func (c *Client) postForm(action string, params map[string]string, out any) error {
	values := url.Values{}
	values.Set("Action", action)
	values.Set("Version", snsAPIVersion)
	for key, value := range params {
		values.Set(key, value)
	}

	body := values.Encode()
	request, err := http.NewRequest(http.MethodPost, c.endpoint, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("sns client: failed to build %s request: %w", action, err)
	}

	request.Header.Set("Content-Type", snsContentType)

	if err := c.signRequest(request, []byte(body)); err != nil {
		return fmt.Errorf("sns client: failed to sign %s request: %w", action, err)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("sns client: %s request failed: %w", action, err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("sns client: failed to read %s response body: %w", action, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if awsErr := parseSNSError(responseBody); awsErr != nil {
			return fmt.Errorf("sns client: %s request failed: %w", action, awsErr)
		}
		return fmt.Errorf("sns client: %s request failed with status %d: %s", action, response.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := xml.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("sns client: failed to decode %s response: %w", action, err)
	}

	return nil
}

// signRequest signs a request using SigV4 for SNS.
func (c *Client) signRequest(request *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, request, payloadHash, snsServiceName, c.region, time.Now())
}

// attributeEntriesToMap converts XML attribute entries into a normalized map.
func attributeEntriesToMap(entries []attributeEntry) map[string]string {
	attributes := make(map[string]string, len(entries))
	for _, entry := range entries {
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			continue
		}
		attributes[key] = strings.TrimSpace(entry.Value)
	}
	return attributes
}

// sortedKeys returns sorted keys for deterministic query-parameter generation.
func sortedKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	var keys []string
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// parseSNSError extracts AWS error information from SNS XML responses.
func parseSNSError(body []byte) *common.Error {
	var payload snsErrorPayload
	if err := xml.Unmarshal(body, &payload); err != nil {
		return nil
	}

	code := strings.TrimSpace(payload.Error.Code)
	message := strings.TrimSpace(payload.Error.Message)
	if code == "" && message == "" {
		return nil
	}

	return &common.Error{Code: code, Message: message}
}

func topicNameFromArn(topicArn string) string {
	parts := strings.Split(strings.TrimSpace(topicArn), ":")
	if len(parts) == 0 {
		return strings.TrimSpace(topicArn)
	}

	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return strings.TrimSpace(topicArn)
	}

	return name
}

func boolAttribute(attributes map[string]string, key string) bool {
	value, ok := attributes[key]
	if !ok {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(value), "true")
}
