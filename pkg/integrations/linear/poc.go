package linear

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	seenLabels := make(map[string]struct{}, len(payload.Data.Labels))
	for _, label := range payload.Data.Labels {
		name := strings.TrimSpace(label.Name)
		if name == "" {
			continue
		}
		if _, exists := seenLabels[name]; exists {
			continue
		}
		seenLabels[name] = struct{}{}
		labels = append(labels, name)
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
	teamID := strings.TrimSpace(input.TeamID)
	title := strings.TrimSpace(input.Title)
	if teamID == "" {
		return nil, errors.New("teamId is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if input.Priority < 0 || input.Priority > 4 {
		return nil, errors.New("priority must be in range 0..4")
	}

	mutationInput := map[string]any{
		"teamId":   teamID,
		"title":    title,
		"priority": input.Priority,
	}

	if description := strings.TrimSpace(input.Description); description != "" {
		mutationInput["description"] = description
	}
	if assigneeID := strings.TrimSpace(input.AssigneeID); assigneeID != "" {
		mutationInput["assigneeId"] = assigneeID
	}
	if len(input.LabelIDs) > 0 {
		labelIDs := make([]string, 0, len(input.LabelIDs))
		seenLabelIDs := make(map[string]struct{}, len(input.LabelIDs))
		for _, raw := range input.LabelIDs {
			labelID := strings.TrimSpace(raw)
			if labelID == "" {
				continue
			}
			if _, exists := seenLabelIDs[labelID]; exists {
				continue
			}
			seenLabelIDs[labelID] = struct{}{}
			labelIDs = append(labelIDs, labelID)
		}
		if len(labelIDs) > 0 {
			mutationInput["labelIds"] = labelIDs
		}
	}
	if stateID := strings.TrimSpace(input.StateID); stateID != "" {
		mutationInput["stateId"] = stateID
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
