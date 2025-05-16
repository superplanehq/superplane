package actions

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListTags(ctx context.Context, req *pb.ListTagsRequest) (*pb.ListTagsResponse, error) {
	//
	// StageId is not required, but if it is specified,
	// we should validate it is a proper UUID.
	//
	if req.StageId != "" {
		err := ValidateUUIDs(req.StageId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid stage ID")
		}
	}

	tags, err := models.ListStageTags(req.Name, req.Value, tagStatesFromProto(req.States), req.StageId, "")
	if err != nil {
		log.Errorf("Error listing tags: %v", err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &pb.ListTagsResponse{
		Tags: serializeStageTags(tags),
	}, nil
}

func tagStatesFromProto(in []pb.Tag_State) []string {
	out := []string{}
	for _, i := range in {
		out = append(out, tagStateFromProto(i))
	}

	return out
}

func serializeStageTags(in []models.StageTag) []*pb.StageTag {
	out := []*pb.StageTag{}

	for _, i := range in {
		out = append(out, &pb.StageTag{
			StageId:         i.StageID.String(),
			StageEventState: stateToProto(i.EventState),
			Tag: &pb.Tag{
				Name:  i.TagName,
				Value: i.TagValue,
				State: tagStateToProto(i.TagState),
			},
		})
	}

	return out
}
