package core

type FactoryContext interface {
	CreateWorkOrder(params WorkOrderParams) (*WorkOrder, error)
	UpdateWorkOrderStatus(params UpdateWorkOrderStatusParams) (*WorkOrder, error)
	AddWorkOrderComment(params AddWorkOrderCommentParams) error
	AddWorkOrderArtifact(params AddWorkOrderArtifactParams) (*WorkOrderArtifact, error)
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
