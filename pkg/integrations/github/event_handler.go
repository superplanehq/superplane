package github

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/manifest"
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

func (i *GitHubEventHandler) Manifest() *manifest.TypeManifest {
	eventTypes := github.MessageTypes()
	options := make([]manifest.Option, 0, len(eventTypes))
	for _, et := range eventTypes {
		options = append(options, manifest.Option{
			Value: et,
			Label: et,
		})
	}

	return &manifest.TypeManifest{
		Type:            "github",
		DisplayName:     "GitHub",
		Description:     "Receive events from GitHub webhooks",
		Category:        "event_source",
		IntegrationType: "github",
		Icon:            "github",
		Fields: []manifest.FieldManifest{
			{
				Name:         "resource",
				DisplayName:  "Repository",
				Type:         manifest.FieldTypeResource,
				Required:     true,
				Description:  "The GitHub repository to listen to",
				ResourceType: "repository",
			},
			{
				Name:        "eventTypes",
				DisplayName: "Event Type Filters",
				Type:        manifest.FieldTypeArray,
				ItemType:    manifest.FieldTypeObject,
				Required:    false,
				Description: "Filter which events should trigger executions",
				Fields: []manifest.FieldManifest{
					{
						Name:        "type",
						DisplayName: "Event Type",
						Type:        manifest.FieldTypeSelect,
						Required:    true,
						Description: "The GitHub event type",
						Options:     options,
					},
					{
						Name:        "filter_operator",
						DisplayName: "Filter Operator",
						Type:        manifest.FieldTypeSelect,
						Required:    false,
						Description: "How to combine multiple filters",
						Options: []manifest.Option{
							{Value: "and", Label: "AND"},
							{Value: "or", Label: "OR"},
						},
						Default: "and",
					},
					{
						Name:        "filters",
						DisplayName: "Filters",
						Type:        manifest.FieldTypeArray,
						ItemType:    manifest.FieldTypeObject,
						Required:    false,
						Description: "Conditions to match on event data",
						Fields: []manifest.FieldManifest{
							{
								Name:        "type",
								DisplayName: "Filter Type",
								Type:        manifest.FieldTypeSelect,
								Required:    true,
								Description: "What to filter on",
								Options: []manifest.Option{
									{Value: "data", Label: "Event Data"},
									{Value: "header", Label: "HTTP Header"},
								},
							},
							{
								Name:        "data",
								DisplayName: "Data Filter",
								Type:        manifest.FieldTypeObject,
								Required:    false,
								Description: "Filter on event payload data",
								DependsOn:   "type",
								Fields: []manifest.FieldManifest{
									{
										Name:        "path",
										DisplayName: "JSON Path",
										Type:        manifest.FieldTypeString,
										Required:    true,
										Description: "JSON path to the field (e.g., $.ref)",
										Placeholder: "$.ref",
									},
									{
										Name:        "value",
										DisplayName: "Value",
										Type:        manifest.FieldTypeString,
										Required:    true,
										Description: "Value to match",
										Placeholder: "refs/heads/main",
									},
								},
							},
							{
								Name:        "header",
								DisplayName: "Header Filter",
								Type:        manifest.FieldTypeObject,
								Required:    false,
								Description: "Filter on HTTP headers",
								DependsOn:   "type",
								Fields: []manifest.FieldManifest{
									{
										Name:        "name",
										DisplayName: "Header Name",
										Type:        manifest.FieldTypeString,
										Required:    true,
										Description: "HTTP header name",
										Placeholder: "X-GitHub-Event",
									},
									{
										Name:        "value",
										DisplayName: "Value",
										Type:        manifest.FieldTypeString,
										Required:    true,
										Description: "Value to match",
										Placeholder: "push",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
