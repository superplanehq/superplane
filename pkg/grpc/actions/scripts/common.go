package scripts

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeScript(in *models.Script) *pb.Script {
	s := &pb.Script{
		Id:             in.ID.String(),
		OrganizationId: in.OrganizationID.String(),
		Name:           in.Name,
		Label:          in.Label,
		Description:    in.Description,
		Source:         in.Source,
		ManifestJson:   string(in.Manifest),
		Status:         in.Status,
	}

	if in.CreatedBy != nil {
		s.CreatedBy = in.CreatedBy.String()
	}

	if in.CreatedAt != nil {
		s.CreatedAt = timestamppb.New(*in.CreatedAt)
	}

	if in.UpdatedAt != nil {
		s.UpdatedAt = timestamppb.New(*in.UpdatedAt)
	}

	return s
}
