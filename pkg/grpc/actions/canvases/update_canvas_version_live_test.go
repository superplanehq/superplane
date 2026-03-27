package canvases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestUpdateLiveCanvasWithoutVersioningRemapsSoftDeletedNodeIDConflicts(t *testing.T) {
	r := support.Setup(t)

	require.NoError(
		t,
		database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("versioning_enabled", false).
			Error,
	)

	canvasNode := models.CanvasNode{
		NodeID:        "node-1",
		Name:          "Noop",
		Type:          models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(models.Position{X: 0, Y: 0}),
	}

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{canvasNode}, []models.Edge{})

	require.NoError(
		t,
		database.Conn().Transaction(func(tx *gorm.DB) error {
			node, err := models.FindCanvasNode(tx, canvas.ID, "node-1")
			require.NoError(t, err)
			return models.DeleteCanvasNode(tx, *node)
		}),
	)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	emptyStruct, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)

	resp, err := UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Test"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:            "node-1",
						Name:          "Noop",
						Type:          componentpb.Node_TYPE_COMPONENT,
						Configuration: emptyStruct,
						Metadata:      emptyStruct,
						Position:      &componentpb.Position{X: 0, Y: 0},
						Component:     &componentpb.Node_ComponentRef{Name: "noop"},
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		testWebhookBaseURL,
	)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Version)
	require.NotNil(t, resp.Version.Spec)
	require.Len(t, resp.Version.Spec.Nodes, 1)
	require.Equal(t, readdedNodeIDWithAttempt(canvas.ID, "node-1", 0), resp.Version.Spec.Nodes[0].Id)
}

func TestUpdateLiveCanvasWithoutVersioningReaddAfterDeletingReaddedNodeDoesNot500(t *testing.T) {
	r := support.Setup(t)

	require.NoError(
		t,
		database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("versioning_enabled", false).
			Error,
	)

	canvasNode := models.CanvasNode{
		NodeID:        "node-1",
		Name:          "Noop",
		Type:          models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(models.Position{X: 0, Y: 0}),
	}

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{canvasNode}, []models.Edge{})

	require.NoError(
		t,
		database.Conn().Transaction(func(tx *gorm.DB) error {
			node, err := models.FindCanvasNode(tx, canvas.ID, "node-1")
			require.NoError(t, err)
			return models.DeleteCanvasNode(tx, *node)
		}),
	)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	emptyStruct, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)

	first, err := UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Test"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:            "node-1",
						Name:          "Noop",
						Type:          componentpb.Node_TYPE_COMPONENT,
						Configuration: emptyStruct,
						Metadata:      emptyStruct,
						Position:      &componentpb.Position{X: 0, Y: 0},
						Component:     &componentpb.Node_ComponentRef{Name: "noop"},
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		testWebhookBaseURL,
	)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotNil(t, first.Version)
	require.NotNil(t, first.Version.Spec)
	require.Len(t, first.Version.Spec.Nodes, 1)
	require.Equal(t, readdedNodeIDWithAttempt(canvas.ID, "node-1", 0), first.Version.Spec.Nodes[0].Id)

	require.NoError(
		t,
		database.Conn().Transaction(func(tx *gorm.DB) error {
			node, err := models.FindCanvasNode(tx, canvas.ID, readdedNodeIDWithAttempt(canvas.ID, "node-1", 0))
			require.NoError(t, err)
			return models.DeleteCanvasNode(tx, *node)
		}),
	)

	second, err := UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Test"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:            "node-1",
						Name:          "Noop",
						Type:          componentpb.Node_TYPE_COMPONENT,
						Configuration: emptyStruct,
						Metadata:      emptyStruct,
						Position:      &componentpb.Position{X: 0, Y: 0},
						Component:     &componentpb.Node_ComponentRef{Name: "noop"},
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		testWebhookBaseURL,
	)
	require.NoError(t, err)
	require.NotNil(t, second)
	require.NotNil(t, second.Version)
	require.NotNil(t, second.Version.Spec)
	require.Len(t, second.Version.Spec.Nodes, 1)
	require.Equal(t, readdedNodeIDWithAttempt(canvas.ID, "node-1", 1), second.Version.Spec.Nodes[0].Id)
}

func TestUpdateLiveCanvasWithoutVersioningRejectsMissingAppInstallationID(t *testing.T) {
	r := support.Setup(t)

	require.NoError(
		t,
		database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("versioning_enabled", false).
			Error,
	)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	emptyStruct, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)

	resp, err := UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Test"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:            "node-1",
						Name:          "Noop",
						Type:          componentpb.Node_TYPE_COMPONENT,
						Configuration: emptyStruct,
						Metadata:      emptyStruct,
						Position:      &componentpb.Position{X: 0, Y: 0},
						Component:     &componentpb.Node_ComponentRef{Name: "noop"},
						Integration:   &componentpb.IntegrationRef{Id: uuid.New().String(), Name: "missing"},
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		testWebhookBaseURL,
	)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Version)
	require.NotNil(t, resp.Version.Spec)
	require.Len(t, resp.Version.Spec.Nodes, 1)
	require.Equal(t, "integration not found", resp.Version.Spec.Nodes[0].ErrorMessage)
	require.NotContains(t, resp.Version.Spec.Nodes[0].ErrorMessage, "SQLSTATE")
	require.NotContains(t, resp.Version.Spec.Nodes[0].ErrorMessage, "violates foreign key constraint")
}

func TestUpdateLiveCanvasWithoutVersioningPersistsLongSetupErrors(t *testing.T) {
	r := support.Setup(t)

	require.NoError(
		t,
		database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("versioning_enabled", false).
			Error,
	)

	componentName := support.RandomName("failing-setup")
	setupError := strings.Repeat("setup failure with a deliberately long message. ", 8)
	expectedError := fmt.Sprintf("error setting up node node-1: %s", setupError)
	require.Greater(t, len(expectedError), 255)

	r.Registry.Components[componentName] = support.NewDummyComponent(support.DummyComponentOptions{
		SetupFunc: func(ctx core.SetupContext) error {
			return errors.New(setupError)
		},
	})

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	emptyStruct, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)

	resp, err := UpdateCanvasVersion(
		ctx,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Test"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{
					{
						Id:            "node-1",
						Name:          "Failing setup",
						Type:          componentpb.Node_TYPE_COMPONENT,
						Configuration: emptyStruct,
						Metadata:      emptyStruct,
						Position:      &componentpb.Position{X: 0, Y: 0},
						Component:     &componentpb.Node_ComponentRef{Name: componentName},
					},
				},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		testWebhookBaseURL,
	)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Version)
	require.NotNil(t, resp.Version.Spec)
	require.Len(t, resp.Version.Spec.Nodes, 1)
	require.Equal(t, expectedError, resp.Version.Spec.Nodes[0].ErrorMessage)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, "node-1")
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeStateError, node.State)
	require.NotNil(t, node.StateReason)
	require.Equal(t, expectedError, *node.StateReason)
}
