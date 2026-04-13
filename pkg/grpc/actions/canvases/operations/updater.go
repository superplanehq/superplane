package operations

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

type Updater struct {
	canvas   *models.CanvasVersion
	newNodes map[string]models.Node
}

func NewOperator(canvas *models.CanvasVersion) *Updater {
	return &Updater{
		canvas:   canvas,
		newNodes: make(map[string]models.Node),
	}
}

func (u *Updater) Execute(operations []*pb.CanvasUpdateOperation) error {
	for _, operation := range operations {
		err := u.executeOperation(operation)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *Updater) executeOperation(operation *pb.CanvasUpdateOperation) error {
	switch operation.Type {
	case pb.CanvasUpdateOperation_ADD_NODE:
		return u.addNode(operation)
	case pb.CanvasUpdateOperation_DELETE_NODE:
		return u.deleteNode(operation)
	case pb.CanvasUpdateOperation_UPDATE_NODE:
		return u.updateNode(operation)
	case pb.CanvasUpdateOperation_CONNECT_NODES:
		return u.connectNodes(operation)
	case pb.CanvasUpdateOperation_DISCONNECT_NODES:
		return u.disconnectNodes(operation)
	}

	return fmt.Errorf("unknown operation type: %s", operation.Type)
}

func (u *Updater) addNode(operation *pb.CanvasUpdateOperation) error {
	return fmt.Errorf("not implemented")
}

func (u *Updater) deleteNode(operation *pb.CanvasUpdateOperation) error {
	return fmt.Errorf("not implemented")
}

func (u *Updater) updateNode(operation *pb.CanvasUpdateOperation) error {
	return fmt.Errorf("not implemented")
}

func (u *Updater) connectNodes(operation *pb.CanvasUpdateOperation) error {
	return fmt.Errorf("not implemented")
}

func (u *Updater) disconnectNodes(operation *pb.CanvasUpdateOperation) error {
	return fmt.Errorf("not implemented")
}
