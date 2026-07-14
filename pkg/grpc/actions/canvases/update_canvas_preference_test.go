package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
)

func Test__UpdateCanvasPreference__StoresAndClearsPreferences(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	response, err := UpdateCanvasPreference(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.UpdateCanvasPreferenceRequest{
		CanvasId:                  canvas.ID.String(),
		Pinned:                    proto.Bool(true),
		Starred:                   proto.Bool(true),
		AutoLayoutOnUpdateEnabled: proto.Bool(true),
	})
	require.NoError(t, err)
	require.NotNil(t, response.Preference)
	assert.True(t, response.Preference.Pinned)
	assert.True(t, response.Preference.Starred)
	assert.True(t, response.Preference.AutoLayoutOnUpdateEnabled)
	assert.NotNil(t, response.Preference.PinnedAt)
	assert.NotNil(t, response.Preference.StarredAt)

	var count int64
	err = database.DB(context.Background()).Model(&models.UserCanvasPreference{}).
		Where("organization_id = ?", r.Organization.ID).
		Where("user_id = ?", r.User).
		Where("canvas_id = ?", canvas.ID).
		Count(&count).
		Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	response, err = UpdateCanvasPreference(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.UpdateCanvasPreferenceRequest{
		CanvasId:                  canvas.ID.String(),
		Pinned:                    proto.Bool(false),
		Starred:                   proto.Bool(false),
		AutoLayoutOnUpdateEnabled: proto.Bool(false),
	})
	require.NoError(t, err)
	require.NotNil(t, response.Preference)
	assert.False(t, response.Preference.Pinned)
	assert.False(t, response.Preference.Starred)
	assert.False(t, response.Preference.AutoLayoutOnUpdateEnabled)
	assert.Nil(t, response.Preference.PinnedAt)
	assert.Nil(t, response.Preference.StarredAt)

	err = database.DB(context.Background()).Model(&models.UserCanvasPreference{}).
		Where("organization_id = ?", r.Organization.ID).
		Where("user_id = ?", r.User).
		Where("canvas_id = ?", canvas.ID).
		Count(&count).
		Error
	require.NoError(t, err)
	assert.Zero(t, count)
}
