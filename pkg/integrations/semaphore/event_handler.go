package semaphore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations"
)

const (
	PipelineDoneEvent = "pipeline_done"

	PipelineStateDone    = "done"
	PipelineResultPassed = "passed"
	PipelineResultFailed = "failed"

	ResourceTypeTask         = "task"
	ResourceTypeProject      = "project"
	ResourceTypeWorkflow     = "workflow"
	ResourceTypeNotification = "notification"
	ResourceTypeSecret       = "secret"
	ResourceTypePipeline     = "pipeline"
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

func (i *SemaphoreEventHandler) Status(data []byte) (integrations.StatefulResource, error) {
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
