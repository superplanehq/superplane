package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const bodySectionRequestLease = 5 * time.Minute

type bodySectionPublicationState struct {
	SourceRunID string     `json:"sourceRunId"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	Reason      string     `json:"reason,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
	NextRetryAt *time.Time `json:"nextRetryAt,omitempty"`
	LeaseUntil  *time.Time `json:"leaseUntil,omitempty"`
}

type bodySectionActionAttempt struct {
	request                  *models.CanvasNodeRequest
	execution                *models.CanvasNodeExecution
	source                   *models.CanvasRun
	configuration            factorycomponent.UpdatePullRequestBodySectionConfiguration
	integrationConfiguration prbody.Configuration
	component                string
	node                     *models.CanvasNode
	state                    bodySectionPublicationState
	events                   []models.CanvasEvent
}

// ProcessBodySectionPublicationRequest releases the request transaction
// before provider I/O. A lease recovers interrupted attempts through the same
// scheduled action queue used by other components.
func ProcessBodySectionPublicationRequest(ctx context.Context, deps IntakeDependencies, requestID uuid.UUID,
	lockRequest func(*gorm.DB, uuid.UUID) (*models.CanvasNodeRequest, error),
) ([]models.CanvasEvent, error) {
	db := database.DB(ctx)
	attempt := &bodySectionActionAttempt{}
	err := db.Transaction(func(tx *gorm.DB) error {
		request, err := lockRequest(tx, requestID)
		if err != nil {
			return err
		}
		attempt.request = request
		return attempt.claim(tx)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if attempt.state.Status != "publishing" {
		return attempt.events, nil
	}

	outcome, publishErr := attempt.publish(ctx, db, deps)
	err = db.Transaction(func(tx *gorm.DB) error { return attempt.finish(tx, outcome, publishErr) })
	return attempt.events, err
}

// stillActive rereads the run and the execution. A cancellation that arrives
// while the provider reads the description must stop the write.
func (a *bodySectionActionAttempt) stillActive(db *gorm.DB) (bool, error) {
	execution, err := models.FindNodeExecutionInTransaction(db, a.execution.WorkflowID, a.execution.ID)
	if err != nil {
		return false, err
	}
	if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		return false, nil
	}
	run, err := models.FindCanvasRunInTransaction(db, a.source.WorkflowID, a.source.ID)
	if err != nil {
		return false, err
	}
	return run.State == models.CanvasRunStateStarted, nil
}

func (a *bodySectionActionAttempt) collectEvents(events []models.CanvasEvent) {
	a.events = append(a.events, events...)
}

func (a *bodySectionActionAttempt) claim(tx *gorm.DB) error {
	request := a.request
	hook := request.Spec.Data().InvokeAction
	if request.Type != models.NodeRequestTypeInvokeAction || hook == nil || hook.ActionName != factorycomponent.UpdatePullRequestBodySectionHook || request.ExecutionID == nil {
		return fmt.Errorf("invalid publication action request")
	}
	execution, err := models.LockNodeExecution(tx, request.WorkflowID, *request.ExecutionID)
	if err != nil {
		return err
	}
	a.execution = execution
	if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		return request.Complete(tx)
	}
	node, err := models.FindUnscopedCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return err
	}
	if node.DeletedAt.Valid {
		if err := contexts.NewExecutionStateContext(tx, execution, a.collectEvents).Cancel(); err != nil {
			return err
		}
		return request.Complete(tx)
	}
	component := execution.FrozenComponentName()
	if component == "" && node.Ref.Data().Component != nil {
		component = node.Ref.Data().Component.Name
	}
	if !prbody.IsComponent(component) {
		return a.reject(tx, "Publication request has an invalid component.")
	}
	a.component, a.node = component, node
	encoded, err := execution.Metadata.MarshalJSON()
	if err != nil {
		return err
	}
	if err := models.UnmarshalJSONValue(encoded, &a.state); err != nil {
		return err
	}
	if a.state.LeaseUntil != nil && a.state.LeaseUntil.After(time.Now()) {
		return gorm.ErrRecordNotFound
	}
	a.state.SourceRunID = execution.RunID.String()
	a.state.Attempts++
	if err := a.resolveSource(tx); err != nil {
		return err
	}
	if a.state.Status == "rejected" {
		return nil
	}
	// PostgreSQL timestamps have microsecond precision. Persist and compare the
	// same value when fencing a completed attempt against a newer lease.
	until := time.Now().Add(bodySectionRequestLease).Truncate(time.Microsecond)
	a.state.Status, a.state.LeaseUntil, a.state.NextRetryAt = "publishing", &until, nil
	if err := setBodySectionPublicationState(tx, execution, a.state); err != nil {
		return err
	}
	return request.Reschedule(tx, until)
}

func (a *bodySectionActionAttempt) resolveSource(tx *gorm.DB) error {
	var err error
	a.source, err = models.FindCanvasRunInTransaction(tx, a.execution.WorkflowID, a.execution.RunID)
	if err != nil {
		return err
	}
	if a.source.State != models.CanvasRunStateStarted {
		return a.reject(tx, "The source run is no longer active.")
	}
	if a.component != factorycomponent.UpdatePullRequestBodySectionComponentName {
		a.integrationConfiguration, err = prbody.Decode(a.execution.Configuration.Data())
		if err == nil {
			err = a.integrationConfiguration.Validate()
		}
		if err != nil {
			return a.reject(tx, "Publication configuration is invalid: "+err.Error())
		}
		return nil
	}
	a.configuration, err = factorycomponent.DecodeBodySectionConfiguration(a.execution.Configuration.Data())
	if err != nil {
		return a.reject(tx, "Publication configuration is invalid: "+err.Error())
	}

	return nil
}

func (a *bodySectionActionAttempt) reject(tx *gorm.DB, message string) error {
	a.state.Status, a.state.Reason = "rejected", message
	if err := setBodySectionPublicationState(tx, a.execution, a.state); err != nil {
		return err
	}
	if err := contexts.NewExecutionStateContext(tx, a.execution, a.collectEvents).Fail(models.CanvasNodeExecutionResultReasonError, message); err != nil {
		return err
	}
	return a.request.Complete(tx)
}

func (a *bodySectionActionAttempt) finish(tx *gorm.DB, outcome bodySectionPublicationOutcome, publishErr error) error {
	current, err := models.LockNodeRequestNow(tx, a.request.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// An expired attempt cannot overwrite a newer attempt's state.
	if !current.RunAt.Equal(a.request.RunAt) {
		return nil
	}
	execution, err := models.LockNodeExecution(tx, current.WorkflowID, *current.ExecutionID)
	if err != nil {
		return err
	}
	if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		return current.Complete(tx)
	}
	a.state.LeaseUntil = nil
	if publishErr != nil {
		return a.scheduleRetry(tx, current, execution, publishErr)
	}
	a.state.Status, a.state.Reason, a.state.NextRetryAt = string(outcome.Status), outcome.Reason, nil
	if err := setBodySectionPublicationState(tx, execution, a.state); err != nil {
		return err
	}
	if err := contexts.NewExecutionStateContext(tx, execution, a.collectEvents).Emit("default", factorycomponent.PullRequestBodySectionUpdatedPayloadType, []any{a.state}); err != nil {
		return err
	}
	return current.Complete(tx)
}

func (a *bodySectionActionAttempt) scheduleRetry(tx *gorm.DB, current *models.CanvasNodeRequest, execution *models.CanvasNodeExecution, publishErr error) error {
	delay := min(time.Duration(a.state.Attempts)*10*time.Second, 5*time.Minute)
	next := time.Now().Add(delay)
	a.state.Status, a.state.LastError, a.state.NextRetryAt = "retrying", publishErr.Error(), &next
	if err := setBodySectionPublicationState(tx, execution, a.state); err != nil {
		return err
	}
	if err := current.Complete(tx); err != nil {
		return err
	}
	return contexts.NewExecutionRequestContext(tx, execution).ScheduleActionCall(factorycomponent.UpdatePullRequestBodySectionHook, current.Spec.Data().InvokeAction.Parameters, delay)
}

func setBodySectionPublicationState(tx *gorm.DB, execution *models.CanvasNodeExecution, state bodySectionPublicationState) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return err
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return err
	}
	metadata := maps.Clone(execution.Metadata.Data())
	if metadata == nil {
		metadata = map[string]any{}
	}
	for _, key := range []string{"reason", "lastError", "nextRetryAt", "leaseUntil"} {
		delete(metadata, key)
	}
	maps.Copy(metadata, fields)
	return contexts.NewExecutionMetadataContext(tx, execution).Set(metadata)
}
