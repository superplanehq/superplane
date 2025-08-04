package github

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/integrations"
)

const (
	ResourceTypeRepository = "repository"
	ResourceTypeWorkflow   = "workflow"
	ResourceTypeWebHook    = "webhook"
)

type GitHubEventHandler struct{}

type GitHubEvent struct {
	EventType        string
	PayloadSignature string
}

func (e *GitHubEvent) Signature() string {
	return e.PayloadSignature
}

func (e *GitHubEvent) Type() string {
	return e.EventType
}

func (i *GitHubEventHandler) EventTypes() []string {
	return github.MessageTypes()
}

func (i *GitHubEventHandler) Handle(data []byte, header http.Header) (integrations.Event, error) {
	signature := header.Get("X-Hub-Signature-256")
	if signature == "" {
		return nil, integrations.ErrInvalidSignature
	}

	eventType := header.Get("X-GitHub-Event")
	if eventType == "" {
		return nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if !slices.Contains(github.MessageTypes(), eventType) {
		return nil, fmt.Errorf("unknown event type %s", eventType)
	}

	_, err := github.ParseWebHook(eventType, data)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s event: %v", eventType, err)
	}

	event := GitHubEvent{
		EventType:        eventType,
		PayloadSignature: strings.TrimPrefix(signature, "sha256="),
	}

	return &event, nil
}

func (i *GitHubEventHandler) Status(eventType string, eventPayload []byte) (integrations.StatefulResource, error) {
	if eventType != "workflow_run" {
		return nil, fmt.Errorf("unsupported event type %s", eventType)
	}

	event, err := github.ParseWebHook(eventType, eventPayload)
	if err != nil {
		return nil, fmt.Errorf("error parsing webhook: %v", err)
	}

	switch e := event.(type) {
	case *github.WorkflowRunEvent:
		return &WorkflowRun{
			ID:         e.GetWorkflowRun().GetID(),
			Status:     e.GetWorkflowRun().GetStatus(),
			Conclusion: e.GetWorkflowRun().GetConclusion(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported event type %T", event)
	}
}

func parseRepoName(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("invalid repository name format: %s", fullName)
}

type Webhook struct {
	ID          int64
	WebhookName string
	Repo        *Repository
}

func (h *Webhook) Id() string {
	return fmt.Sprintf("%d", h.ID)
}

func (h *Webhook) Name() string {
	return h.WebhookName
}

func (h *Webhook) Type() string {
	return ResourceTypeWebHook
}
