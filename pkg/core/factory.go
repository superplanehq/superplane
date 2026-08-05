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
	WorkOrderID string
	State       string
	Result      string
}

type AddWorkOrderCommentParams struct {
	WorkOrderID string
	Body        string
	AuthorKind  string
	AuthorLabel string
}

type AddWorkOrderArtifactParams struct {
	WorkOrderID string
	Type        string
	URL         string
	Title       string
	Body        string
	Data        map[string]any
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
	URL         string         `json:"url,omitempty"`
	Title       string         `json:"title,omitempty"`
	Body        string         `json:"body,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}
