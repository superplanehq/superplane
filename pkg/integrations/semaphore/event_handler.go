package semaphore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/manifest"
)

const (
	PipelineDoneEvent = "pipeline_done"
)

type SemaphoreEventHandler struct{}

type Hook struct {
	Workflow HookWorkflow
	Pipeline HookPipeline
}

type HookWorkflow struct {
	ID string `json:"id"`
}

type HookPipeline struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Result string `json:"result"`
}

func (i *SemaphoreEventHandler) EventTypes() []string {
	return []string{PipelineDoneEvent}
}

func (i *SemaphoreEventHandler) Status(_ string, data []byte) (integrations.StatefulResource, error) {
	var hook Hook
	err := json.Unmarshal(data, &hook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling webhook data: %v", err)
	}

	return &Workflow{
		WfID: hook.Workflow.ID,
		Pipeline: &Pipeline{
			PipelineID: hook.Pipeline.ID,
			State:      hook.Pipeline.State,
			Result:     hook.Pipeline.Result,
		},
	}, nil
}

func (i *SemaphoreEventHandler) Handle(data []byte, header http.Header) (integrations.Event, error) {
	var hook Hook
	err := json.Unmarshal(data, &hook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling webhook data: %v", err)
	}

	signature := header.Get("X-Semaphore-Signature-256")
	if signature == "" {
		return nil, integrations.ErrInvalidSignature
	}

	return &SemaphoreEvent{
		PayloadSignature: strings.TrimPrefix(signature, "sha256="),
		Payload:          hook,
	}, nil
}

type SemaphoreEvent struct {
	PayloadSignature string `json:"signature"`
	Payload          Hook   `json:"payload"`
}

// Semaphore only has one type of event.
// Even though the event type is not in the event payload,
// this event is always sent when a pipeline is done,
// so we use the 'pipeline_done' type here.
func (e *SemaphoreEvent) Type() string {
	return PipelineDoneEvent
}

func (e *SemaphoreEvent) Signature() string {
	return e.PayloadSignature
}

func (i *SemaphoreEventHandler) Manifest() *manifest.TypeManifest {
	return &manifest.TypeManifest{
		Type:            "semaphore",
		DisplayName:     "Semaphore CI",
		Description:     "Receive events from Semaphore CI webhooks",
		Category:        "event_source",
		IntegrationType: "semaphore",
		Icon:            "semaphore",
		Fields: []manifest.FieldManifest{
			{
				Name:         "resource",
				DisplayName:  "Project",
				Type:         manifest.FieldTypeResource,
				Required:     true,
				Description:  "The Semaphore project to listen to",
				ResourceType: "project",
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
						Description: "The Semaphore event type",
						Options: []manifest.Option{
							{Value: PipelineDoneEvent, Label: "Pipeline Done"},
						},
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
										Description: "JSON path to the field",
										Placeholder: "$.pipeline.result",
									},
									{
										Name:        "value",
										DisplayName: "Value",
										Type:        manifest.FieldTypeString,
										Required:    true,
										Description: "Value to match",
										Placeholder: "passed",
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
										Placeholder: "X-Semaphore-Signature-256",
									},
									{
										Name:        "value",
										DisplayName: "Value",
										Type:        manifest.FieldTypeString,
										Required:    true,
										Description: "Value to match",
										Placeholder: "value",
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
