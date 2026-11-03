package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const mergeConfidenceRequestLease = 5 * time.Minute

type mergeConfidencePublicationState struct {
	SourceRunID string     `json:"sourceRunId"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	Reason      string     `json:"reason,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
	NextRetryAt *time.Time `json:"nextRetryAt,omitempty"`
	LeaseUntil  *time.Time `json:"leaseUntil,omitempty"`
}

type mergeConfidenceActionAttempt struct {
	request   *models.CanvasNodeRequest
	execution *models.CanvasNodeExecution
	source    *models.CanvasRun
	state     mergeConfidencePublicationState
	events    []models.CanvasEvent
}

// ProcessMergeConfidencePublicationRequest releases the request transaction
// before provider I/O. A lease recovers interrupted attempts through the same
// scheduled action queue used by other components.
func ProcessMergeConfidencePublicationRequest(ctx context.Context, deps IntakeDependencies, requestID uuid.UUID,
	lockRequest func(*gorm.DB, uuid.UUID) (*models.CanvasNodeRequest, error),
) ([]models.CanvasEvent, error) {
	db := database.DB(ctx)
	attempt := &mergeConfidenceActionAttempt{}
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

	baseURL, _ := attempt.request.Spec.Data().InvokeAction.Parameters["baseURL"].(string)
	outcome, publishErr := PublishMergeConfidenceWithOutcome(ctx, db, deps, attempt.source.WorkflowID, attempt.source.ID, baseURL)
	err = db.Transaction(func(tx *gorm.DB) error { return attempt.finish(tx, outcome, publishErr) })
	return attempt.events, err
}

func (a *mergeConfidenceActionAttempt) collectEvents(events []models.CanvasEvent) {
	a.events = append(a.events, events...)
}

func (a *mergeConfidenceActionAttempt) claim(tx *gorm.DB) error {
	request := a.request
	hook := request.Spec.Data().InvokeAction
	if request.Type != models.NodeRequestTypeInvokeAction || hook == nil || hook.ActionName != factorycomponent.PublishMergeConfidenceHook || request.ExecutionID == nil {
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
	if component != factorycomponent.PublishMergeConfidenceComponentName {
		return a.reject(tx, "Publication request has an invalid component.")
	}
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
	var config factorycomponent.PublishMergeConfidenceConfiguration
	if err := mapstructure.Decode(execution.Configuration.Data(), &config); err != nil {
		return err
	}
	a.state.SourceRunID = config.SourceRunID
	a.state.Attempts++
	if err := a.resolveSource(tx); err != nil {
		return err
	}
	if a.state.Status == "rejected" {
		return nil
	}
	// PostgreSQL timestamps have microsecond precision. Persist and compare the
	// same value when fencing a completed attempt against a newer lease.
	until := time.Now().Add(mergeConfidenceRequestLease).Truncate(time.Microsecond)
	a.state.Status, a.state.LeaseUntil, a.state.NextRetryAt = "publishing", &until, nil
	if err := setMergeConfidencePublicationState(tx, execution, a.state); err != nil {
		return err
	}
	return request.Reschedule(tx, until)
}

func (a *mergeConfidenceActionAttempt) resolveSource(tx *gorm.DB) error {
	sourceID, err := uuid.Parse(a.state.SourceRunID)
	if err != nil {
		return a.reject(tx, "Source run ID is invalid.")
	}
	a.source, err = models.FindUnscopedCanvasRun(tx, sourceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a.reject(tx, "Source run was not found.")
	}
	if err != nil {
		return err
	}
	owner, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, a.execution.WorkflowID)
	if err != nil {
		return err
	}
	sourceCanvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, a.source.WorkflowID)
	if err != nil {
		return err
	}
	if owner.FactoryID == nil || sourceCanvas.FactoryID == nil || owner.OrganizationID != sourceCanvas.OrganizationID || *owner.FactoryID != *sourceCanvas.FactoryID {
		return a.reject(tx, "Source run must belong to this organization and factory.")
	}
	return nil
}

func (a *mergeConfidenceActionAttempt) reject(tx *gorm.DB, message string) error {
	a.state.Status, a.state.Reason = "rejected", message
	if err := setMergeConfidencePublicationState(tx, a.execution, a.state); err != nil {
		return err
	}
	if err := contexts.NewExecutionStateContext(tx, a.execution, a.collectEvents).Fail(models.CanvasNodeExecutionResultReasonError, message); err != nil {
		return err
	}
	return a.request.Complete(tx)
}

func (a *mergeConfidenceActionAttempt) finish(tx *gorm.DB, outcome MergeConfidencePublicationOutcome, publishErr error) error {
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
	a.state.Status, a.state.Reason, a.state.NextRetryAt = outcome.Status, outcome.Reason, nil
	if err := setMergeConfidencePublicationState(tx, execution, a.state); err != nil {
		return err
	}
	if err := contexts.NewExecutionStateContext(tx, execution, a.collectEvents).Emit("default", factorycomponent.MergeConfidencePublishedPayloadType, []any{a.state}); err != nil {
		return err
	}
	return current.Complete(tx)
}

func (a *mergeConfidenceActionAttempt) scheduleRetry(tx *gorm.DB, current *models.CanvasNodeRequest, execution *models.CanvasNodeExecution, publishErr error) error {
	delay := min(time.Duration(a.state.Attempts)*10*time.Second, 5*time.Minute)
	next := time.Now().Add(delay)
	a.state.Status, a.state.LastError, a.state.NextRetryAt = "retrying", publishErr.Error(), &next
	if err := setMergeConfidencePublicationState(tx, execution, a.state); err != nil {
		return err
	}
	if err := current.Complete(tx); err != nil {
		return err
	}
	return contexts.NewExecutionRequestContext(tx, execution).ScheduleActionCall(factorycomponent.PublishMergeConfidenceHook, current.Spec.Data().InvokeAction.Parameters, delay)
}

func setMergeConfidencePublicationState(tx *gorm.DB, execution *models.CanvasNodeExecution, state mergeConfidencePublicationState) error {
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
