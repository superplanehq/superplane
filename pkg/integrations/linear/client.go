package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const LinearGraphQLURL = "https://api.linear.app/graphql"

type Client struct {
	http        core.HTTPContext
	integration core.Integration
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func NewClient(http core.HTTPContext, integration core.Integration) *Client {
	return &Client{
		http:        http,
		integration: integration,
	}
}

func (c *Client) getAPIKey() (string, error) {
	var config Configuration
	err := mapstructure.Decode(c.integration.Configuration(), &config)
	if err != nil {
		return "", err
	}
	return config.APIKey, nil
}

func (c *Client) query(query string, variables map[string]interface{}, result interface{}) error {
	apiKey, err := c.getAPIKey()
	if err != nil {
		return err
	}

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", LinearGraphQLURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("linear API error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var graphQLResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphQLResp); err != nil {
		return err
	}

	if len(graphQLResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", graphQLResp.Errors[0].Message)
	}

	return json.Unmarshal(graphQLResp.Data, result)
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (c *Client) GetViewer() (*User, error) {
	query := `query { viewer { id name email } }`

	var result struct {
		Viewer User `json:"viewer"`
	}

	err := c.query(query, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result.Viewer, nil
}

func (c *Client) ListTeams() ([]Team, error) {
	query := `query { teams { nodes { id name key } } }`

	var result struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}

	err := c.query(query, nil, &result)
	if err != nil {
		return nil, err
	}

	return result.Teams.Nodes, nil
}

type IssueInput struct {
	TeamID      string   `json:"teamId"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	AssigneeID  string   `json:"assigneeId,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	StateID     string   `json:"stateId,omitempty"`
	LabelIDs    []string `json:"labelIds,omitempty"`
}

type Issue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

func (c *Client) CreateIssue(input IssueInput) (*Issue, error) {
	query := `
		mutation IssueCreate($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					url
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": input,
	}

	var result struct {
		IssueCreate struct {
			Success bool  `json:"success"`
			Issue   Issue `json:"issue"`
		} `json:"issueCreate"`
	}

	err := c.query(query, variables, &result)
	if err != nil {
		return nil, err
	}

	if !result.IssueCreate.Success {
		return nil, fmt.Errorf("failed to create issue")
	}

	return &result.IssueCreate.Issue, nil
}
