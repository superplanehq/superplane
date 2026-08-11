package core

import "errors"

// ErrWorkOrderNotFound is returned by FindWorkOrder when nothing matches
// the given lookup. Components that treat "not found" as benign (e.g. a
// PR merge with no tracked order) check for it with errors.Is.
var ErrWorkOrderNotFound = errors.New("work order not found")

type FactoryContext interface {
	CreateWorkOrder(params WorkOrderParams) (*WorkOrder, error)
	// FindWorkOrder resolves a work order by id or by one of its
	// artifacts' keys, without requiring the current run to be attached
	// to a `factory_work_order_executions` row. Returns ErrWorkOrderNotFound
	// (wrapped) when nothing matches.
	FindWorkOrder(params FindWorkOrderParams) (*WorkOrder, error)
	// UpdateWorkOrderStatus reports whether the row actually transitioned
	// via the second return value; callers must skip downstream emits
	// when `changed` is false so a no-op doesn't leak into the timeline.
	UpdateWorkOrderStatus(params UpdateWorkOrderStatusParams) (order *WorkOrder, changed bool, err error)
	AddWorkOrderComment(params AddWorkOrderCommentParams) error
	AddWorkOrderArtifact(params AddWorkOrderArtifactParams) (*WorkOrderArtifact, error)
}

type WorkOrderParams struct {
	Title       string
	Description string
}

// FindWorkOrderParams configures FactoryContext.FindWorkOrder. By selects
// the lookup strategy: "id" resolves OrderID directly, "artifactKey"
// resolves the work order that owns the artifact tagged with ArtifactKey.
type FindWorkOrderParams struct {
	By          string
	OrderID     string
	ArtifactKey string
}

type UpdateWorkOrderStatusParams struct {
	// OrderID identifies the work order to target. Required; the
	// component field defaults to `{{ order().id }}` (the work order
	// driving the current run) but callers must always resolve and pass
	// an explicit id.
	OrderID string
	State   string
	Result  string
}

type AddWorkOrderCommentParams struct {
	// OrderID identifies the work order to target; see
	// UpdateWorkOrderStatusParams.OrderID.
	OrderID string
	Body    string
}

type AddWorkOrderArtifactParams struct {
	// OrderID identifies the work order to target; see
	// UpdateWorkOrderStatusParams.OrderID.
	OrderID string
	Type    string
	Data    map[string]any
	// Key optionally tags the artifact with a queryable key so a later
	// FindWorkOrder(by: artifactKey) can resolve the work order from it.
	Key string
}

type WorkOrder struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Result      string `json:"result,omitempty"`
}

type WorkOrderArtifact struct {
	ID          string         `json:"id"`
	WorkOrderID string         `json:"workOrderId"`
	Type        string         `json:"type"`
	Data        map[string]any `json:"data,omitempty"`
}
