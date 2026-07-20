package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__SetUserCanvasPreference__StoresUpdatesAndClearsPreferences(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	preference, err := models.SetUserCanvasPreference(
		database.Conn(),
		r.Organization.ID,
		r.User,
		canvas.ID,
		boolPointer(true),
	)
	require.NoError(t, err)
	require.NotNil(t, preference.StarredAt)

	preference, err = models.SetUserCanvasPreference(
		database.Conn(),
		r.Organization.ID,
		r.User,
		canvas.ID,
		boolPointer(false),
	)
	require.NoError(t, err)
	assert.Nil(t, preference.StarredAt)
	assertUserCanvasPreferenceCount(t, canvas.ID, 0)
}

func Test__SetUserCanvasPreference__DoesNotCreateEmptyPreference(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	preference, err := models.SetUserCanvasPreference(
		database.Conn(),
		r.Organization.ID,
		r.User,
		canvas.ID,
		boolPointer(false),
	)
	require.NoError(t, err)
	assert.Nil(t, preference.StarredAt)
	assertUserCanvasPreferenceCount(t, canvas.ID, 0)

	preference, err = models.SetUserCanvasPreference(
		database.Conn(),
		r.Organization.ID,
		r.User,
		canvas.ID,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, r.Organization.ID, preference.OrganizationID)
	assert.Equal(t, r.User, preference.UserID)
	assert.Equal(t, canvas.ID, preference.CanvasID)
	assert.Nil(t, preference.StarredAt)
}

func Test__SetUserCanvasPreference__RequiresExistingCanvas(t *testing.T) {
	r := support.Setup(t)

	_, err := models.SetUserCanvasPreference(
		database.Conn(),
		r.Organization.ID,
		r.User,
		uuid.New(),
		boolPointer(true),
	)
	require.Error(t, err)
}

func Test__FindUserCanvasPreferencesForCanvases(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	withoutPreference, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	preferences, err := models.FindUserCanvasPreferencesForCanvases(
		database.Conn(),
		r.Organization.ID,
		r.User,
		nil,
	)
	require.NoError(t, err)
	assert.Empty(t, preferences)

	_, err = models.SetUserCanvasPreference(
		database.Conn(),
		r.Organization.ID,
		r.User,
		canvas.ID,
		boolPointer(true),
	)
	require.NoError(t, err)

	preferences, err = models.FindUserCanvasPreferencesForCanvases(
		database.Conn(),
		r.Organization.ID,
		r.User,
		[]uuid.UUID{canvas.ID, withoutPreference.ID},
	)
	require.NoError(t, err)
	require.Len(t, preferences, 1)
	require.Contains(t, preferences, canvas.ID)
	assert.NotContains(t, preferences, withoutPreference.ID)
	assert.NotNil(t, preferences[canvas.ID].StarredAt)
}

func assertUserCanvasPreferenceCount(t *testing.T, canvasID uuid.UUID, expected int64) {
	t.Helper()

	var count int64
	err := database.Conn().
		Model(&models.UserCanvasPreference{}).
		Where("canvas_id = ?", canvasID).
		Count(&count).
		Error
	require.NoError(t, err)
	assert.Equal(t, expected, count)
}

func boolPointer(value bool) *bool {
	return &value
}
