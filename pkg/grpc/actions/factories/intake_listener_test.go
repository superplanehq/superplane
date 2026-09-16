package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__MarkUnknownListenersDown(t *testing.T) {
	boundCanvas := uuid.New()
	boundIntake := uuid.New()
	unboundIntake := uuid.New()
	intakes := []models.FactoryIntake{
		{ID: boundIntake, CanvasID: boundCanvas},
		{ID: unboundIntake, CanvasID: uuid.New()},
	}
	listening := map[uuid.UUID]bool{
		boundIntake:   true,
		unboundIntake: true,
	}

	markUnknownListenersDown(listening, intakes, map[uuid.UUID]string{
		boundCanvas: intakeTriggerNodeID,
	})

	assert.False(t, listening[boundIntake])
	assert.True(t, listening[unboundIntake])
}
