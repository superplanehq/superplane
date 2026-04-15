package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__IsChangeManagementEnabledForCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("nil canvas returns false", func(t *testing.T) {
		enabled, err := isChangeManagementEnabledForCanvas(nil)
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("template canvas returns false", func(t *testing.T) {
		canvas := &models.Canvas{IsTemplate: true, OrganizationID: r.Organization.ID, ChangeManagementEnabled: true}
		enabled, err := isChangeManagementEnabledForCanvasInTransaction(database.Conn(), canvas)
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("organization change management enabled overrides canvas setting", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		canvas.ChangeManagementEnabled = false
		require.NoError(t, database.Conn().Save(canvas).Error)

		r.Organization.ChangeManagementEnabled = true
		require.NoError(t, database.Conn().Save(r.Organization).Error)
		defer func() {
			r.Organization.ChangeManagementEnabled = false
			_ = database.Conn().Save(r.Organization).Error
		}()

		enabled, err := isChangeManagementEnabledForCanvas(canvas)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("canvas setting used when organization has it disabled", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		canvas.ChangeManagementEnabled = true
		require.NoError(t, database.Conn().Save(canvas).Error)

		r.Organization.ChangeManagementEnabled = false
		require.NoError(t, database.Conn().Save(r.Organization).Error)

		enabled, err := isChangeManagementEnabledForCanvas(canvas)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("both disabled returns false", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		canvas.ChangeManagementEnabled = false
		require.NoError(t, database.Conn().Save(canvas).Error)

		r.Organization.ChangeManagementEnabled = false
		require.NoError(t, database.Conn().Save(r.Organization).Error)

		enabled, err := isChangeManagementEnabledForCanvas(canvas)
		require.NoError(t, err)
		assert.False(t, enabled)
	})
}
