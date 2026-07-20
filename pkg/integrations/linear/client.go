package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// APIURL is Linear's single GraphQL endpoint. Linear has no REST API.
const APIURL = "https://api.linear.app/graphql"

// Client talks to Linear's GraphQL API using a personal API key. Linear expects
// the raw key in the Authorization header, without a "Bearer " prefix.
type Client struct {
	APIKey string
	http   core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, fmt.Errorf("error reading API key: %v", err)
	}

	if len(strings.TrimSpace(string(apiKey))) == 0 {
		return nil, fmt.Errorf("missing Linear API key")
	}

	if httpCtx == nil {
		return nil, fmt.Errorf("missing HTTP context")
	}

	return &Client{
		APIKey: strings.TrimSpace(string(apiKey)),
		http:   httpCtx,
	}, nil
}

// graphQLError is a single entry of the top-level `errors` array that Linear
// returns for failed operations. Linear answers with HTTP 200 even when the
// operation failed, so the errors array is the only reliable failure signal.
type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

// execute runs a GraphQL document and unmarshals the `data` object into out.
func (c *Client) execute(query string, variables map[string]any, out any) error {
	payload := map[string]any{"query": query}
	if len(variables) > 0 {
		payload["variables"] = variables
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, APIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.APIKey)

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	response := graphQLResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if len(response.Errors) > 0 {
		messages := make([]string, 0, len(response.Errors))
		for _, e := range response.Errors {
			messages = append(messages, e.Message)
		}
		return fmt.Errorf("linear API error: %s", strings.Join(messages, "; "))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(response.Data, out); err != nil {
		return fmt.Errorf("error parsing response data: %v", err)
	}

	return nil
}

type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
}

type Organization struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URLKey string `json:"urlKey"`
}

type Team struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type WorkflowState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LabelList flattens Linear's `labels { nodes { ... } }` connection into a plain
// array, so emitted payloads expose `labels[0].name` rather than
// `labels.nodes[0].name`.
type LabelList []Label

func (l *LabelList) UnmarshalJSON(data []byte) error {
	connection := struct {
		Nodes *[]Label `json:"nodes"`
	}{}

	if err := json.Unmarshal(data, &connection); err == nil && connection.Nodes != nil {
		*l = *connection.Nodes
		return nil
	}

	plain := []Label{}
	if err := json.Unmarshal(data, &plain); err != nil {
		return err
	}

	*l = plain
	return nil
}

type Issue struct {
	ID            string         `json:"id"`
	Identifier    string         `json:"identifier"`
	Number        int            `json:"number"`
	Title         string         `json:"title"`
	Description   string         `json:"description,omitempty"`
	URL           string         `json:"url"`
	Priority      int            `json:"priority"`
	PriorityLabel string         `json:"priorityLabel,omitempty"`
	BranchName    string         `json:"branchName,omitempty"`
	CreatedAt     string         `json:"createdAt,omitempty"`
	UpdatedAt     string         `json:"updatedAt,omitempty"`
	State         *WorkflowState `json:"state,omitempty"`
	Team          *Team          `json:"team,omitempty"`
	Assignee      *User          `json:"assignee,omitempty"`
	Creator       *User          `json:"creator,omitempty"`
	Project       *Project       `json:"project,omitempty"`
	Labels        LabelList      `json:"labels,omitempty"`
}

// Viewer identifies the account behind the API key and the workspace it belongs to.
type Viewer struct {
	User         *User        `json:"viewer"`
	Organization Organization `json:"organization"`
}

const viewerQuery = `
query Viewer {
  viewer { id name displayName email }
  organization { id name urlKey }
}`

func (c *Client) GetViewer() (*Viewer, error) {
	viewer := Viewer{}
	if err := c.execute(viewerQuery, nil, &viewer); err != nil {
		return nil, err
	}

	if viewer.User == nil {
		return nil, fmt.Errorf("no user returned for the API key")
	}

	return &viewer, nil
}

const teamsQuery = `
query Teams {
  teams(first: 250) { nodes { id key name } }
}`

func (c *Client) ListTeams() ([]Team, error) {
	response := struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}{}

	if err := c.execute(teamsQuery, nil, &response); err != nil {
		return nil, err
	}

	return response.Teams.Nodes, nil
}

const workflowStatesQuery = `
query WorkflowStates($teamId: ID!) {
  workflowStates(first: 250, filter: { team: { id: { eq: $teamId } } }) {
    nodes { id name type }
  }
}`

func (c *Client) ListWorkflowStates(teamID string) ([]WorkflowState, error) {
	response := struct {
		WorkflowStates struct {
			Nodes []WorkflowState `json:"nodes"`
		} `json:"workflowStates"`
	}{}

	if err := c.execute(workflowStatesQuery, map[string]any{"teamId": teamID}, &response); err != nil {
		return nil, err
	}

	return response.WorkflowStates.Nodes, nil
}

const teamMembersQuery = `
query TeamMembers($teamId: String!) {
  team(id: $teamId) {
    members(first: 250) { nodes { id name displayName email } }
  }
}`

func (c *Client) ListTeamMembers(teamID string) ([]User, error) {
	response := struct {
		Team *struct {
			Members struct {
				Nodes []User `json:"nodes"`
			} `json:"members"`
		} `json:"team"`
	}{}

	if err := c.execute(teamMembersQuery, map[string]any{"teamId": teamID}, &response); err != nil {
		return nil, err
	}

	if response.Team == nil {
		return nil, fmt.Errorf("team %s not found", teamID)
	}

	return response.Team.Members.Nodes, nil
}

// labelsQuery includes workspace-level labels alongside the team's own labels.
// Workspace labels have a null team, so filtering on team id alone hides them.
const labelsQuery = `
query Labels($teamId: ID!) {
  issueLabels(first: 250, filter: { or: [{ team: { id: { eq: $teamId } } }, { team: { null: true } }] }) {
    nodes { id name }
  }
}`

func (c *Client) ListLabels(teamID string) ([]Label, error) {
	response := struct {
		IssueLabels struct {
			Nodes []Label `json:"nodes"`
		} `json:"issueLabels"`
	}{}

	if err := c.execute(labelsQuery, map[string]any{"teamId": teamID}, &response); err != nil {
		return nil, err
	}

	return response.IssueLabels.Nodes, nil
}

const issueFields = `
      id identifier number title description url
      priority priorityLabel branchName createdAt updatedAt
      state { id name type }
      team { id key name }
      assignee { id name displayName email }
      creator { id name displayName email }
      project { id name }
      labels { nodes { id name } }`

const createIssueMutation = `
mutation CreateIssue($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    success
    issue {` + issueFields + `
    }
  }
}`

func (c *Client) CreateIssue(input map[string]any) (*Issue, error) {
	response := struct {
		IssueCreate struct {
			Success bool   `json:"success"`
			Issue   *Issue `json:"issue"`
		} `json:"issueCreate"`
	}{}

	if err := c.execute(createIssueMutation, map[string]any{"input": input}, &response); err != nil {
		return nil, err
	}

	if !response.IssueCreate.Success || response.IssueCreate.Issue == nil {
		return nil, fmt.Errorf("linear reported the issue was not created")
	}

	return response.IssueCreate.Issue, nil
}

type Webhook struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

const createWebhookMutation = `
mutation CreateWebhook($input: WebhookCreateInput!) {
  webhookCreate(input: $input) {
    success
    webhook { id url }
  }
}`

// CreateWebhook registers a webhook on Linear. Managing webhooks requires the
// API key owner to be a workspace admin.
func (c *Client) CreateWebhook(url, secret, label, teamID string, resourceTypes []string) (*Webhook, error) {
	input := map[string]any{
		"url":           url,
		"secret":        secret,
		"label":         label,
		"resourceTypes": resourceTypes,
		"enabled":       true,
	}

	if teamID != "" {
		input["teamId"] = teamID
	} else {
		input["allPublicTeams"] = true
	}

	response := struct {
		WebhookCreate struct {
			Success bool     `json:"success"`
			Webhook *Webhook `json:"webhook"`
		} `json:"webhookCreate"`
	}{}

	if err := c.execute(createWebhookMutation, map[string]any{"input": input}, &response); err != nil {
		return nil, err
	}

	if !response.WebhookCreate.Success || response.WebhookCreate.Webhook == nil {
		return nil, fmt.Errorf("linear reported the webhook was not created")
	}

	return response.WebhookCreate.Webhook, nil
}

const deleteWebhookMutation = `
mutation DeleteWebhook($id: String!) {
  webhookDelete(id: $id) { success }
}`

func (c *Client) DeleteWebhook(id string) error {
	response := struct {
		WebhookDelete struct {
			Success bool `json:"success"`
		} `json:"webhookDelete"`
	}{}

	if err := c.execute(deleteWebhookMutation, map[string]any{"id": id}, &response); err != nil {
		return err
	}

	if !response.WebhookDelete.Success {
		return fmt.Errorf("linear reported the webhook was not deleted")
	}

	return nil
}
