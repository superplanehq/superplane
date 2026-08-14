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
// `labels.nodes[0].name`. Null connections and null node lists decode as empty
// rather than failing: erroring here would fail the execution after the issue
// was already created on Linear, inviting duplicate-creating retries.
type LabelList []Label

func (l *LabelList) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*l = nil
		return nil
	}

	if trimmed[0] == '{' {
		connection := struct {
			Nodes []Label `json:"nodes"`
		}{}

		if err := json.Unmarshal(trimmed, &connection); err != nil {
			return err
		}

		*l = connection.Nodes
		return nil
	}

	plain := []Label{}
	if err := json.Unmarshal(trimmed, &plain); err != nil {
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

// Comment is a comment on an issue. user is the author and is null when Linear
// attributes the comment to a bot or integration rather than a person. editedAt
// stays null until the body is changed, so it doubles as an edited marker.
type Comment struct {
	ID        string        `json:"id"`
	Body      string        `json:"body"`
	URL       string        `json:"url"`
	CreatedAt string        `json:"createdAt,omitempty"`
	UpdatedAt string        `json:"updatedAt,omitempty"`
	EditedAt  string        `json:"editedAt,omitempty"`
	User      *User         `json:"user,omitempty"`
	Issue     *CommentIssue `json:"issue,omitempty"`
}

// CommentIssue is the issue a comment belongs to, trimmed to the fields worth
// surfacing alongside the comment.
type CommentIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	URL        string `json:"url"`
}

// Attachment is a link card on an issue. Linear deduplicates attachments by url
// within an issue, so creating one with an existing url updates it in place.
// iconUrl is deliberately absent: Linear accepts it on AttachmentCreateInput but
// does not expose it on the Attachment type, so it can never be read back.
type Attachment struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Subtitle  string        `json:"subtitle,omitempty"`
	URL       string        `json:"url"`
	CreatedAt string        `json:"createdAt,omitempty"`
	UpdatedAt string        `json:"updatedAt,omitempty"`
	Creator   *User         `json:"creator,omitempty"`
	Issue     *CommentIssue `json:"issue,omitempty"`
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

const issueQuery = `
query Issue($id: String!) {
  issue(id: $id) {` + issueFields + `
  }
}`

// GetIssue fetches a single issue. Linear accepts either the issue UUID or the
// human-readable identifier (e.g. ENG-142) as the id argument.
func (c *Client) GetIssue(id string) (*Issue, error) {
	response := struct {
		Issue *Issue `json:"issue"`
	}{}

	if err := c.execute(issueQuery, map[string]any{"id": id}, &response); err != nil {
		return nil, err
	}

	if response.Issue == nil {
		return nil, fmt.Errorf("issue %s not found", id)
	}

	return response.Issue, nil
}

const updateIssueMutation = `
mutation UpdateIssue($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
    issue {` + issueFields + `
    }
  }
}`

// UpdateIssue applies the given IssueUpdateInput fields to an issue. Like
// GetIssue, the id may be the issue UUID or its identifier (e.g. ENG-142).
func (c *Client) UpdateIssue(id string, input map[string]any) (*Issue, error) {
	response := struct {
		IssueUpdate struct {
			Success bool   `json:"success"`
			Issue   *Issue `json:"issue"`
		} `json:"issueUpdate"`
	}{}

	if err := c.execute(updateIssueMutation, map[string]any{"id": id, "input": input}, &response); err != nil {
		return nil, err
	}

	if !response.IssueUpdate.Success || response.IssueUpdate.Issue == nil {
		return nil, fmt.Errorf("linear reported the issue was not updated")
	}

	return response.IssueUpdate.Issue, nil
}

const commentFields = `
      id body url createdAt updatedAt editedAt
      user { id name displayName email }
      issue { id identifier title url }`

const createCommentMutation = `
mutation CommentCreate($input: CommentCreateInput!) {
  commentCreate(input: $input) {
    success
    comment {` + commentFields + `
    }
  }
}`

// CreateComment adds a comment to an issue. The input's issueId may be the issue
// UUID or its identifier (e.g. ENG-142).
func (c *Client) CreateComment(input map[string]any) (*Comment, error) {
	response := struct {
		CommentCreate struct {
			Success bool     `json:"success"`
			Comment *Comment `json:"comment"`
		} `json:"commentCreate"`
	}{}

	if err := c.execute(createCommentMutation, map[string]any{"input": input}, &response); err != nil {
		return nil, err
	}

	if !response.CommentCreate.Success || response.CommentCreate.Comment == nil {
		return nil, fmt.Errorf("linear reported the comment was not created")
	}

	return response.CommentCreate.Comment, nil
}

const updateCommentMutation = `
mutation CommentUpdate($id: String!, $input: CommentUpdateInput!) {
  commentUpdate(id: $id, input: $input) {
    success
    comment {` + commentFields + `
    }
  }
}`

// UpdateComment edits an existing comment. Unlike issues, comments are addressed
// only by their UUID - there is no human-readable identifier for one.
func (c *Client) UpdateComment(id string, input map[string]any) (*Comment, error) {
	response := struct {
		CommentUpdate struct {
			Success bool     `json:"success"`
			Comment *Comment `json:"comment"`
		} `json:"commentUpdate"`
	}{}

	if err := c.execute(updateCommentMutation, map[string]any{"id": id, "input": input}, &response); err != nil {
		return nil, err
	}

	if !response.CommentUpdate.Success || response.CommentUpdate.Comment == nil {
		return nil, fmt.Errorf("linear reported the comment was not updated")
	}

	return response.CommentUpdate.Comment, nil
}

const issueCommentsQuery = `
query IssueComments($id: String!) {
  issue(id: $id) {
    comments(first: 100) {
      nodes {
        id body
        user { id name displayName email }
      }
    }
  }
}`

// ListIssueComments returns the comments on a single issue, for the comment
// picker. Issues carry few enough comments that one page is plenty.
func (c *Client) ListIssueComments(issueID string) ([]Comment, error) {
	response := struct {
		Issue *struct {
			Comments struct {
				Nodes []Comment `json:"nodes"`
			} `json:"comments"`
		} `json:"issue"`
	}{}

	if err := c.execute(issueCommentsQuery, map[string]any{"id": issueID}, &response); err != nil {
		return nil, err
	}

	if response.Issue == nil {
		return nil, fmt.Errorf("issue %s not found", issueID)
	}

	return response.Issue.Comments.Nodes, nil
}

const attachmentFields = `
      id title subtitle url createdAt updatedAt
      creator { id name displayName email }
      issue { id identifier title url }`

const createAttachmentMutation = `
mutation AttachmentCreate($input: AttachmentCreateInput!) {
  attachmentCreate(input: $input) {
    success
    attachment {` + attachmentFields + `
    }
  }
}`

// CreateAttachment adds a link attachment to an issue. Linear deduplicates by
// url within an issue, so an existing url updates that attachment instead of
// creating a second one. The input's issueId may be the issue UUID or its
// identifier (e.g. ENG-142).
func (c *Client) CreateAttachment(input map[string]any) (*Attachment, error) {
	response := struct {
		AttachmentCreate struct {
			Success    bool        `json:"success"`
			Attachment *Attachment `json:"attachment"`
		} `json:"attachmentCreate"`
	}{}

	if err := c.execute(createAttachmentMutation, map[string]any{"input": input}, &response); err != nil {
		return nil, err
	}

	if !response.AttachmentCreate.Success || response.AttachmentCreate.Attachment == nil {
		return nil, fmt.Errorf("linear reported the attachment was not created")
	}

	return response.AttachmentCreate.Attachment, nil
}

const deleteAttachmentMutation = `
mutation AttachmentDelete($id: String!) {
  attachmentDelete(id: $id) {
    success
  }
}`

// DeleteAttachment removes an attachment by its ID. Linear has no delete-by-url,
// so the ID must come from createAttachment's output or an attachment event.
func (c *Client) DeleteAttachment(id string) error {
	response := struct {
		AttachmentDelete struct {
			Success bool `json:"success"`
		} `json:"attachmentDelete"`
	}{}

	if err := c.execute(deleteAttachmentMutation, map[string]any{"id": id}, &response); err != nil {
		return err
	}

	if !response.AttachmentDelete.Success {
		return fmt.Errorf("linear reported the attachment was not deleted")
	}

	return nil
}

const issueAttachmentsQuery = `
query IssueAttachments($id: String!) {
  issue(id: $id) {
    attachments(first: 100) {
      nodes { id title subtitle url }
    }
  }
}`

// ListIssueAttachments returns the attachments on a single issue, for the
// attachment picker. Issues carry few enough attachments that one page is plenty.
func (c *Client) ListIssueAttachments(issueID string) ([]Attachment, error) {
	response := struct {
		Issue *struct {
			Attachments struct {
				Nodes []Attachment `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}{}

	if err := c.execute(issueAttachmentsQuery, map[string]any{"id": issueID}, &response); err != nil {
		return nil, err
	}

	if response.Issue == nil {
		return nil, fmt.Errorf("issue %s not found", issueID)
	}

	return response.Issue.Attachments.Nodes, nil
}

// Reaction is an emoji reaction on an issue or a comment. Linear normalizes
// emoji aliases on write, so the stored emoji can differ from the one sent
// (thumbsup becomes +1).
type Reaction struct {
	ID        string   `json:"id"`
	Emoji     string   `json:"emoji"`
	CreatedAt string   `json:"createdAt,omitempty"`
	User      *User    `json:"user,omitempty"`
	Issue     *Issue   `json:"issue,omitempty"`
	Comment   *Comment `json:"comment,omitempty"`
}

const reactionFields = `
      id emoji createdAt
      user { id name displayName email }`

const createReactionMutation = `
mutation ReactionCreate($input: ReactionCreateInput!) {
  reactionCreate(input: $input) {
    success
    reaction {` + reactionFields + `
    }
  }
}`

// CreateReaction adds an emoji reaction to an issue or a comment. Re-sending the
// same emoji returns the existing reaction rather than failing, but sending an
// alias of an emoji already present is rejected - see isReactionConflict.
func (c *Client) CreateReaction(input map[string]any) (*Reaction, error) {
	response := struct {
		ReactionCreate struct {
			Success  bool      `json:"success"`
			Reaction *Reaction `json:"reaction"`
		} `json:"reactionCreate"`
	}{}

	if err := c.execute(createReactionMutation, map[string]any{"input": input}, &response); err != nil {
		return nil, err
	}

	if !response.ReactionCreate.Success || response.ReactionCreate.Reaction == nil {
		return nil, fmt.Errorf("linear reported the reaction was not created")
	}

	return response.ReactionCreate.Reaction, nil
}

const addIssueLabelMutation = `
mutation IssueAddLabel($id: String!, $labelId: String!) {
  issueAddLabel(id: $id, labelId: $labelId) {
    success
    issue {` + issueFields + `
    }
  }
}`

const removeIssueLabelMutation = `
mutation IssueRemoveLabel($id: String!, $labelId: String!) {
  issueRemoveLabel(id: $id, labelId: $labelId) {
    success
    issue {` + issueFields + `
    }
  }
}`

// RemoveIssueLabel removes a single label from an issue without touching its
// other labels, returning the updated issue. Like AddIssueLabel, the id may be
// the issue UUID or its identifier (e.g. ENG-142). Linear errors when the label
// is not on the issue - see isLabelNotOnIssue.
func (c *Client) RemoveIssueLabel(id, labelID string) (*Issue, error) {
	response := struct {
		IssueRemoveLabel struct {
			Success bool   `json:"success"`
			Issue   *Issue `json:"issue"`
		} `json:"issueRemoveLabel"`
	}{}

	if err := c.execute(removeIssueLabelMutation, map[string]any{"id": id, "labelId": labelID}, &response); err != nil {
		return nil, err
	}

	if !response.IssueRemoveLabel.Success || response.IssueRemoveLabel.Issue == nil {
		return nil, fmt.Errorf("linear reported the label was not removed")
	}

	return response.IssueRemoveLabel.Issue, nil
}

// AddIssueLabel adds a single label to an issue without touching its existing
// labels, returning the updated issue. Like UpdateIssue, the id may be the issue
// UUID or its identifier (e.g. ENG-142).
func (c *Client) AddIssueLabel(id, labelID string) (*Issue, error) {
	response := struct {
		IssueAddLabel struct {
			Success bool   `json:"success"`
			Issue   *Issue `json:"issue"`
		} `json:"issueAddLabel"`
	}{}

	if err := c.execute(addIssueLabelMutation, map[string]any{"id": id, "labelId": labelID}, &response); err != nil {
		return nil, err
	}

	if !response.IssueAddLabel.Success || response.IssueAddLabel.Issue == nil {
		return nil, fmt.Errorf("linear reported the label was not added")
	}

	return response.IssueAddLabel.Issue, nil
}

const createLabelMutation = `
mutation IssueLabelCreate($input: IssueLabelCreateInput!) {
  issueLabelCreate(input: $input) {
    success
    issueLabel { id name }
  }
}`

// CreateLabel creates a label and returns it. A non-empty teamID scopes the
// label to that team; an empty teamID creates a workspace-level label. Linear
// assigns a color when none is given.
func (c *Client) CreateLabel(name, teamID string) (*Label, error) {
	input := map[string]any{"name": name}
	if teamID != "" {
		input["teamId"] = teamID
	}

	response := struct {
		IssueLabelCreate struct {
			Success bool   `json:"success"`
			Label   *Label `json:"issueLabel"`
		} `json:"issueLabelCreate"`
	}{}

	if err := c.execute(createLabelMutation, map[string]any{"input": input}, &response); err != nil {
		return nil, err
	}

	if !response.IssueLabelCreate.Success || response.IssueLabelCreate.Label == nil {
		return nil, fmt.Errorf("linear reported the label was not created")
	}

	return response.IssueLabelCreate.Label, nil
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
