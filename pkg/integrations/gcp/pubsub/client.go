package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const (
	pubsubBaseURL  = "https://pubsub.googleapis.com/v1"
	loggingBaseURL = "https://logging.googleapis.com/v2"
)

// --- Pub/Sub Topic ---

func CreateTopic(ctx context.Context, client *common.Client, projectID, topicID string) error {
	url := fmt.Sprintf("%s/projects/%s/topics/%s", pubsubBaseURL, projectID, topicID)
	_, err := client.ExecRequest(ctx, "PUT", url, nil)
	if err != nil {
		if common.IsAlreadyExistsError(err) {
			return nil
		}
		return err
	}
	return nil
}

func DeleteTopic(ctx context.Context, client *common.Client, projectID, topicID string) error {
	url := fmt.Sprintf("%s/projects/%s/topics/%s", pubsubBaseURL, projectID, topicID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

type listTopicsResponse struct {
	Topics        []TopicResource `json:"topics"`
	NextPageToken string          `json:"nextPageToken"`
}

type TopicResource struct {
	Name string `json:"name"`
}

func ListTopics(ctx context.Context, client *common.Client, projectID string) ([]TopicResource, error) {
	var all []TopicResource
	pageToken := ""
	for {
		u := fmt.Sprintf("%s/projects/%s/topics", pubsubBaseURL, projectID)
		if pageToken != "" {
			u += "?pageToken=" + pageToken
		}
		body, err := client.GetURL(ctx, u)
		if err != nil {
			return nil, err
		}
		var resp listTopicsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse list topics response: %w", err)
		}
		all = append(all, resp.Topics...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return all, nil
}

func TopicShortName(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

// --- Pub/Sub Publish ---

type publishRequest struct {
	Messages []pubsubMessage `json:"messages"`
}

type pubsubMessage struct {
	Data       string            `json:"data"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type publishResponse struct {
	MessageIDs []string `json:"messageIds"`
}

func PublishMessageToTopic(ctx context.Context, client *common.Client, projectID, topicID, data string, attributes map[string]string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/topics/%s:publish", pubsubBaseURL, projectID, topicID)
	req := publishRequest{
		Messages: []pubsubMessage{
			{Data: data, Attributes: attributes},
		},
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal publish body: %w", err)
	}
	resp, err := client.ExecRequest(ctx, "POST", url, strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	var result publishResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse publish response: %w", err)
	}
	if len(result.MessageIDs) == 0 {
		return "", fmt.Errorf("no message ID returned")
	}
	return result.MessageIDs[0], nil
}

// --- Pub/Sub Subscription ---

type pushConfig struct {
	PushEndpoint string `json:"pushEndpoint"`
}

type subscriptionRequest struct {
	Topic                    string      `json:"topic"`
	PushConfig               *pushConfig `json:"pushConfig,omitempty"`
	AckDeadlineSeconds       int         `json:"ackDeadlineSeconds"`
	MessageRetentionDuration string      `json:"messageRetentionDuration"`
	Filter                   string      `json:"filter,omitempty"`
}

func CreatePushSubscription(ctx context.Context, client *common.Client, projectID, subscriptionID, topicID, pushEndpoint string, filter ...string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s", pubsubBaseURL, projectID, subscriptionID)
	req := subscriptionRequest{
		Topic:                    fmt.Sprintf("projects/%s/topics/%s", projectID, topicID),
		PushConfig:               &pushConfig{PushEndpoint: pushEndpoint},
		AckDeadlineSeconds:       30,
		MessageRetentionDuration: "600s",
	}
	if len(filter) > 0 && filter[0] != "" {
		req.Filter = filter[0]
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal subscription body: %w", err)
	}
	_, err = client.ExecRequest(ctx, "PUT", url, strings.NewReader(string(raw)))
	if err != nil {
		if common.IsAlreadyExistsError(err) {
			return nil
		}
		return err
	}
	return nil
}

func CreatePullSubscription(ctx context.Context, client *common.Client, projectID, subscriptionID, topicID string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s", pubsubBaseURL, projectID, subscriptionID)
	req := subscriptionRequest{
		Topic:                    fmt.Sprintf("projects/%s/topics/%s", projectID, topicID),
		AckDeadlineSeconds:       30,
		MessageRetentionDuration: "604800s",
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal subscription body: %w", err)
	}
	_, err = client.ExecRequest(ctx, "PUT", url, strings.NewReader(string(raw)))
	if err != nil {
		if common.IsAlreadyExistsError(err) {
			return nil
		}
		return err
	}
	return nil
}

func UpdatePushEndpoint(ctx context.Context, client *common.Client, projectID, subscriptionID, pushEndpoint string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s:modifyPushConfig", pubsubBaseURL, projectID, subscriptionID)
	body := map[string]any{
		"pushConfig": map[string]string{
			"pushEndpoint": pushEndpoint,
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal modifyPushConfig body: %w", err)
	}
	_, err = client.ExecRequest(ctx, "POST", url, strings.NewReader(string(raw)))
	return err
}

func DeleteSubscription(ctx context.Context, client *common.Client, projectID, subscriptionID string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s", pubsubBaseURL, projectID, subscriptionID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

type listSubscriptionsResponse struct {
	Subscriptions []SubscriptionResource `json:"subscriptions"`
	NextPageToken string                 `json:"nextPageToken"`
}

type SubscriptionResource struct {
	Name  string `json:"name"`
	Topic string `json:"topic"`
}

func ListSubscriptions(ctx context.Context, client *common.Client, projectID string) ([]SubscriptionResource, error) {
	var all []SubscriptionResource
	pageToken := ""
	for {
		u := fmt.Sprintf("%s/projects/%s/subscriptions", pubsubBaseURL, projectID)
		if pageToken != "" {
			u += "?pageToken=" + pageToken
		}
		body, err := client.GetURL(ctx, u)
		if err != nil {
			return nil, err
		}
		var resp listSubscriptionsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse list subscriptions response: %w", err)
		}
		all = append(all, resp.Subscriptions...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return all, nil
}

func SubscriptionShortName(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

// --- Cloud Logging Sink ---

type sinkRequest struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
	Filter      string `json:"filter"`
}

type sinkResponse struct {
	Name           string `json:"name"`
	WriterIdentity string `json:"writerIdentity"`
}

func CreateSink(ctx context.Context, client *common.Client, projectID, sinkID, topicID, filter string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/sinks?uniqueWriterIdentity=true", loggingBaseURL, projectID)
	req := sinkRequest{
		Name:        sinkID,
		Destination: fmt.Sprintf("pubsub.googleapis.com/projects/%s/topics/%s", projectID, topicID),
		Filter:      filter,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal sink body: %w", err)
	}
	resp, err := client.ExecRequest(ctx, "POST", url, strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}

	var s sinkResponse
	if err := json.Unmarshal(resp, &s); err != nil {
		return "", fmt.Errorf("parse sink response: %w", err)
	}
	return s.WriterIdentity, nil
}

func GetSink(ctx context.Context, client *common.Client, projectID, sinkID string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/sinks/%s", loggingBaseURL, projectID, sinkID)
	resp, err := client.ExecRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	var s sinkResponse
	if err := json.Unmarshal(resp, &s); err != nil {
		return "", fmt.Errorf("parse sink response: %w", err)
	}
	return s.WriterIdentity, nil
}

func DeleteSink(ctx context.Context, client *common.Client, projectID, sinkID string) error {
	url := fmt.Sprintf("%s/projects/%s/sinks/%s", loggingBaseURL, projectID, sinkID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

// --- IAM ---

type iamPolicy struct {
	Bindings []iamBinding `json:"bindings"`
	Etag     string       `json:"etag,omitempty"`
}

type iamBinding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

type setIAMPolicyRequest struct {
	Policy iamPolicy `json:"policy"`
}

func EnsureTopicPublisher(ctx context.Context, client *common.Client, projectID, topicID, writerIdentity string) error {
	getURL := fmt.Sprintf("%s/projects/%s/topics/%s:getIamPolicy", pubsubBaseURL, projectID, topicID)
	resp, err := client.ExecRequest(ctx, "GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("get topic IAM policy: %w", err)
	}

	var policy iamPolicy
	if err := json.Unmarshal(resp, &policy); err != nil {
		return fmt.Errorf("parse IAM policy: %w", err)
	}

	const publisherRole = "roles/pubsub.publisher"
	for _, binding := range policy.Bindings {
		if binding.Role == publisherRole {
			for _, m := range binding.Members {
				if m == writerIdentity {
					return nil
				}
			}
		}
	}

	found := false
	for i, binding := range policy.Bindings {
		if binding.Role == publisherRole {
			policy.Bindings[i].Members = append(binding.Members, writerIdentity)
			found = true
			break
		}
	}
	if !found {
		policy.Bindings = append(policy.Bindings, iamBinding{
			Role:    publisherRole,
			Members: []string{writerIdentity},
		})
	}

	setURL := fmt.Sprintf("%s/projects/%s/topics/%s:setIamPolicy", pubsubBaseURL, projectID, topicID)
	raw, err := json.Marshal(setIAMPolicyRequest{Policy: policy})
	if err != nil {
		return fmt.Errorf("marshal IAM policy: %w", err)
	}
	_, err = client.ExecRequest(ctx, "POST", setURL, strings.NewReader(string(raw)))
	if err != nil {
		return fmt.Errorf("set topic IAM policy: %w", err)
	}
	return nil
}

// --- Service Usage (API enablement check) ---

func IsAPIEnabled(ctx context.Context, client *common.Client, projectID, service string) (bool, error) {
	url := fmt.Sprintf("https://serviceusage.googleapis.com/v1/projects/%s/services/%s", projectID, service)
	body, err := client.GetURL(ctx, url)
	if err != nil {
		return false, err
	}

	var resp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return false, fmt.Errorf("parse service state: %w", err)
	}
	return resp.State == "ENABLED", nil
}
