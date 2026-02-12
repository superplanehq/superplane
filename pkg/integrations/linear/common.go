package linear

import "github.com/superplanehq/superplane/pkg/core"

type NodeMetadata struct {
	Team *Team `json:"team"`
}

func GetTeamFromMetadata(ctx core.ExecutionContext) *Team {
	metadata, ok := ctx.NodeMetadata().Metadata().(NodeMetadata)
	if !ok {
		return nil
	}
	return metadata.Team
}
