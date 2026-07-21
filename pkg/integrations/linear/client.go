package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// APIURL is Linear's single GraphQL endpoint. Linear has no REST API.
const APIURL = "https://api.linear.app/graphql"

// Client talks to Linear's GraphQL API using an OAuth access token, which
// Linear expects with a "Bearer " prefix — unlike personal API keys.
type Client struct {
	AccessToken string
	http        core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	accessToken, err := findSecret(ctx, OAuthAccessToken)
	if err != nil {
		return nil, fmt.Errorf("error reading access token: %v", err)
	}

	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("missing Linear access token - authorize the integration first")
	}

	if httpCtx == nil {
		return nil, fmt.Errorf("missing HTTP context")
	}

	return &Client{
		AccessToken: strings.TrimSpace(accessToken),
		http:        httpCtx,
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
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

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

// Viewer identifies the account behind the access token and the workspace it belongs to.
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
		return nil, fmt.Errorf("no user returned for the access token")
	}

	return &viewer, nil
}

const (
	// pageSize is how many nodes each page requests. Linear rejects a single
	// query above 10,000 complexity points and a connection multiplies its
	// children by this argument, so this stays well clear of that ceiling.
	pageSize = 100

	// maxPages bounds a paginated fetch so a misbehaving cursor cannot loop
	// forever. At pageSize this covers 25,000 records.
	maxPages = 250
)

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type connection[T any] struct {
	Nodes    []T      `json:"nodes"`
	PageInfo pageInfo `json:"pageInfo"`
}

// collectPages walks a Linear connection to completion, following the cursor
// until the API reports no further pages. decode selects the connection from
// each response, since every query nests it under a different field.
func collectPages[T any](c *Client, query string, variables map[string]any, decode func(json.RawMessage) (*connection[T], error)) ([]T, error) {
	pageVariables := map[string]any{}
	maps.Copy(pageVariables, variables)

	all := []T{}
	for range maxPages {
		data := json.RawMessage{}
		if err := c.execute(query, pageVariables, &data); err != nil {
			return nil, err
		}

		page, err := decode(data)
		if err != nil {
			return nil, err
		}

		all = append(all, page.Nodes...)

		if !page.PageInfo.HasNextPage || page.PageInfo.EndCursor == "" {
			return all, nil
		}

		pageVariables["after"] = page.PageInfo.EndCursor
	}

	return nil, fmt.Errorf("gave up paginating after %d pages", maxPages)
}

const teamsQuery = `
query Teams($first: Int!, $after: String) {
  teams(first: $first, after: $after) {
    nodes { id key name }
    pageInfo { hasNextPage endCursor }
  }
}`

func (c *Client) ListTeams() ([]Team, error) {
	return collectPages(c, teamsQuery, map[string]any{"first": pageSize}, func(data json.RawMessage) (*connection[Team], error) {
		response := struct {
			Teams connection[Team] `json:"teams"`
		}{}

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("error parsing teams: %v", err)
		}

		return &response.Teams, nil
	})
}

// workflowStatesQuery excludes "duplicate"-type states: an issue can only enter
// one by being marked as a duplicate of another issue, and issueCreate rejects
// them with "invalid state", so they must not appear in the status picker.
const workflowStatesQuery = `
query WorkflowStates($teamId: ID!, $first: Int!, $after: String) {
  workflowStates(first: $first, after: $after, filter: { team: { id: { eq: $teamId } }, type: { neq: "duplicate" } }) {
    nodes { id name type }
    pageInfo { hasNextPage endCursor }
  }
}`

func (c *Client) ListWorkflowStates(teamID string) ([]WorkflowState, error) {
	variables := map[string]any{"teamId": teamID, "first": pageSize}

	return collectPages(c, workflowStatesQuery, variables, func(data json.RawMessage) (*connection[WorkflowState], error) {
		response := struct {
			WorkflowStates connection[WorkflowState] `json:"workflowStates"`
		}{}

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("error parsing workflow states: %v", err)
		}

		return &response.WorkflowStates, nil
	})
}

const teamMembersQuery = `
query TeamMembers($teamId: String!, $first: Int!, $after: String) {
  team(id: $teamId) {
    members(first: $first, after: $after) {
      nodes { id name displayName email }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

func (c *Client) ListTeamMembers(teamID string) ([]User, error) {
	variables := map[string]any{"teamId": teamID, "first": pageSize}

	return collectPages(c, teamMembersQuery, variables, func(data json.RawMessage) (*connection[User], error) {
		response := struct {
			Team *struct {
				Members connection[User] `json:"members"`
			} `json:"team"`
		}{}

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("error parsing team members: %v", err)
		}

		if response.Team == nil {
			return nil, fmt.Errorf("team %s not found", teamID)
		}

		return &response.Team.Members, nil
	})
}

const teamProjectsQuery = `
query TeamProjects($teamId: String!, $first: Int!, $after: String) {
  team(id: $teamId) {
    projects(first: $first, after: $after) {
      nodes { id name }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

func (c *Client) ListTeamProjects(teamID string) ([]Project, error) {
	variables := map[string]any{"teamId": teamID, "first": pageSize}

	return collectPages(c, teamProjectsQuery, variables, func(data json.RawMessage) (*connection[Project], error) {
		response := struct {
			Team *struct {
				Projects connection[Project] `json:"projects"`
			} `json:"team"`
		}{}

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("error parsing team projects: %v", err)
		}

		if response.Team == nil {
			return nil, fmt.Errorf("team %s not found", teamID)
		}

		return &response.Team.Projects, nil
	})
}

// labelsQuery includes workspace-level labels alongside the team's own labels.
// Workspace labels have a null team, so filtering on team id alone hides them.
const labelsQuery = `
query Labels($teamId: ID!, $first: Int!, $after: String) {
  issueLabels(first: $first, after: $after, filter: { or: [{ team: { id: { eq: $teamId } } }, { team: { null: true } }] }) {
    nodes { id name }
    pageInfo { hasNextPage endCursor }
  }
}`

func (c *Client) ListLabels(teamID string) ([]Label, error) {
	variables := map[string]any{"teamId": teamID, "first": pageSize}

	return collectPages(c, labelsQuery, variables, func(data json.RawMessage) (*connection[Label], error) {
		response := struct {
			IssueLabels connection[Label] `json:"issueLabels"`
		}{}

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("error parsing labels: %v", err)
		}

		return &response.IssueLabels, nil
	})
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

// CreateWebhook registers a webhook on Linear. Managing webhooks requires a
// workspace admin or an OAuth token carrying the admin scope.
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
