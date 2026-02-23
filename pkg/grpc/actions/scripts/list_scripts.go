package scripts

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
)

func ListScripts(ctx context.Context, organizationID string) (*pb.ListScriptsResponse, error) {
	scripts, err := models.FindScriptsByOrganization(organizationID)
	if err != nil {
		return nil, err
	}

	protoScripts := make([]*pb.Script, len(scripts))
	for i, s := range scripts {
		protoScripts[i] = SerializeScript(&s)
	}

	return &pb.ListScriptsResponse{
		Scripts: protoScripts,
	}, nil
}
