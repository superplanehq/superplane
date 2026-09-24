package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestCreateCanvasDuplicateName(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	baseURL := "https://example.com"
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, "Duplicate Canvas", "", nil, nil, nil)
	require.NoError(t, err)

	_, err = CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, "Duplicate Canvas", "", nil, nil, nil)
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}

func TestCreateCanvasNameUniquenessIsScopedToWorkspace(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "https://example.com"

	firstWorkspace, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	secondWorkspace, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	createPlan := func(factoryID *uuid.UUID) error {
		_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, "Plan", "", factoryID, nil, nil)
		return err
	}

	// Every workspace installs the same templates, so a second workspace has to
	// be able to keep the template name instead of falling back to "Plan (2)".
	t.Run("two workspaces can both hold an app with the same name", func(t *testing.T) {
		require.NoError(t, createPlan(&firstWorkspace.ID))
		require.NoError(t, createPlan(&secondWorkspace.ID))
	})

	t.Run("one workspace rejects two apps with the same name", func(t *testing.T) {
		err := createPlan(&firstWorkspace.ID)
		require.Error(t, err)
		require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
	})

	t.Run("apps outside a workspace stay unique per organization", func(t *testing.T) {
		require.NoError(t, createPlan(nil))

		err := createPlan(nil)
		require.Error(t, err)
		require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
	})
}

func TestCreateCanvasRejectsWhitespaceOnlyName(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	baseURL := "https://example.com"
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, "   ", "", nil, nil, nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, "canvas name is required", func() string {
		_, msg, ok := grpcerrors.HandlerStatus(err)
		if ok {
			return msg
		}
		return err.Error()
	}())
}

func TestCreateCanvasOnFreshOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	baseURL := "https://example.com"
	response, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, "Health Check Monitor", "Quick start canvas on a fresh organization", nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	require.Equal(t, "Health Check Monitor", response.Canvas.Metadata.Name)
	require.Equal(t, r.Organization.ID.String(), response.Canvas.Metadata.OrganizationId)
	require.NotEmpty(t, response.Canvas.Metadata.Id)

	canvasID, err := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, err)
	persisted, err := models.FindCanvas(r.Organization.ID, canvasID)
	require.NoError(t, err)
	require.Equal(t, "Health Check Monitor", persisted.Name)
	require.Equal(t, r.Organization.ID, persisted.OrganizationID)

	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), persisted)
	require.NoError(t, err)
	require.Empty(t, liveVersion.Nodes)
	require.Empty(t, liveVersion.Edges)
}
