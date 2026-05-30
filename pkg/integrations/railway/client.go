package railway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultRailwayBaseURL = "https://backboard.railway.com/graphql/v2"

type Client struct {
	APIToken string
	BaseURL  string
	http     core.HTTPContext
}

type APIError struct {
	Errors []GraphQLError
}

func (e *APIError) Error() string {
	var msgs []string
	for _, err := range e.Errors {
		msgs = append(msgs, err.Message)
	}
	return fmt.Sprintf("GraphQL error: %s", strings.Join(msgs, "; "))
}

func NewClientWithAPIToken(http core.HTTPContext, apiToken string) *Client {
	return &Client{
		APIToken: strings.TrimSpace(apiToken),
		BaseURL:  defaultRailwayBaseURL,
		http:     http,
	}
}

func NewClientWithStorageContexts(http core.HTTPContext, properties core.IntegrationPropertyStorageReader, secrets core.IntegrationSecretStorageReader) (*Client, error) {
	apiToken, err := secrets.Get("apiToken")
	if err != nil {
		return nil, fmt.Errorf("apiToken not found in secrets: %w", err)
	}

	return NewClientWithAPIToken(http, apiToken), nil
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	if !ctx.LegacySetup() {
		return NewClientWithStorageContexts(http, ctx.Properties(), ctx.Secrets())
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("apiToken config not found: %w", err)
	}

	return NewClientWithAPIToken(http, string(apiToken)), nil
}

func (c *Client) execQuery(query string, variables map[string]any, responseData any) error {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	encodedBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL, bytes.NewReader(encodedBody))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIToken)

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("http request failed with status %d: %s", res.StatusCode, string(responseBody))
	}

	var gqlResponse GraphQLResponse
	if err := json.Unmarshal(responseBody, &gqlResponse); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL response: %w", err)
	}

	if len(gqlResponse.Errors) > 0 {
		return &APIError{Errors: gqlResponse.Errors}
	}

	if responseData != nil {
		if err := json.Unmarshal(gqlResponse.Data, responseData); err != nil {
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return nil
}

func (c *Client) Verify() error {
	query := `
		query {
			apiToken {
				workspaces {
					id
				}
			}
		}
	`
	var result struct {
		APIToken struct {
			Workspaces []struct {
				ID string `json:"id"`
			} `json:"workspaces"`
		} `json:"apiToken"`
	}

	return c.execQuery(query, nil, &result)
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	query := `
		query {
			apiToken {
				workspaces {
					id
					name
				}
			}
		}
	`
	var result struct {
		APIToken struct {
			Workspaces []Workspace `json:"workspaces"`
		} `json:"apiToken"`
	}

	if err := c.execQuery(query, nil, &result); err != nil {
		return nil, err
	}

	return result.APIToken.Workspaces, nil
}

func (c *Client) ListProjects(workspaceID string) ([]Project, error) {
	query := `
		query($workspaceId: String!) {
			projects(workspaceId: $workspaceId) {
				edges {
					node {
						id
						name
					}
				}
			}
		}
	`
	variables := map[string]any{
		"workspaceId": workspaceID,
	}

	var result struct {
		Projects struct {
			Edges []struct {
				Node Project `json:"node"`
			} `json:"edges"`
		} `json:"projects"`
	}

	if err := c.execQuery(query, variables, &result); err != nil {
		return nil, err
	}

	projects := make([]Project, 0, len(result.Projects.Edges))
	for _, edge := range result.Projects.Edges {
		projects = append(projects, edge.Node)
	}

	return projects, nil
}

func (c *Client) GetProjectDetails(projectID string) (*Project, error) {
	query := `
		query($id: String!) {
			project(id: $id) {
				id
				name
				workspaceId
				services {
					edges {
						node {
							id
							name
						}
					}
				}
				environments {
					edges {
						node {
							id
							name
						}
					}
				}
			}
		}
	`
	variables := map[string]any{
		"id": projectID,
	}

	var result struct {
		Project Project `json:"project"`
	}

	if err := c.execQuery(query, variables, &result); err != nil {
		return nil, err
	}

	return &result.Project, nil
}

func (c *Client) TriggerDeploy(environmentID, serviceID string) (string, error) {
	query := `
		mutation($environmentId: String!, $serviceId: String!) {
			serviceInstanceDeployV2(environmentId: $environmentId, serviceId: $serviceId)
		}
	`
	variables := map[string]any{
		"environmentId": environmentID,
		"serviceId":     serviceID,
	}

	var result struct {
		ServiceInstanceDeployV2 string `json:"serviceInstanceDeployV2"`
	}

	if err := c.execQuery(query, variables, &result); err != nil {
		return "", err
	}

	return result.ServiceInstanceDeployV2, nil
}

func (c *Client) GetDeployment(deploymentID string) (*Deployment, error) {
	query := `
		query($id: String!) {
			deployment(id: $id) {
				id
				status
				createdAt
				updatedAt
				statusUpdatedAt
				projectId
				serviceId
				environmentId
				snapshotId
				staticUrl
				url
				canRollback
				canRedeploy
				deploymentStopped
				meta
				diagnosis
				creator {
					id
					name
					email
					avatar
				}
			}
		}
	`
	variables := map[string]any{
		"id": deploymentID,
	}

	var result struct {
		Deployment Deployment `json:"deployment"`
	}

	if err := c.execQuery(query, variables, &result); err != nil {
		return nil, err
	}

	return &result.Deployment, nil
}

func (c *Client) RollbackDeployment(deploymentID string) error {
	query := `
		mutation($id: String!) {
			deploymentRollback(id: $id)
		}
	`
	variables := map[string]any{
		"id": deploymentID,
	}
	var result struct {
		DeploymentRollback bool `json:"deploymentRollback"`
	}

	if err := c.execQuery(query, variables, &result); err != nil {
		return err
	}

	if !result.DeploymentRollback {
		return fmt.Errorf("Railway did not accept rollback request")
	}

	return nil
}

func (c *Client) ListNotificationRules(workspaceID, projectID string) ([]NotificationRule, error) {
	query := `
		query($workspaceId: String!, $projectId: String!) {
			notificationRules(workspaceId: $workspaceId, projectId: $projectId) {
				id
				projectId
				eventTypes
				severities
				ephemeralEnvironments
				createdAt
				updatedAt
				channels {
					id
					config
					createdAt
					updatedAt
				}
			}
		}
	`
	variables := map[string]any{
		"workspaceId": workspaceID,
		"projectId":   projectID,
	}

	var result struct {
		NotificationRules []NotificationRule `json:"notificationRules"`
	}

	if err := c.execQuery(query, variables, &result); err != nil {
		return nil, err
	}

	return result.NotificationRules, nil
}

func (c *Client) CreateNotificationRule(workspaceID, projectID string, eventTypes []string, webhookURL string) (*NotificationRule, error) {
	query := `
		mutation($input: CreateNotificationRuleInput!) {
			notificationRuleCreate(input: $input) {
				id
				projectId
				eventTypes
				severities
				ephemeralEnvironments
				createdAt
				updatedAt
				channels {
					id
					config
					createdAt
					updatedAt
				}
			}
		}
	`
	variables := map[string]any{
		"input": map[string]any{
			"workspaceId": workspaceID,
			"projectId":   projectID,
			"eventTypes":  eventTypes,
			"channelConfigs": []map[string]any{
				{
					"url":  webhookURL,
					"type": "webhook",
				},
			},
		},
	}

	var result struct {
		NotificationRuleCreate NotificationRule `json:"notificationRuleCreate"`
	}

	if err := c.execQuery(query, variables, &result); err != nil {
		return nil, err
	}

	return &result.NotificationRuleCreate, nil
}

func (c *Client) DeleteNotificationRule(ruleID string) error {
	query := `
		mutation($id: String!) {
			notificationRuleDelete(id: $id)
		}
	`
	variables := map[string]any{
		"id": ruleID,
	}

	var result struct {
		NotificationRuleDelete bool `json:"notificationRuleDelete"`
	}

	return c.execQuery(query, variables, &result)
}
