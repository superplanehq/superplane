package core

type FactoryContext interface {
	CreateWorkOrder(params WorkOrderParams) (*WorkOrder, error)
}

type WorkOrderParams struct {
	Title       string
	Description string
}

type WorkOrder struct {
	ID          string
	Title       string
	Description string
}
