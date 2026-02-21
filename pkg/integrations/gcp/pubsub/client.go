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

// --- Pub/Sub Push Subscription ---

type pushConfig struct {
	PushEndpoint string `json:"pushEndpoint"`
}

type subscriptionRequest struct {
	Topic                    string      `json:"topic"`
	PushConfig               *pushConfig `json:"pushConfig"`
	AckDeadlineSeconds       int         `json:"ackDeadlineSeconds"`
	MessageRetentionDuration string      `json:"messageRetentionDuration"`
}

func CreatePushSubscription(ctx context.Context, client *common.Client, projectID, subscriptionID, topicID, pushEndpoint string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s", pubsubBaseURL, projectID, subscriptionID)
	req := subscriptionRequest{
		Topic:                    fmt.Sprintf("projects/%s/topics/%s", projectID, topicID),
		PushConfig:               &pushConfig{PushEndpoint: pushEndpoint},
		AckDeadlineSeconds:       30,
		MessageRetentionDuration: "600s",
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

func DeleteSubscription(ctx context.Context, client *common.Client, projectID, subscriptionID string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s", pubsubBaseURL, projectID, subscriptionID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
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
