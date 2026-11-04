package runner

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var extraRunnerComponents sync.Map

const (
	LiveLogErrorCodeHeader          = "X-Superplane-Error-Code"
	LiveLogSessionNotReadyErrorCode = "live_log_session_not_ready"
)

var (
	ErrLiveLogCanvasNotFound    = errors.New("canvas not found")
	ErrLiveLogExecutionNotFound = errors.New("execution not found")
	ErrLiveLogNodeNotFound      = errors.New("node not found")
	ErrLiveLogNotRunner         = errors.New("not a runner component")
	ErrLiveLogTaskMissing       = errors.New("runner task missing")
)

// LiveLogSession describes the organization-authorized SuperPlane log endpoint.
type LiveLogSession struct {
	Backend       string              `json:"backend"`
	StreamURL     string              `json:"stream_url"`
	HostedPricing *HostedUsagePricing `json:"hosted_pricing,omitempty"`
}

// LiveLogAccessContext is the resolved runner execution context for live log access.
type LiveLogAccessContext struct {
	Task          *models.RunnerTask
	ComponentName string
	Configuration map[string]any
}

func IsRunnerComponent(name string) bool {
	name = strings.TrimSpace(name)
	switch name {
	case ComponentName, RunJSComponentName, RunPythonComponentName, RunBashComponentName:
		return true
	default:
		_, ok := extraRunnerComponents.Load(name)
		return ok
	}
}

// RegisterRunnerComponent marks a runner subpackage component as eligible for live logs.
func RegisterRunnerComponent(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	extraRunnerComponents.Store(name, struct{}{})
}

func BrokerTaskIDFromExecutionMetadata(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	v, ok := meta[ExecutionMetadataBrokerTaskID]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func ResolveLiveLogAccess(tx *gorm.DB, orgID uuid.UUID, canvasID uuid.UUID, executionID uuid.UUID) (*LiveLogAccessContext, error) {
	if _, err := models.FindCanvasInTransaction(tx, orgID, canvasID); err != nil {
		return nil, liveLogLookupError(ErrLiveLogCanvasNotFound, err)
	}

	execution, err := models.FindNodeExecutionInTransaction(tx, canvasID, executionID)
	if err != nil {
		return nil, liveLogLookupError(ErrLiveLogExecutionNotFound, err)
	}

	node, err := models.FindCanvasNode(tx, canvasID, execution.NodeID)
	if err != nil {
		return nil, liveLogLookupError(ErrLiveLogNodeNotFound, err)
	}

	ref := node.Ref.Data()
	if ref.Component == nil || !IsRunnerComponent(ref.Component.Name) {
		return nil, ErrLiveLogNotRunner
	}

	brokerTaskID := BrokerTaskIDFromExecutionMetadata(execution.Metadata.Data())
	if brokerTaskID == "" {
		brokerTaskID, err = models.FindLatestNodeExecutionKVValueInTransaction(tx, execution.ID, "task_id")
		if err != nil {
			return nil, liveLogLookupError(ErrLiveLogTaskMissing, err)
		}
	}
	taskID, err := uuid.Parse(brokerTaskID)
	if err != nil {
		return nil, ErrLiveLogTaskMissing
	}
	task, err := models.FindRunnerTask(tx, taskID)
	if errors.Is(err, models.ErrRunnerTaskNotFound) {
		return nil, ErrLiveLogTaskMissing
	}
	if err != nil {
		return nil, err
	}
	if task.OrganizationID != orgID {
		return nil, ErrLiveLogTaskMissing
	}

	return &LiveLogAccessContext{
		Task:          task,
		ComponentName: ref.Component.Name,
		Configuration: execution.Configuration.Data(),
	}, nil
}

func liveLogLookupError(notFound error, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound
	}
	return err
}
