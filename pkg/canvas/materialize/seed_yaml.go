package materialize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"gopkg.in/yaml.v3"
)

// CanvasYAML is the canonical YAML representation of a canvas spec used when
// seeding or backfilling git repositories.
type CanvasYAML struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   CanvasYAMLMetadata `json:"metadata" yaml:"metadata"`
	Spec       CanvasYAMLSpec     `json:"spec" yaml:"spec"`
}

type CanvasYAMLMetadata struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type CanvasYAMLSpec struct {
	Nodes                   []models.Node
	Edges                   []models.Edge
	ChangeManagementEnabled bool
	ChangeRequestApprovers  []models.CanvasChangeRequestApprover
}

type canvasYAMLResource struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   CanvasYAMLMetadata `json:"metadata" yaml:"metadata"`
	Spec       canvasYAMLSpec     `json:"spec" yaml:"spec"`
}

type canvasYAMLSpec struct {
	Nodes            []*componentpb.Node         `json:"nodes" yaml:"nodes"`
	Edges            []*componentpb.Edge         `json:"edges" yaml:"edges"`
	ChangeManagement *pb.Canvas_ChangeManagement `json:"changeManagement,omitempty" yaml:"changeManagement,omitempty"`
}

func CanvasYAMLFromVersion(version *models.CanvasVersion) *CanvasYAML {
	if version == nil {
		return nil
	}

	return &CanvasYAML{
		APIVersion: "v1",
		Kind:       "Canvas",
		Metadata: CanvasYAMLMetadata{
			Name:        version.Name,
			Description: version.Description,
		},
		Spec: CanvasYAMLSpec{
			Nodes:                   version.Nodes,
			Edges:                   version.Edges,
			ChangeManagementEnabled: version.ChangeManagementEnabled,
			ChangeRequestApprovers:  version.EffectiveChangeRequestApprovers(),
		},
	}
}

func BuildCanvasYAMLFromCanvas(canvas *CanvasYAML) ([]byte, error) {
	if canvas == nil {
		return nil, fmt.Errorf("canvas yaml is required")
	}

	apiVersion := strings.TrimSpace(canvas.APIVersion)
	if apiVersion == "" {
		apiVersion = "v1"
	}

	kind := strings.TrimSpace(canvas.Kind)
	if kind == "" {
		kind = "Canvas"
	}

	resource := canvasYAMLResource{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata:   canvas.Metadata,
		Spec: canvasYAMLSpec{
			Nodes: actions.NodesToProto(canvas.Spec.Nodes),
			Edges: actions.EdgesToProto(canvas.Spec.Edges),
		},
	}

	if canvas.Spec.ChangeManagementEnabled || len(canvas.Spec.ChangeRequestApprovers) > 0 {
		resource.Spec.ChangeManagement = serializeChangeManagement(
			canvas.Spec.ChangeManagementEnabled,
			canvas.Spec.ChangeRequestApprovers,
		)
	}

	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshal canvas yaml: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return nil, fmt.Errorf("encode canvas yaml: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return nil, fmt.Errorf("encode canvas yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("encode canvas yaml: %w", err)
	}

	return buf.Bytes(), nil
}

func BuildConsoleYAMLFromDashboard(console *models.ConsoleYAML) ([]byte, error) {
	if console == nil {
		return BuildEmptyConsoleYAML("", "")
	}

	jsonBytes, err := json.Marshal(console)
	if err != nil {
		return nil, fmt.Errorf("marshal console yaml: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return nil, fmt.Errorf("encode console yaml: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return nil, fmt.Errorf("encode console yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("encode console yaml: %w", err)
	}

	return buf.Bytes(), nil
}

func BuildEmptyConsoleYAML(canvasID, canvasName string) ([]byte, error) {
	if strings.TrimSpace(canvasID) == "" {
		return models.CanvasVersionToConsoleYML(&models.CanvasVersion{Name: canvasName})
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas id: %w", err)
	}

	return models.CanvasVersionToConsoleYML(&models.CanvasVersion{
		WorkflowID: canvasUUID,
		Name:       canvasName,
	})
}

func BuildConsoleYAMLFromVersion(version *models.CanvasVersion) ([]byte, error) {
	if version == nil {
		return BuildEmptyConsoleYAML("", "")
	}

	return models.CanvasVersionToConsoleYML(version)
}

func serializeChangeManagement(
	enabled bool,
	approvers []models.CanvasChangeRequestApprover,
) *pb.Canvas_ChangeManagement {
	if !enabled && len(approvers) == 0 {
		return nil
	}

	protoApprovers := make([]*pb.Canvas_ChangeManagement_Approver, 0, len(approvers))
	for _, approver := range approvers {
		item := &pb.Canvas_ChangeManagement_Approver{}
		switch approver.Type {
		case models.CanvasChangeRequestApproverTypeAnyone:
			item.Type = pb.Canvas_ChangeManagement_Approver_TYPE_ANYONE
		case models.CanvasChangeRequestApproverTypeUser:
			item.Type = pb.Canvas_ChangeManagement_Approver_TYPE_USER
			item.UserId = approver.User
		case models.CanvasChangeRequestApproverTypeRole:
			item.Type = pb.Canvas_ChangeManagement_Approver_TYPE_ROLE
			item.RoleName = approver.Role
		}
		protoApprovers = append(protoApprovers, item)
	}

	return &pb.Canvas_ChangeManagement{
		Enabled:   enabled,
		Approvals: protoApprovers,
	}
}
