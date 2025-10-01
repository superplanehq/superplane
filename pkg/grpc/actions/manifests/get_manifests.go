package manifests

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/manifest"
	pb "github.com/superplanehq/superplane/pkg/protos/manifests"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func GetManifests(ctx context.Context, req *pb.GetManifestsRequest, reg *registry.Registry) (*pb.GetManifestsResponse, error) {
	if req.Category == "" {
		return nil, status.Error(codes.InvalidArgument, "category is required")
	}

	var manifests []*manifest.TypeManifest

	switch req.Category {
	case "executor":
		manifests = reg.ManifestRegistry.GetAllExecutorManifests()
	case "event_source":
		manifests = reg.ManifestRegistry.GetAllEventSourceManifests()
	default:
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid category: %s", req.Category))
	}

	pbManifests := make([]*pb.TypeManifest, 0, len(manifests))
	for _, m := range manifests {
		pbManifest, err := serializeManifest(m)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("error serializing manifest: %v", err))
		}
		pbManifests = append(pbManifests, pbManifest)
	}

	return &pb.GetManifestsResponse{
		Manifests: pbManifests,
	}, nil
}

func serializeManifest(m *manifest.TypeManifest) (*pb.TypeManifest, error) {
	if m == nil {
		return nil, nil
	}

	fields, err := serializeFields(m.Fields)
	if err != nil {
		return nil, err
	}

	return &pb.TypeManifest{
		Type:            m.Type,
		DisplayName:     m.DisplayName,
		Description:     m.Description,
		Category:        m.Category,
		IntegrationType: m.IntegrationType,
		Icon:            m.Icon,
		Fields:          fields,
	}, nil
}

func serializeFields(fields []manifest.FieldManifest) ([]*pb.FieldManifest, error) {
	pbFields := make([]*pb.FieldManifest, 0, len(fields))
	for _, f := range fields {
		pbField, err := serializeField(&f)
		if err != nil {
			return nil, err
		}
		pbFields = append(pbFields, pbField)
	}
	return pbFields, nil
}

func serializeField(f *manifest.FieldManifest) (*pb.FieldManifest, error) {
	if f == nil {
		return nil, nil
	}

	pbField := &pb.FieldManifest{
		Name:         f.Name,
		DisplayName:  f.DisplayName,
		Type:         string(f.Type),
		Required:     f.Required,
		Description:  f.Description,
		ResourceType: f.ResourceType,
		Placeholder:  f.Placeholder,
		DependsOn:    f.DependsOn,
		Hidden:       f.Hidden,
		ItemType:     string(f.ItemType),
	}

	// Serialize options
	if len(f.Options) > 0 {
		pbField.Options = make([]*pb.Option, 0, len(f.Options))
		for _, opt := range f.Options {
			pbField.Options = append(pbField.Options, &pb.Option{
				Value:       opt.Value,
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
	}

	// Serialize default value
	if f.Default != nil {
		defaultValue, err := structpb.NewValue(f.Default)
		if err != nil {
			return nil, fmt.Errorf("error serializing default value: %v", err)
		}
		pbField.Default = defaultValue
	}

	// Serialize validation
	if f.Validation != nil {
		pbField.Validation = &pb.Validation{
			Pattern:    f.Validation.Pattern,
			CustomRule: f.Validation.CustomRule,
		}
		if f.Validation.Min != nil {
			pbField.Validation.Min = f.Validation.Min
		}
		if f.Validation.Max != nil {
			pbField.Validation.Max = f.Validation.Max
		}
		if f.Validation.MinLength != nil {
			pbField.Validation.MinLength = f.Validation.MinLength
		}
		if f.Validation.MaxLength != nil {
			pbField.Validation.MaxLength = f.Validation.MaxLength
		}
	}

	// Serialize nested fields
	if len(f.Fields) > 0 {
		nestedFields, err := serializeFields(f.Fields)
		if err != nil {
			return nil, err
		}
		pbField.Fields = nestedFields
	}

	return pbField, nil
}
