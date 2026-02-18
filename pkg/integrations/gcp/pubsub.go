package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

const pubSubBaseURL = "https://pubsub.googleapis.com/v1"

func CreateTopic(ctx context.Context, client *Client, projectID, topicID string) error {
	url := fmt.Sprintf("%s/projects/%s/topics/%s", pubSubBaseURL, projectID, topicID)
	_, err := client.ExecRequest(ctx, "PUT", url, nil)
	return err
}

func CreatePushSubscription(ctx context.Context, client *Client, projectID, topicID, subscriptionID, pushEndpoint string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s", pubSubBaseURL, projectID, subscriptionID)
	topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
	body := map[string]any{
		"topic": topicName,
		"pushConfig": map[string]string{
			"pushEndpoint": pushEndpoint,
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal subscription body: %w", err)
	}
	_, err = client.ExecRequest(ctx, "PUT", url, bytes.NewReader(raw))
	return err
}

func GrantTopicPublish(ctx context.Context, client *Client, projectID, topicID, member string) error {
	policy, err := getTopicIamPolicy(ctx, client, projectID, topicID)
	if err != nil {
		return err
	}
	role := "roles/pubsub.publisher"
	for i := range policy.Bindings {
		if policy.Bindings[i].Role == role {
			if !sliceContains(policy.Bindings[i].Members, member) {
				policy.Bindings[i].Members = append(policy.Bindings[i].Members, member)
			}
			return setTopicIamPolicy(ctx, client, projectID, topicID, policy)
		}
	}
	policy.Bindings = append(policy.Bindings, iamBinding{Role: role, Members: []string{member}})
	return setTopicIamPolicy(ctx, client, projectID, topicID, policy)
}

func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

type iamPolicy struct {
	Bindings []iamBinding `json:"bindings"`
	ETag     string       `json:"etag,omitempty"`
}

type iamBinding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

func getTopicIamPolicy(ctx context.Context, client *Client, projectID, topicID string) (*iamPolicy, error) {
	resource := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
	url := fmt.Sprintf("%s/%s:getIamPolicy", pubSubBaseURL, resource)
	body, err := client.ExecRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("get topic IAM policy: %w", err)
	}
	var policy iamPolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return nil, fmt.Errorf("parse IAM policy: %w", err)
	}
	return &policy, nil
}

func setTopicIamPolicy(ctx context.Context, client *Client, projectID, topicID string, policy *iamPolicy) error {
	url := fmt.Sprintf("%s/projects/%s/topics/%s:setIamPolicy", pubSubBaseURL, projectID, topicID)
	reqBody := map[string]any{"policy": policy}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal IAM policy: %w", err)
	}
	_, err = client.ExecRequest(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("set topic IAM policy: %w", err)
	}
	return nil
}

func DeleteSubscription(ctx context.Context, client *Client, projectID, subscriptionID string) error {
	url := fmt.Sprintf("%s/projects/%s/subscriptions/%s", pubSubBaseURL, projectID, subscriptionID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

func DeleteTopic(ctx context.Context, client *Client, projectID, topicID string) error {
	url := fmt.Sprintf("%s/projects/%s/topics/%s", pubSubBaseURL, projectID, topicID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}
