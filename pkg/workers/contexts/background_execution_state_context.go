package contexts

import (
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// BackgroundExecutionStateContext keeps each mutation atomic without holding
// a connection while a component calls an external service.
type BackgroundExecutionStateContext struct {
	db          *gorm.DB
	request     *models.CanvasNodeRequest
	execution   *models.CanvasNodeExecution
	onNewEvents func([]models.CanvasEvent)
}

func NewBackgroundExecutionStateContext(db *gorm.DB, request *models.CanvasNodeRequest, execution *models.CanvasNodeExecution, onNewEvents func([]models.CanvasEvent)) *BackgroundExecutionStateContext {
	return &BackgroundExecutionStateContext{db: db, request: request, execution: execution, onNewEvents: onNewEvents}
}

func (s *BackgroundExecutionStateContext) IsFinished() bool {
	finished, err := s.execution.IsFinished(s.db)
	return err != nil || finished
}

func (s *BackgroundExecutionStateContext) IsCancelling() bool {
	execution, err := models.FindNodeExecutionInTransaction(s.db, s.execution.WorkflowID, s.execution.ID)
	return err != nil || execution.State == models.CanvasNodeExecutionStateCancelling
}

func (s *BackgroundExecutionStateContext) Emit(channel, payloadType string, payloads []any) error {
	return s.mutate(func(state *ExecutionStateContext) error { return state.Emit(channel, payloadType, payloads) })
}

func (s *BackgroundExecutionStateContext) EmitAndContinue(channel, payloadType string, payloads []any) error {
	return s.mutate(func(state *ExecutionStateContext) error { return state.EmitAndContinue(channel, payloadType, payloads) })
}

func (s *BackgroundExecutionStateContext) Pass() error {
	return s.mutate(func(state *ExecutionStateContext) error { return state.Pass() })
}

func (s *BackgroundExecutionStateContext) Fail(reason, message string) error {
	return s.mutate(func(state *ExecutionStateContext) error { return state.Fail(reason, message) })
}

func (s *BackgroundExecutionStateContext) Cancel() error {
	return s.mutate(func(state *ExecutionStateContext) error { return state.Cancel() })
}

func (s *BackgroundExecutionStateContext) SetKV(key, value string) error {
	return s.mutate(func(state *ExecutionStateContext) error { return state.SetKV(key, value) })
}

func (s *BackgroundExecutionStateContext) GetKV(key string) (string, error) {
	return NewExecutionStateContext(s.db, s.execution, nil).GetKV(key)
}

func (s *BackgroundExecutionStateContext) mutate(change func(*ExecutionStateContext) error) error {
	var events []models.CanvasEvent
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Cancellation locks the run before its executions and requests.
		run, err := models.LockCanvasRunInTransaction(tx, s.execution.RunID)
		if err != nil {
			return err
		}
		execution, err := models.LockNodeExecution(tx, s.execution.WorkflowID, s.execution.ID)
		if err != nil {
			return err
		}
		if err := lockBackgroundRequestLease(tx, s.request); err != nil {
			return err
		}
		if run.State == models.CanvasRunStateFinished || run.State == models.CanvasRunStateCancelling || run.CancelledAt != nil || execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
			return nil
		}
		node, err := models.FindUnscopedCanvasNode(tx, execution.WorkflowID, execution.NodeID)
		if err != nil {
			return err
		}
		if node.DeletedAt.Valid {
			return execution.RequestCancellation(tx, nil)
		}
		return change(NewExecutionStateContext(tx, execution, func(created []models.CanvasEvent) { events = append(events, created...) }))
	})
	if err == nil && len(events) > 0 && s.onNewEvents != nil {
		s.onNewEvents(events)
	}
	return err
}
