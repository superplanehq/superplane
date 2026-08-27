package contexts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

type FactoryContext struct {
	tx        *gorm.DB
	canvas    *models.Canvas
	execution *models.CanvasNodeExecution

	// Optional websocket fan-out callback: invoked after every successful
	// mutation with a `reason` string (currently the event type that was
	// recorded). Wired by the node executor via WithWorkOrderUpdated.
	onWorkOrderUpdated func(factoryID, orderID, reason string)

	// Optional notification fan-out callback: invoked with a fully built
	// notification payload for mutations that should email work order
	// owners/creators. The node executor collects these and publishes
	// them after the surrounding transaction commits.
	onWorkOrderNotification func(messages.FactoryWorkOrderNotificationMessage)

	lineStepOnce   bool
	lineStepLoaded bool
	lineStepCache  lineStepInfo
}

type lineStepInfo struct {
	LineID    uuid.UUID
	LineName  string
	StepIndex int
	StepName  string
}

func NewFactoryContext(tx *gorm.DB, canvas *models.Canvas, execution *models.CanvasNodeExecution) *FactoryContext {
	return &FactoryContext{
		tx:        tx,
		canvas:    canvas,
		execution: execution,
	}
}

func (c *FactoryContext) WithWorkOrderUpdated(callback func(factoryID, orderID, reason string)) *FactoryContext {
	c.onWorkOrderUpdated = callback
	return c
}

func (c *FactoryContext) WithWorkOrderNotification(
	callback func(messages.FactoryWorkOrderNotificationMessage),
) *FactoryContext {
	c.onWorkOrderNotification = callback
	return c
}

func (c *FactoryContext) CreateWorkOrder(params core.WorkOrderParams) (*core.WorkOrder, error) {
	// A run already tied to another work order must not spawn a new one.
	_, err := models.FindWorkOrderExecutionByRunID(c.tx, c.execution.RunID)
	if err == nil {
		return nil, errors.New("cannot create work order while executing another work order")
	}
	if !errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
		return nil, err
	}

	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	sourceRunID := c.execution.RunID
	order, err := c.createFactoryWorkOrder(f, params, sourceRunID)
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(f.ID, order.ID, factory.EventTypeOrderStatusUpdated)
	return workOrderToCore(order), nil
}

func (c *FactoryContext) createFactoryWorkOrder(
	factoryModel *models.Factory,
	params core.WorkOrderParams,
	sourceRunID uuid.UUID,
) (*models.FactoryWorkOrder, error) {
	if origin := c.originFromSourceRun(sourceRunID); origin != nil {
		return factoryModel.CreateWorkOrderWithOrigin(
			c.tx,
			params.Title,
			params.Description,
			nil,
			[]uuid.UUID{},
			&sourceRunID,
			*origin,
		)
	}

	return factoryModel.CreateWorkOrder(c.tx, params.Title, params.Description, nil, []uuid.UUID{}, &sourceRunID)
}

func (c *FactoryContext) originFromSourceRun(sourceRunID uuid.UUID) *models.WorkOrderOrigin {
	if _, err := models.FindFactoryIntakeByCanvasID(c.tx, c.canvas.ID); err != nil {
		return nil
	}

	event, err := models.FindRootEventForRun(c.tx, sourceRunID)
	if err != nil {
		return nil
	}

	return models.OriginFromIntakeRootEvent(event)
}

func (c *FactoryContext) UpdateWorkOrderStatus(params core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, false, err
	}

	fromState := order.State
	changed, err := order.UpdateStatus(c.tx, models.FactoryWorkOrderStatusUpdate{
		ToState:    params.State,
		Result:     params.Result,
		Automation: c.automationRef(),
		Run:        c.runRef(),
		App:        c.appRef(),
		SkipSame:   true,
	})
	if err != nil {
		return nil, false, err
	}

	// A SkipSame no-op leaves the row untouched — skipping the fan-out
	// keeps downstream nodes from re-firing on a phantom transition when
	// the component is re-run.
	if !changed {
		return workOrderToCore(order), false, nil
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderStatusUpdated)
	c.notifyWorkOrderNotification(messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: order.OrganizationID.String(),
		FactoryID:      order.FactoryID.String(),
		OrderID:        order.ID.String(),
		EventType:      factory.EventTypeOrderStatusUpdated,
		ActorName:      c.automationName(),
		FromState:      fromState,
		ToState:        order.State,
		Result:         order.Result,
	})
	return workOrderToCore(order), true, nil
}

func (c *FactoryContext) AddWorkOrderComment(params core.AddWorkOrderCommentParams) error {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return err
	}

	body := strings.TrimSpace(params.Body)
	if body == "" {
		return errors.New("comment body is required")
	}

	// Canvas comments always attribute to `automation`; user comments
	// only come from the interactive API path.
	author := factory.WorkOrderCommentAuthor{
		Kind:       factory.CommentAuthorKindAutomation,
		Automation: c.automationRef(),
	}

	if _, err := order.RecordCommentAdded(c.tx, models.FactoryWorkOrderCommentParams{
		Body:   body,
		Author: author,
		Run:    c.runRef(),
	}); err != nil {
		return err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderCommentAdded)
	c.notifyWorkOrderNotification(messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: order.OrganizationID.String(),
		FactoryID:      order.FactoryID.String(),
		OrderID:        order.ID.String(),
		EventType:      factory.EventTypeOrderCommentAdded,
		ActorName:      c.automationName(),
		CommentBody:    body,
	})
	return nil
}

func (c *FactoryContext) AddWorkOrderArtifact(params core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, err
	}

	artifact, err := order.CreateArtifact(c.tx, models.FactoryWorkOrderArtifactParams{
		Type:       params.Type,
		Data:       params.Data,
		Key:        params.Key,
		Automation: c.automationRef(),
		Run:        c.runRef(),
	})
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderArtifactAdded)
	c.notifyWorkOrderNotification(messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: order.OrganizationID.String(),
		FactoryID:      order.FactoryID.String(),
		OrderID:        order.ID.String(),
		EventType:      factory.EventTypeOrderArtifactAdded,
		ActorName:      c.automationName(),
		ArtifactType:   artifact.Type,
	})
	return artifactToCore(artifact)
}

func (c *FactoryContext) ReportWorkOrderCheck(params core.ReportWorkOrderCheckParams) (*core.WorkOrderCheck, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, err
	}

	check, err := order.ReportCheck(c.tx, models.FactoryWorkOrderCheckParams{
		Key:        params.CheckKey,
		Name:       params.Name,
		Score:      params.Score,
		MaxScore:   params.MaxScore,
		Format:     params.Format,
		Level:      params.Level,
		Summary:    params.Summary,
		Analysis:   params.Analysis,
		Automation: c.automationRef(),
		Run:        c.runRef(),
	})
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderCheckReported)
	return checkToCore(check), nil
}

func (c *FactoryContext) SetWorkOrderStatusNote(params core.SetWorkOrderStatusNoteParams) (*core.WorkOrderStatusNote, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, err
	}

	note, err := order.SetStatusNote(c.tx, models.FactoryWorkOrderStatusNoteParams{
		Key:                 params.NoteKey,
		Kind:                params.Kind,
		Headline:            params.Headline,
		Body:                params.Body,
		CtaLabel:            params.CtaLabel,
		CtaURL:              params.CtaURL,
		ShowOnlyWhenWaiting: params.ShowOnlyWhenWaiting,
		Automation:          c.automationRef(),
		Run:                 c.runRef(),
	})
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderStatusNoteUpdated)
	c.notifyWorkOrderNotification(messages.FactoryWorkOrderNotificationMessage{
		OrganizationID:     order.OrganizationID.String(),
		FactoryID:          order.FactoryID.String(),
		OrderID:            order.ID.String(),
		EventType:          factory.EventTypeOrderStatusNoteUpdated,
		ActorName:          c.automationName(),
		StatusNoteHeadline: note.Headline,
		StatusNoteBody:     note.Body,
		StatusNoteCtaLabel: note.CtaLabel,
		StatusNoteCtaURL:   note.CtaURL,
	})
	return statusNoteToCore(order, note), nil
}

// FindWorkOrder resolves a work order by id or by an artifact key,
// independent of the current run's `factory_work_order_executions` row.
// This is what lets a plain webhook-triggered run (e.g. github.onPullRequest)
// locate a work order to act on.
func (c *FactoryContext) FindWorkOrder(params core.FindWorkOrderParams) (*core.WorkOrder, error) {
	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	var order *models.FactoryWorkOrder
	switch params.By {
	case "id":
		orderID, parseErr := uuid.Parse(params.OrderID)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid orderId %q: %w", params.OrderID, parseErr)
		}
		order, err = f.FindWorkOrder(c.tx, orderID)
	case "artifactKey":
		order, err = f.FindWorkOrderByArtifactKey(c.tx, params.ArtifactKey)
	default:
		return nil, fmt.Errorf("unknown findWorkOrder lookup %q", params.By)
	}
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
			return nil, core.ErrWorkOrderNotFound
		}
		return nil, err
	}

	return workOrderToCore(order), nil
}

// resolveWorkOrder resolves the work order a mutation should target.
// orderID is always required and explicit — the component field that
// feeds it defaults to `{{ order().id }}` (the current run's work order)
// but every caller must resolve and pass a real id, which is what lets a
// run not attached to any `factory_work_order_executions` row (e.g. a
// plain github.onPullRequest webhook run) still target a specific order.
func (c *FactoryContext) resolveWorkOrder(orderID string) (*models.FactoryWorkOrder, error) {
	if orderID == "" {
		return nil, errors.New("orderId is required")
	}

	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid orderId %q: %w", orderID, err)
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	order, err := f.FindWorkOrder(c.tx, id)
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
			return nil, fmt.Errorf("work order %q not found", orderID)
		}
		return nil, err
	}

	return order, nil
}

func (c *FactoryContext) notifyWorkOrderUpdated(factoryID, orderID uuid.UUID, reason string) {
	if c.onWorkOrderUpdated == nil {
		return
	}
	c.onWorkOrderUpdated(factoryID.String(), orderID.String(), reason)
}

func (c *FactoryContext) notifyWorkOrderNotification(message messages.FactoryWorkOrderNotificationMessage) {
	if c.onWorkOrderNotification == nil {
		return
	}
	c.onWorkOrderNotification(message)
}

// automationName picks a display name for the automation actor shown in
// notification emails: the canvas node name when known, else the app name.
func (c *FactoryContext) automationName() string {
	ref := c.automationRef()
	if ref == nil {
		return ""
	}
	if ref.NodeName != "" {
		return ref.NodeName
	}
	return ref.AppName
}

// runRef attributes emitted events back to the currently executing run.
func (c *FactoryContext) runRef() *factory.RunRef {
	if c.execution == nil {
		return nil
	}

	return &factory.RunRef{
		ID: c.execution.RunID,
	}
}

// appRef attributes emitted events back to the canvas ("app") the current
// run belongs to. Paired with runRef, this lets the timeline link straight
// to the originating run.
func (c *FactoryContext) appRef() *factory.AppRef {
	if c.canvas == nil {
		return nil
	}

	return &factory.AppRef{ID: c.canvas.ID, Name: c.canvas.Name}
}

// automationRef captures node/app/line/step identity for timeline
// attribution. Missing fields are tolerated so the event still lands.
func (c *FactoryContext) automationRef() *factory.AutomationRef {
	if c.execution == nil || c.canvas == nil {
		return nil
	}

	ref := &factory.AutomationRef{
		NodeID:  c.execution.NodeID,
		AppID:   c.canvas.ID,
		AppName: c.canvas.Name,
	}

	node, err := c.canvas.FindNode(c.execution.NodeID)
	if err == nil && node != nil {
		ref.NodeName = node.Name
	}

	if info, ok := c.lineStep(); ok {
		stepIndex := info.StepIndex
		ref.LineID = info.LineID
		ref.LineName = info.LineName
		ref.StepIndex = &stepIndex
		ref.StepName = info.StepName
	}

	return ref
}

// lineStep resolves the factory line + step that owns the current run
// so events can be attributed back to the pipeline the user configured.
// Runs not attached to a work order (e.g. `CreateWorkOrder`) return
// ok=false and callers omit the fields.
func (c *FactoryContext) lineStep() (lineStepInfo, bool) {
	if c.lineStepOnce {
		return c.lineStepCache, c.lineStepLoaded
	}
	c.lineStepOnce = true

	if c.execution == nil || c.canvas == nil || c.canvas.FactoryID == nil {
		return lineStepInfo{}, false
	}

	execution, err := models.FindWorkOrderExecutionByRunID(c.tx, c.execution.RunID)
	if err != nil {
		return lineStepInfo{}, false
	}

	// Attribute back to the line dispatch's snapshot, not the live line —
	// this is a historical fact about the traversal, so a line rename after
	// dispatch shouldn't change what earlier events say.
	dispatch, err := models.FindWorkOrderLineDispatch(c.tx, execution.LineDispatchID)
	if err != nil {
		return lineStepInfo{}, false
	}

	c.lineStepCache = lineStepInfo{
		LineID:    dispatch.LineID,
		LineName:  dispatch.LineName,
		StepIndex: execution.StepIndex,
		StepName:  execution.StepName,
	}
	c.lineStepLoaded = true
	return c.lineStepCache, true
}

func workOrderToCore(order *models.FactoryWorkOrder) *core.WorkOrder {
	return &core.WorkOrder{
		ID:          order.ID.String(),
		Title:       order.Title,
		Description: order.Description,
		State:       order.State,
		Result:      order.Result,
		Number:      order.Number,
	}
}

func pullRequestToCore(pullRequest *models.FactoryPullRequest) *core.PullRequest {
	item := &core.PullRequest{
		ID:          pullRequest.ID.String(),
		WorkOrderID: pullRequest.WorkOrderID.String(),
		Provider:    pullRequest.Provider,
		Repository:  pullRequest.Repository,
		Number:      pullRequest.Number,
		URL:         pullRequest.URL,
		Title:       pullRequest.Title,
		State:       pullRequest.State,
	}
	if pullRequest.ExternalID != nil {
		item.ExternalID = *pullRequest.ExternalID
	}
	return item
}

func (c *FactoryContext) pullRequestMatch(pullRequest *models.FactoryPullRequest) (*core.PullRequestMatch, error) {
	factoryModel, err := models.FindFactory(c.tx, pullRequest.OrganizationID, pullRequest.FactoryID)
	if err != nil {
		return nil, err
	}
	order, err := factoryModel.FindWorkOrder(c.tx, pullRequest.WorkOrderID)
	if err != nil {
		return nil, err
	}
	workOrder := workOrderToCore(order)
	workOrder.Key = factoryModel.WorkOrderKey(order.Number)
	return &core.PullRequestMatch{
		PullRequest: pullRequestToCore(pullRequest),
		WorkOrder:   workOrder,
	}, nil
}

func (c *FactoryContext) currentFactory() (*models.Factory, error) {
	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}
	return models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
}

func (c *FactoryContext) AddPullRequest(params core.AddPullRequestParams) (*core.PullRequest, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, err
	}

	pullRequest, err := order.CreatePullRequest(c.tx, models.FactoryPullRequestParams{
		Provider:   params.Provider,
		ExternalID: params.ExternalID,
		Repository: params.Repository,
		Number:     params.Number,
		URL:        params.URL,
		Title:      params.Title,
		State:      params.State,
		MergedAt:   params.MergedAt,
		ClosedAt:   params.ClosedAt,
		Automation: c.automationRef(),
		Run:        c.runRef(),
	})
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderPullRequestAdded)
	return pullRequestToCore(pullRequest), nil
}

func (c *FactoryContext) UpdatePullRequest(params core.UpdatePullRequestParams) (*core.PullRequest, error) {
	factoryModel, err := c.currentFactory()
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(params.PullRequestID)
	if err != nil {
		return nil, fmt.Errorf("invalid pullRequestId %q: %w", params.PullRequestID, err)
	}

	pullRequest, err := factoryModel.FindPullRequest(c.tx, models.FactoryPullRequestLookup{ID: id})
	if err != nil {
		if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
			return nil, core.ErrPullRequestNotFound
		}
		return nil, err
	}

	if err := pullRequest.Update(c.tx, models.FactoryPullRequestPatch{
		ExternalID: params.ExternalID,
		Repository: params.Repository,
		URL:        params.URL,
		Title:      params.Title,
		State:      params.State,
		MergedAt:   params.MergedAt,
		ClosedAt:   params.ClosedAt,
		Automation: c.automationRef(),
		Run:        c.runRef(),
	}); err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(pullRequest.FactoryID, pullRequest.WorkOrderID, factory.EventTypeOrderPullRequestUpdated)
	return pullRequestToCore(pullRequest), nil
}

func (c *FactoryContext) FindPullRequest(params core.FindPullRequestParams) (*core.PullRequestMatch, error) {
	factoryModel, err := c.currentFactory()
	if err != nil {
		return nil, err
	}

	lookup := models.FactoryPullRequestLookup{
		Provider:   params.Provider,
		ExternalID: params.ExternalID,
		Repository: params.Repository,
		Number:     params.Number,
		URL:        params.URL,
	}
	if params.ID != "" {
		id, parseErr := uuid.Parse(params.ID)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid pullRequestId %q: %w", params.ID, parseErr)
		}
		lookup.ID = id
	}

	pullRequest, err := factoryModel.FindPullRequest(c.tx, lookup)
	if err != nil {
		if errors.Is(err, models.ErrFactoryPullRequestNotFound) || errors.Is(err, models.ErrFactoryPullRequestLookupIncomplete) {
			return nil, core.ErrPullRequestNotFound
		}
		return nil, err
	}

	return c.pullRequestMatch(pullRequest)
}

func (c *FactoryContext) AddPullRequestActivity(params core.AddPullRequestActivityParams) (*core.PullRequestMatch, error) {
	if c.execution == nil {
		return nil, errors.New("run is required to add pull request activity")
	}

	factoryModel, err := c.currentFactory()
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(params.PullRequestID)
	if err != nil {
		return nil, fmt.Errorf("invalid pullRequestId %q: %w", params.PullRequestID, err)
	}

	pullRequest, err := factoryModel.FindPullRequest(c.tx, models.FactoryPullRequestLookup{ID: id})
	if err != nil {
		if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
			return nil, core.ErrPullRequestNotFound
		}
		return nil, err
	}

	if err := pullRequest.LinkRun(c.tx, c.execution.RunID, params.Description); err != nil {
		return nil, err
	}

	return c.pullRequestMatch(pullRequest)
}

func artifactToCore(artifact *models.FactoryWorkOrderArtifact) (*core.WorkOrderArtifact, error) {
	data := map[string]any{}
	if len(artifact.Data) > 0 {
		if err := json.Unmarshal(artifact.Data, &data); err != nil {
			return nil, err
		}
	}

	return &core.WorkOrderArtifact{
		ID:          artifact.ID.String(),
		WorkOrderID: artifact.WorkOrderID.String(),
		Type:        artifact.Type,
		Data:        data,
	}, nil
}

func statusNoteToCore(order *models.FactoryWorkOrder, note *models.FactoryWorkOrderStatusNote) *core.WorkOrderStatusNote {
	return &core.WorkOrderStatusNote{
		WorkOrderID:         order.ID.String(),
		Key:                 note.Key,
		Kind:                note.Kind,
		Headline:            note.Headline,
		Body:                note.Body,
		CtaLabel:            note.CtaLabel,
		CtaURL:              note.CtaURL,
		ShowOnlyWhenWaiting: note.ShowOnlyWhenWaiting,
	}
}

func checkToCore(check *models.FactoryWorkOrderCheck) *core.WorkOrderCheck {
	return &core.WorkOrderCheck{
		ID:            check.ID.String(),
		WorkOrderID:   check.WorkOrderID.String(),
		Key:           check.Key,
		Name:          check.Name,
		Score:         check.Score,
		MaxScore:      check.MaxScore,
		Format:        check.Format,
		Level:         check.Level,
		PreviousScore: check.PreviousScore,
		RecentScores:  check.RecentScores,
	}
}
