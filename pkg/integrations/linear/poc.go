package linear

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrInvalidWebhookPayload = errors.New("invalid linear webhook payload")
	ErrNotIssueCreatedEvent  = errors.New("not a linear issue.created event")
)

type webhookPayload struct {
	Action string      `json:"action"`
	Type   string      `json:"type"`
	Data   webhookData `json:"data"`
}

type webhookData struct {
	ID         string         `json:"id"`
	Identifier string         `json:"identifier"`
	Title      string         `json:"title"`
	URL        string         `json:"url"`
	Team       webhookTeam    `json:"team"`
	Labels     []webhookLabel `json:"labels"`
}

type webhookTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type webhookLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IssueCreatedEvent struct {
	IssueID     string   `json:"issueId"`
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	TeamID      string   `json:"teamId"`
	TeamName    string   `json:"teamName"`
	IssueURL    string   `json:"issueUrl"`
	IssueLabels []string `json:"issueLabels"`
}

func ParseIssueCreatedWebhook(body []byte) (IssueCreatedEvent, error) {
	payload := webhookPayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return IssueCreatedEvent{}, fmt.Errorf("%w: %v", ErrInvalidWebhookPayload, err)
	}

	if payload.Action != "create" || payload.Type != "Issue" {
		return IssueCreatedEvent{}, ErrNotIssueCreatedEvent
	}

	if payload.Data.ID == "" || payload.Data.Team.ID == "" {
		return IssueCreatedEvent{}, ErrInvalidWebhookPayload
	}

	labels := make([]string, 0, len(payload.Data.Labels))
	for _, label := range payload.Data.Labels {
		if label.Name == "" {
			continue
		}
		labels = append(labels, label.Name)
	}

	return IssueCreatedEvent{
		IssueID:     payload.Data.ID,
		Identifier:  payload.Data.Identifier,
		Title:       payload.Data.Title,
		TeamID:      payload.Data.Team.ID,
		TeamName:    payload.Data.Team.Name,
		IssueURL:    payload.Data.URL,
		IssueLabels: labels,
	}, nil
}

type CreateIssueInput struct {
	TeamID      string
	Title       string
	Description string
	AssigneeID  string
	LabelIDs    []string
	Priority    int
	StateID     string
}

func BuildIssueCreateVariables(input CreateIssueInput) (map[string]any, error) {
	if input.TeamID == "" {
		return nil, errors.New("teamId is required")
	}
	if input.Title == "" {
		return nil, errors.New("title is required")
	}
	if input.Priority < 0 || input.Priority > 4 {
		return nil, errors.New("priority must be in range 0..4")
	}

	mutationInput := map[string]any{
		"teamId":   input.TeamID,
		"title":    input.Title,
		"priority": input.Priority,
	}

	if input.Description != "" {
		mutationInput["description"] = input.Description
	}
	if input.AssigneeID != "" {
		mutationInput["assigneeId"] = input.AssigneeID
	}
	if len(input.LabelIDs) > 0 {
		mutationInput["labelIds"] = input.LabelIDs
	}
	if input.StateID != "" {
		mutationInput["stateId"] = input.StateID
	}

	return map[string]any{
		"input": mutationInput,
	}, nil
}

func IssueCreateMutation() string {
	return `mutation IssueCreate($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    success
    issue {
      id
      identifier
      title
      url
    }
  }
}`
}
