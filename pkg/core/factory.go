package core

type FactoryContext interface {
	CreateWorkOrder(params WorkOrderParams) (*WorkOrder, error)
	// UpdateWorkOrderStatus reports whether the row actually transitioned
	// via the second return value; callers must skip downstream emits
	// when `changed` is false so a no-op doesn't leak into the timeline.
	UpdateWorkOrderStatus(params UpdateWorkOrderStatusParams) (order *WorkOrder, changed bool, err error)
	AddWorkOrderComment(params AddWorkOrderCommentParams) error
	AddWorkOrderArtifact(params AddWorkOrderArtifactParams) (*WorkOrderArtifact, error)
	// LinkedWorkOrder returns the work order attached to the current canvas
	// run when this execution was factory-dispatched. ok is false when the
	// run is not linked (not an error).
	LinkedWorkOrder() (link *LinkedWorkOrder, ok bool, err error)
}

// LinkedWorkOrder identifies a factory work order attached to a canvas run.
type LinkedWorkOrder struct {
	ID              string
	FactoryID       string
	CreatedByUserID string
}

type WorkOrderParams struct {
	Title       string
	Description string
}

type UpdateWorkOrderStatusParams struct {
	State  string
	Result string
}

type AddWorkOrderCommentParams struct {
	Body string
}

type AddWorkOrderArtifactParams struct {
	Type string
	Data map[string]any
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
