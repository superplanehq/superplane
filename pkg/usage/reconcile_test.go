package usage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestReconcileCanvasCount_NoMismatch(t *testing.T) {
	r := support.Setup(t)

	var published []string
	fakePublish := func(canvasID, orgID string) error {
		published = append(published, canvasID)
		return nil
	}

	reconcileCanvasCount(r.Organization.ID.String(), 0, fakePublish)

	assert.Empty(t, published)
}

func TestReconcileCanvasCount_ReEnqueuesOnMismatch(t *testing.T) {
	r := support.Setup(t)

	canvas1, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	canvas2, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	var published []string
	fakePublish := func(canvasID, orgID string) error {
		published = append(published, canvasID)
		assert.Equal(t, r.Organization.ID.String(), orgID)
		return nil
	}

	reconcileCanvasCount(r.Organization.ID.String(), 0, fakePublish)

	require.Len(t, published, 2)
	assert.Contains(t, published, canvas1.ID.String())
	assert.Contains(t, published, canvas2.ID.String())
}

func TestReconcileCanvasCount_SkipsWhenCountsMatch(t *testing.T) {
	r := support.Setup(t)

	support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

	var published []string
	fakePublish := func(canvasID, orgID string) error {
		published = append(published, canvasID)
		return nil
	}

	reconcileCanvasCount(r.Organization.ID.String(), 1, fakePublish)

	assert.Empty(t, published)
}
