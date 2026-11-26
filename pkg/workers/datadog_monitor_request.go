package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const datadogMonitorLease = 5 * time.Minute
const datadogMonitorMaxAttempts = 10

type datadogMonitorAttempt struct {
	request       *models.CanvasNodeRequest
	node          *models.CanvasNode
	integration   *models.Integration
	state         models.DatadogMonitorRequest
	configuration map[string]any
	metadata      monitorMetadataBuffer
	hook          core.Trigger
}

// The hook can change its in-memory metadata only. Completion merges the owned
// fields into the locked current node after checking the lease and configuration.
type monitorMetadataBuffer struct{ value map[string]any }

func (m *monitorMetadataBuffer) Get() any { return m.value }
func (m *monitorMetadataBuffer) Set(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, &m.value)
}

func (e *NodeRequestExecutor) processDatadogMonitorRequest(id uuid.UUID, runNow bool) error {
	db := database.DB(context.Background())
	attempt := &datadogMonitorAttempt{}
	err := db.Transaction(func(tx *gorm.DB) error { return attempt.claim(tx, e, id, runNow) })
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if attempt.integration == nil {
		return nil
	}
	parameters := map[string]any{}
	if attempt.state.RetiredMonitor != nil {
		parameters["retiredMonitor"] = true
	}
	_, hookErr := attempt.hook.HandleHook(core.TriggerHookContext{
		Name:          datadog.ReconcileIntakeMonitorAction,
		Parameters:    parameters,
		Configuration: attempt.configuration,
		Metadata:      &attempt.metadata,
		HTTP:          e.registry.HTTPContext(),
		Integration:   contexts.NewIntegrationContext(db, attempt.node, attempt.integration, e.encryptor, e.registry, nil),
		Logger:        logging.WithIntegration(logging.ForNode(*attempt.node), *attempt.integration),
	})
	if hookErr != nil {
		e.logger.WithError(hookErr).WithField("request", id).Warn("Datadog monitor reconciliation attempt failed")
	}
	return db.Transaction(func(tx *gorm.DB) error { return attempt.finish(tx, hookErr) })
}

func (a *datadogMonitorAttempt) claim(tx *gorm.DB, e *NodeRequestExecutor, id uuid.UUID, runNow bool) error {
	request, err := models.LockCanvasNodeRequest(tx, id)
	if err != nil {
		return err
	}
	hook := request.Spec.Data().InvokeAction
	if request.State != models.NodeExecutionRequestStatePending || (!runNow && request.RunAt.After(time.Now())) {
		return nil
	}
	if request.Type != models.NodeRequestTypeInvokeAction || request.ExecutionID != nil || hook == nil || hook.ActionName != datadog.ReconcileIntakeMonitorAction {
		return request.Complete(tx)
	}
	a.request = request
	if state := request.Spec.Data().DatadogMonitor; state != nil {
		a.state = *state
	}
	if a.state.LeaseUntil != nil && a.state.LeaseUntil.After(time.Now()) {
		return nil
	}
	canvas, err := models.LockUnscopedCanvas(tx, request.WorkflowID)
	if err != nil {
		return err
	}
	_, err = models.FindOrganizationByIDInTransaction(tx, canvas.OrganizationID.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return request.Complete(tx)
	}
	if err != nil {
		return err
	}
	node, err := models.LockUnscopedCanvasNode(tx, request.WorkflowID, request.NodeID)
	if err != nil {
		return err
	}
	a.node = node
	until, err := request.ActiveDatadogMonitorLease(tx)
	if err != nil {
		return err
	}
	if until != nil {
		return request.Reschedule(tx, *until)
	}
	var integrationID uuid.UUID
	retired := a.state.RetiredMonitor
	if retired != nil {
		if retired.OrganizationID != canvas.OrganizationID || retired.Service == "" {
			return request.Complete(tx)
		}
		integrationID = retired.IntegrationID
		a.configuration = map[string]any{"service": retired.Service}
		a.metadata.value = map[string]any{"monitorId": retired.MonitorID, "monitorService": retired.Service}
	} else {
		if previous := a.state.Target; previous != nil {
			service, _ := node.Configuration.Data()["service"].(string)
			retiredTarget := node.DeletedAt.Valid || canvas.DeletedAt.Valid || !isDatadogMonitorNode(node) || node.AppInstallationID == nil || *node.AppInstallationID != previous.IntegrationID || strings.TrimSpace(service) != previous.Service
			if retiredTarget {
				a.state.Attempts = 0
				if err := a.scheduleRetiredMonitor(tx, node, *previous); err != nil {
					return err
				}
			}
		}
		if node.DeletedAt.Valid || canvas.DeletedAt.Valid {
			return request.Complete(tx)
		}
		if !isDatadogMonitorNode(node) {
			return request.Complete(tx)
		}
		if node.AppInstallationID == nil {
			return a.reject(tx, "Datadog is not connected.")
		}
		integrationID = *node.AppInstallationID
		a.configuration, err = contexts.NewNodeConfigurationBuilder(tx, node.WorkflowID).WithNodeID(node.NodeID).
			WithExpressionVariables(map[string]any{"parameters": hook.Parameters}).
			WithConfigurationFields((&datadog.OnErrorTrackingAlert{}).Configuration()).
			Build(contexts.WithoutRunTitleConfiguration(node.Configuration.Data()))
		if err != nil {
			return a.reject(tx, fmt.Sprintf("Cannot resolve Datadog monitor configuration: %v", err))
		}
		a.metadata.value = maps.Clone(node.Metadata.Data())
	}
	integration, err := models.FindIntegrationInTransaction(tx, canvas.OrganizationID, integrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return a.reject(tx, "Datadog is not connected.")
	}
	if err != nil {
		return err
	}
	if integration.AppName != "datadog" || integration.State != models.IntegrationStateReady {
		return a.reject(tx, "Datadog is not ready.")
	}
	a.hook, _, err = e.registry.FindTriggerHook((&datadog.OnErrorTrackingAlert{}).Name(), datadog.ReconcileIntakeMonitorAction)
	if err != nil {
		return a.reject(tx, "Datadog monitor reconciliation is unavailable.")
	}
	if retired == nil {
		service, _ := a.configuration["service"].(string)
		id, _ := a.metadata.value["monitorId"].(string)
		if a.metadata.value["monitorService"] != strings.TrimSpace(service) {
			id = ""
		}
		a.state.Target = &models.RetiredDatadogMonitor{OrganizationID: canvas.OrganizationID, IntegrationID: integration.ID, MonitorID: id, Service: strings.TrimSpace(service)}
	}
	if a.state.Attempts >= datadogMonitorMaxAttempts {
		return a.reject(tx, "Datadog monitor reconciliation failed after ten attempts.")
	}
	a.state.Attempts++
	a.state.LeaseID = uuid.NewString()
	leaseUntil := time.Now().Add(datadogMonitorLease).Truncate(time.Microsecond)
	a.state.LeaseUntil = &leaseUntil
	spec := request.Spec.Data()
	spec.DatadogMonitor = &a.state
	request.Spec = datatypes.NewJSONType(spec)
	if err := request.Reschedule(tx, leaseUntil); err != nil {
		return err
	}
	a.integration = integration
	return nil
}

func isDatadogMonitorNode(node *models.CanvasNode) bool {
	return node.Type == models.NodeTypeTrigger && node.Ref.Data().Trigger != nil && node.Ref.Data().Trigger.Name == (&datadog.OnErrorTrackingAlert{}).Name()
}

func (a *datadogMonitorAttempt) reject(tx *gorm.DB, reason string) error {
	if a.state.RetiredMonitor == nil && a.node != nil && !a.node.DeletedAt.Valid {
		a.node.State = models.CanvasNodeStateError
		a.node.StateReason = &reason
		if err := tx.Save(a.node).Error; err != nil {
			return err
		}
	}
	return a.request.Complete(tx)
}

func (a *datadogMonitorAttempt) finish(tx *gorm.DB, hookErr error) error {
	request, err := models.LockCanvasNodeRequest(tx, a.request.ID)
	if err != nil {
		return err
	}
	state := request.Spec.Data().DatadogMonitor
	ownsLease := state != nil && state.LeaseID == a.state.LeaseID
	// Lock canvas before node, as publication and retired-node cleanup do.
	canvas, err := models.LockUnscopedCanvas(tx, request.WorkflowID)
	if err != nil {
		return err
	}
	node, err := models.LockUnscopedCanvasNode(tx, request.WorkflowID, request.NodeID)
	if err != nil {
		return err
	}
	currentIntegration, err := models.FindIntegrationInTransaction(tx, canvas.OrganizationID, a.integration.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	sameIntegration := currentIntegration != nil && currentIntegration.AppName == "datadog" && currentIntegration.State == models.IntegrationStateReady &&
		reflect.DeepEqual(currentIntegration.Configuration.Data(), a.integration.Configuration.Data()) && reflect.DeepEqual(currentIntegration.Properties, a.integration.Properties)
	current := !canvas.DeletedAt.Valid && !node.DeletedAt.Valid && isDatadogMonitorNode(node) &&
		reflect.DeepEqual(node.AppInstallationID, a.node.AppInstallationID) && reflect.DeepEqual(node.Configuration.Data(), a.node.Configuration.Data()) &&
		reflect.DeepEqual(node.Ref.Data(), a.node.Ref.Data()) && sameIntegration && ownedMonitorMetadataEqual(node, a.node)
	activeLease := ownsLease && state.LeaseUntil != nil && state.LeaseUntil.After(time.Now()) && request.State == models.NodeExecutionRequestStatePending
	if a.state.RetiredMonitor != nil && activeLease && sameIntegration {
		return a.completeAttempt(tx, request, node, hookErr)
	}
	if !activeLease || !current {
		if ownsLease {
			state.LeaseID = ""
			state.LeaseUntil = nil
			spec := request.Spec.Data()
			spec.DatadogMonitor = state
			request.Spec = datatypes.NewJSONType(spec)
			if err := request.Reschedule(tx, time.Now()); err != nil {
				return err
			}
			if err := request.Complete(tx); err != nil {
				return err
			}
		}
		return a.scheduleCurrentState(tx, node, canvas)
	}
	if hookErr != nil {
		return a.completeAttempt(tx, request, node, hookErr)
	}
	metadata := maps.Clone(node.Metadata.Data())
	if metadata == nil {
		metadata = map[string]any{}
	}
	for _, key := range []string{"monitorId", "monitorService"} {
		metadata[key] = a.metadata.value[key]
	}
	node.Metadata = datatypes.NewJSONType(metadata)
	if err := contexts.NewNodeMetadataContext(tx, node).Set(metadata); err != nil {
		return err
	}
	return a.completeAttempt(tx, request, node, nil)
}

func ownedMonitorMetadataEqual(left, right *models.CanvasNode) bool {
	for _, key := range []string{"monitorId", "monitorService"} {
		if !reflect.DeepEqual(left.Metadata.Data()[key], right.Metadata.Data()[key]) {
			return false
		}
	}
	return true
}

func (a *datadogMonitorAttempt) completeAttempt(tx *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, hookErr error) error {
	state := *request.Spec.Data().DatadogMonitor
	state.LastError = ""
	if hookErr != nil {
		state.LastError = hookErr.Error()
	}
	state.LeaseID = ""
	state.LeaseUntil = nil
	spec := request.Spec.Data()
	spec.DatadogMonitor = &state
	request.Spec = datatypes.NewJSONType(spec)
	runAt := time.Now()
	retry := hookErr != nil && state.Attempts < datadogMonitorMaxAttempts && transientDatadogMonitorError(hookErr)
	if retry {
		delay := min(time.Second*time.Duration(1<<min(state.Attempts-1, 6)), time.Minute)
		runAt = runAt.Add(delay)
	}
	if err := request.Reschedule(tx, runAt); err != nil {
		return err
	}
	if retry {
		return nil
	}
	if hookErr != nil && state.RetiredMonitor == nil && !node.DeletedAt.Valid {
		reason := fmt.Sprintf("Datadog monitor reconciliation failed: %v", hookErr)
		node.State = models.CanvasNodeStateError
		node.StateReason = &reason
		if err := tx.Save(node).Error; err != nil {
			return err
		}
	}
	return request.Complete(tx)
}

func transientDatadogMonitorError(err error) bool {
	var apiError *datadog.APIError
	if errors.As(err, &apiError) {
		return apiError.StatusCode == http.StatusRequestTimeout || apiError.StatusCode == http.StatusTooManyRequests || apiError.StatusCode >= 500
	}
	var networkError net.Error
	return errors.As(err, &networkError) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}

func (a *datadogMonitorAttempt) scheduleCurrentState(tx *gorm.DB, node *models.CanvasNode, canvas *models.Canvas) error {
	// A stale successful create still needs durable cleanup, even after deletion
	// or an integration change. Ownership and current users are checked by the hook.
	targets := []models.RetiredDatadogMonitor{}
	if a.state.Target != nil {
		targets = append(targets, *a.state.Target)
	}
	for _, metadata := range []map[string]any{a.node.Metadata.Data(), a.metadata.value} {
		id, _ := metadata["monitorId"].(string)
		service, _ := metadata["monitorService"].(string)
		if id != "" && service != "" {
			targets = append(targets, models.RetiredDatadogMonitor{OrganizationID: canvas.OrganizationID, IntegrationID: a.integration.ID, MonitorID: id, Service: service})
		}
	}
	seen := map[models.RetiredDatadogMonitor]bool{}
	for _, target := range targets {
		if target.Service == "" || seen[target] {
			continue
		}
		seen[target] = true
		if err := a.scheduleRetiredMonitor(tx, node, target); err != nil {
			return err
		}
	}
	if canvas.DeletedAt.Valid || node.DeletedAt.Valid || !isDatadogMonitorNode(node) {
		return nil
	}
	pending, err := node.FindPendingDatadogMonitorRequest(tx)
	if err == nil {
		if state := pending.Spec.Data().DatadogMonitor; state != nil && state.LeaseUntil != nil && state.LeaseUntil.After(time.Now()) {
			return nil
		}
		// A request delayed by this lease can now reconcile the current state.
		runAt := time.Now().Add(time.Second)
		if pending.RunAt.Before(runAt) {
			return nil
		}
		return pending.Reschedule(tx, runAt)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	runAt := time.Now().Add(time.Second)
	return node.CreateRequest(tx, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{ActionName: datadog.ReconcileIntakeMonitorAction, Parameters: map[string]any{}},
	}, &runAt)
}

func (a *datadogMonitorAttempt) scheduleRetiredMonitor(tx *gorm.DB, node *models.CanvasNode, retired models.RetiredDatadogMonitor) error {
	runAt := time.Now().Add(time.Second)
	return node.CreateRequest(tx, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction:   &models.InvokeAction{ActionName: datadog.ReconcileIntakeMonitorAction, Parameters: map[string]any{}},
		DatadogMonitor: &models.DatadogMonitorRequest{RetiredMonitor: &retired},
	}, &runAt)
}
