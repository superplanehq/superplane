package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	graphqlURL = "https://api.linear.app/graphql"
	maxPages   = 50 // safety cap for paginated queries
)

type Client struct {
	accessToken string
	http        core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	accessToken, err := findSecret(ctx, OAuthAccessToken)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	if accessToken == "" {
		return nil, fmt.Errorf("OAuth access token not found")
	}
	return &Client{
		accessToken: accessToken,
		http:        httpCtx,
	}, nil
}

// graphqlReq is the JSON body for a GraphQL request.
type graphqlReq struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphqlRes is the generic GraphQL response envelope.
type graphqlRes struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func (c *Client) execGraphQL(query string, variables map[string]any, result any) error {
	body := graphqlReq{Query: query, Variables: variables}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, graphqlURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("request got %d: %s", res.StatusCode, string(resBody))
	}

	var envelope graphqlRes
	if err := json.Unmarshal(resBody, &envelope); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if len(envelope.Errors) > 0 {
		return fmt.Errorf("graphql errors: %s", envelope.Errors[0].Message)
	}

	if result != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("decode data: %w", err)
		}
	}
	return nil
}

// Viewer is the authenticated user.
type Viewer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetViewer returns the current user (verifies credentials).
func (c *Client) GetViewer() (*Viewer, error) {
	const query = `query { viewer { id name email } }`
	var out struct {
		Viewer Viewer `json:"viewer"`
	}
	if err := c.execGraphQL(query, nil, &out); err != nil {
		return nil, err
	}
	return &out.Viewer, nil
}

// teamsResponse matches the GraphQL teams query.
type teamsResponse struct {
	Teams struct {
		Nodes    []Team   `json:"nodes"`
		PageInfo pageInfo `json:"pageInfo"`
	} `json:"teams"`
}

// ListTeams returns all teams the user can access.
func (c *Client) ListTeams() ([]Team, error) {
	const query = `query($after: String) { teams(first: 100, after: $after) { nodes { id name key } pageInfo { hasNextPage endCursor } } }`
	var all []Team
	var cursor *string
	for range maxPages {
		vars := map[string]any{"after": cursor}
		var out teamsResponse
		if err := c.execGraphQL(query, vars, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Teams.Nodes...)
		if !out.Teams.PageInfo.HasNextPage {
			break
		}
		cursor = &out.Teams.PageInfo.EndCursor
	}
	return all, nil
}

// FindTeam fetches all teams and returns the one matching the given ID.
func (c *Client) FindTeam(id string) (*Team, error) {
	teams, err := c.ListTeams()
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	for i := range teams {
		if teams[i].ID == id {
			return &teams[i], nil
		}
	}
	return nil, fmt.Errorf("team %s not found", id)
}

// organizationLabelsResponse for org-level labels.
type organizationLabelsResponse struct {
	Organization struct {
		Labels struct {
			Nodes    []Label  `json:"nodes"`
			PageInfo pageInfo `json:"pageInfo"`
		} `json:"labels"`
	} `json:"organization"`
}

// ListLabels returns all labels in the organization.
func (c *Client) ListLabels() ([]Label, error) {
	const query = `query($after: String) { organization { labels(first: 100, after: $after) { nodes { id name } pageInfo { hasNextPage endCursor } } } }`
	var all []Label
	var cursor *string
	for range maxPages {
		vars := map[string]any{"after": cursor}
		var out organizationLabelsResponse
		if err := c.execGraphQL(query, vars, &out); err != nil {
			return nil, err
		}
		all = append(all, out.Organization.Labels.Nodes...)
		if !out.Organization.Labels.PageInfo.HasNextPage {
			break
		}
		cursor = &out.Organization.Labels.PageInfo.EndCursor
	}
	return all, nil
}

// teamStatesResponse matches the team states query.
type teamStatesResponse struct {
	Team struct {
		States struct {
			Nodes []WorkflowState `json:"nodes"`
		} `json:"states"`
	} `json:"team"`
}

// ListWorkflowStates returns all workflow states for a team.
func (c *Client) ListWorkflowStates(teamID string) ([]WorkflowState, error) {
	const query = `query($id: String!) { team(id: $id) { states { nodes { id name type } } } }`
	var out teamStatesResponse
	if err := c.execGraphQL(query, map[string]any{"id": teamID}, &out); err != nil {
		return nil, err
	}
	return out.Team.States.Nodes, nil
}

// teamMembersResponse matches the team members query.
type teamMembersResponse struct {
	Team struct {
		Members struct {
			Nodes []Member `json:"nodes"`
		} `json:"members"`
	} `json:"team"`
}

// ListTeamMembers returns all human members of a team (excludes the app user itself).
func (c *Client) ListTeamMembers(teamID string) ([]Member, error) {
	const query = `query($id: String!) { team(id: $id) { members { nodes { id name displayName email active isMe } } } }`
	var out teamMembersResponse
	if err := c.execGraphQL(query, map[string]any{"id": teamID}, &out); err != nil {
		return nil, err
	}
	members := make([]Member, 0, len(out.Team.Members.Nodes))
	for _, m := range out.Team.Members.Nodes {
		if m.Active && !m.IsMe && m.Email != "" {
			members = append(members, m)
		}
	}
	return members, nil
}

// IssueCreateInput is the input for issueCreate mutation.
type IssueCreateInput struct {
	TeamID      string   `json:"teamId"`
	Title       string   `json:"title"`
	Description *string  `json:"description,omitempty"`
	AssigneeID  *string  `json:"assigneeId,omitempty"`
	LabelIDs    []string `json:"labelIds,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	StateID     *string  `json:"stateId,omitempty"`
}

// IssueCreateResponse is the issueCreate mutation result.
type IssueCreateResponse struct {
	IssueCreate struct {
		Success bool  `json:"success"`
		Issue   Issue `json:"issue"`
	} `json:"issueCreate"`
}

// IssueCreate creates a new issue.
func (c *Client) IssueCreate(input IssueCreateInput) (*Issue, error) {
	const query = `
mutation IssueCreate($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    success
    issue { id identifier title description priority url createdAt team { id } state { id } assignee { id } }
  }
}`
	var out IssueCreateResponse
	if err := c.execGraphQL(query, map[string]any{"input": input}, &out); err != nil {
		return nil, err
	}
	if !out.IssueCreate.Success {
		return nil, fmt.Errorf("issueCreate returned success: false")
	}
	return &out.IssueCreate.Issue, nil
}
