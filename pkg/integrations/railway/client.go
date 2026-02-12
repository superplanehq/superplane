package railway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const GraphQLEndpoint = "https://backboard.railway.com/graphql/v2"

type Client struct {
	apiToken string
	http     core.HTTPContext
}

type Project struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	CreatedAt    string        `json:"createdAt"`
	UpdatedAt    string        `json:"updatedAt"`
	Services     []Service     `json:"services"`
	Environments []Environment `json:"environments"`
}

type Service struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   map[string]any `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

type graphqlError struct {
	Message string `json:"message"`
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	token, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %v", err)
	}

	if len(token) == 0 {
		return nil, fmt.Errorf("API token is empty")
	}

	return &Client{
		apiToken: string(token),
		http:     httpCtx,
	}, nil
}

func (c *Client) graphql(query string, variables map[string]any) (map[string]any, error) {
	payload := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", GraphQLEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"Railway GraphQL API request failed with status %d: %s",
			resp.StatusCode,
			string(respBody),
		)
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("Railway GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

func (c *Client) ListProjects() ([]Project, error) {
	query := `
		query {
			projects {
				edges {
					node {
						id
						name
						description
						createdAt
						updatedAt
					}
				}
			}
		}
	`

	data, err := c.graphql(query, nil)
	if err != nil {
		return nil, err
	}

	projectsData, ok := data["projects"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response format for projects")
	}

	edges, ok := projectsData["edges"].([]any)
	if !ok {
		return []Project{}, nil
	}

	projects := make([]Project, 0, len(edges))
	for _, edge := range edges {
		edgeMap, ok := edge.(map[string]any)
		if !ok {
			continue
		}
		node, ok := edgeMap["node"].(map[string]any)
		if !ok {
			continue
		}

		projects = append(projects, Project{
			ID:          getString(node, "id"),
			Name:        getString(node, "name"),
			Description: getString(node, "description"),
			CreatedAt:   getString(node, "createdAt"),
			UpdatedAt:   getString(node, "updatedAt"),
		})
	}

	return projects, nil
}

func (c *Client) GetProject(projectID string) (*Project, error) {
	query := `
		query project($id: String!) {
			project(id: $id) {
				id
				name
				description
				services {
					edges {
						node {
							id
							name
							icon
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

	data, err := c.graphql(query, map[string]any{"id": projectID})
	if err != nil {
		return nil, err
	}

	projectData, ok := data["project"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("project not found")
	}

	project := &Project{
		ID:          getString(projectData, "id"),
		Name:        getString(projectData, "name"),
		Description: getString(projectData, "description"),
	}

	// Parse services
	if servicesData, ok := projectData["services"].(map[string]any); ok {
		if edges, ok := servicesData["edges"].([]any); ok {
			for _, edge := range edges {
				if edgeMap, ok := edge.(map[string]any); ok {
					if node, ok := edgeMap["node"].(map[string]any); ok {
						project.Services = append(project.Services, Service{
							ID:   getString(node, "id"),
							Name: getString(node, "name"),
							Icon: getString(node, "icon"),
						})
					}
				}
			}
		}
	}

	// Parse environments
	if envsData, ok := projectData["environments"].(map[string]any); ok {
		if edges, ok := envsData["edges"].([]any); ok {
			for _, edge := range edges {
				if edgeMap, ok := edge.(map[string]any); ok {
					if node, ok := edgeMap["node"].(map[string]any); ok {
						project.Environments = append(project.Environments, Environment{
							ID:   getString(node, "id"),
							Name: getString(node, "name"),
						})
					}
				}
			}
		}
	}

	return project, nil
}

// Deployment represents a Railway deployment with its current status
type Deployment struct {
	ID     string `json:"id"`
	Status string `json:"status"` // BUILDING, DEPLOYING, SUCCESS, FAILED, CRASHED, REMOVED, SLEEPING, SKIPPED, WAITING, QUEUED
	URL    string `json:"url"`
}

// DeploymentStatus constants
const (
	DeploymentStatusQueued    = "QUEUED"
	DeploymentStatusWaiting   = "WAITING"
	DeploymentStatusBuilding  = "BUILDING"
	DeploymentStatusDeploying = "DEPLOYING"
	DeploymentStatusSuccess   = "SUCCESS"
	DeploymentStatusFailed    = "FAILED"
	DeploymentStatusCrashed   = "CRASHED"
	DeploymentStatusRemoved   = "REMOVED"
	DeploymentStatusSleeping  = "SLEEPING"
	DeploymentStatusSkipped   = "SKIPPED"
)

// IsDeploymentFinalStatus returns true if the status is a final/terminal status
func IsDeploymentFinalStatus(status string) bool {
	switch status {
	case DeploymentStatusSuccess, DeploymentStatusFailed, DeploymentStatusCrashed,
		DeploymentStatusRemoved, DeploymentStatusSkipped:
		return true
	default:
		return false
	}
}

func (c *Client) TriggerDeploy(serviceID, environmentID string) (string, error) {
	// serviceInstanceDeployV2 returns a String! (deployment ID)
	mutation := `
		mutation serviceInstanceDeployV2($serviceId: String!, $environmentId: String!) {
			serviceInstanceDeployV2(serviceId: $serviceId, environmentId: $environmentId)
		}
	`

	variables := map[string]any{
		"serviceId":     serviceID,
		"environmentId": environmentID,
	}

	data, err := c.graphql(mutation, variables)
	if err != nil {
		return "", err
	}

	deploymentID, ok := data["serviceInstanceDeployV2"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format: deployment ID not found")
	}

	return deploymentID, nil
}

// GetDeployment retrieves the current status of a deployment
func (c *Client) GetDeployment(deploymentID string) (*Deployment, error) {
	query := `
		query deployment($id: String!) {
			deployment(id: $id) {
				id
				status
				url
			}
		}
	`

	data, err := c.graphql(query, map[string]any{"id": deploymentID})
	if err != nil {
		return nil, err
	}

	deploymentData, ok := data["deployment"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("deployment not found")
	}

	return &Deployment{
		ID:     getString(deploymentData, "id"),
		Status: getString(deploymentData, "status"),
		URL:    getString(deploymentData, "url"),
	}, nil
}

// Helper function to safely get string from map
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
