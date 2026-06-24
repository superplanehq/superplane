package changesets

import (
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"google.golang.org/protobuf/types/known/structpb"
)

type ChangeType int

const (
	ChangeTypeUnspecified ChangeType = iota
	ChangeTypeAddNode
	ChangeTypeDeleteNode
	ChangeTypeUpdateNode
	ChangeTypeAddEdge
	ChangeTypeDeleteEdge
)

func (t ChangeType) String() string {
	switch t {
	case ChangeTypeAddNode:
		return "ADD_NODE"
	case ChangeTypeDeleteNode:
		return "DELETE_NODE"
	case ChangeTypeUpdateNode:
		return "UPDATE_NODE"
	case ChangeTypeAddEdge:
		return "ADD_EDGE"
	case ChangeTypeDeleteEdge:
		return "DELETE_EDGE"
	default:
		return "UNSPECIFIED"
	}
}

type CanvasChangeset struct {
	Changes []*Change
}

type Change struct {
	Type ChangeType
	Node *ChangeNode
	Edge *ChangeEdge
}

type ChangeNode struct {
	ID            string
	Name          string
	Block         string
	Configuration *structpb.Struct
	IntegrationID string
	Position      *componentpb.Position
	IsCollapsed   *bool
}

type ChangeEdge struct {
	SourceID string
	TargetID string
	Channel  string
}
