package github

import (
	"fmt"
	"strconv"
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

func (i *GitHubEventHandler) HandleWebhook(payload []byte) (integrations.StatefulResource, error) {
	//
	// TODO: I'm hardcoding event type here, but we also listen to different event types
	// and we should make sure we filter them out somehow.
	//
	event, err := github.ParseWebHook("workflow_run", payload)
	if err != nil {
		return nil, fmt.Errorf("error parsing webhook: %v", err)
	}

	switch e := event.(type) {
	case *github.WorkflowRunEvent:
		return &WorkflowRun{
			ID:         e.GetWorkflowRun().GetID(),
			Status:     e.GetWorkflowRun().GetStatus(),
			Conclusion: e.GetWorkflowRun().GetConclusion(),
			Repository: e.GetRepo().GetFullName(),
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

func parseWorkflowRunID(id string) (string, int64, error) {
	parts := strings.Split(id, ":")
	if len(parts) == 2 {
		runID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return "", -1, fmt.Errorf("invalid run ID: %s", parts[1])
		}

		return parts[0], runID, nil
	}

	return "", -1, fmt.Errorf("invalid workflow run ID format: %s (expected format: owner/repo:runID)", id)
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
