package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func TestIntakeRunPlacement(t *testing.T) {
	runID := uuid.New()
	order := models.FactoryWorkOrder{ID: uuid.New(), State: models.FactoryWorkOrderStateIntake}

	t.Run("active pre-promotion run is analyzing", func(t *testing.T) {
		run := models.CanvasRun{ID: runID, State: models.CanvasRunStateStarted}
		context := intakeRunContext{
			orders: map[uuid.UUID]models.FactoryWorkOrder{runID: order},
			stages: map[uuid.UUID]string{},
		}

		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_ANALYZING, intakeRunPlacement(run, context))
	})

	t.Run("successful promotion enters the backlog", func(t *testing.T) {
		run := models.CanvasRun{ID: runID, State: models.CanvasRunStateFinished, Result: models.CanvasRunResultPassed}
		context := intakeRunContext{
			orders: map[uuid.UUID]models.FactoryWorkOrder{
				runID: {ID: order.ID, State: models.FactoryWorkOrderStateDraft},
			},
			stages: map[uuid.UUID]string{},
		}

		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_BACKLOG, intakeRunPlacement(run, context))
	})

	t.Run("dispatched promotion is progressed", func(t *testing.T) {
		run := models.CanvasRun{ID: runID, State: models.CanvasRunStateFinished, Result: models.CanvasRunResultPassed}
		context := intakeRunContext{
			orders: map[uuid.UUID]models.FactoryWorkOrder{
				runID: {ID: order.ID, State: models.FactoryWorkOrderStateOpen},
			},
			stages: map[uuid.UUID]string{runID: "Implement"},
		}

		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_PROGRESSED, intakeRunPlacement(run, context))
	})

	t.Run("completed false branch stays below threshold", func(t *testing.T) {
		run := models.CanvasRun{ID: runID, State: models.CanvasRunStateFinished, Result: models.CanvasRunResultPassed}
		context := intakeRunContext{
			orders: map[uuid.UUID]models.FactoryWorkOrder{runID: order},
			stages: map[uuid.UUID]string{},
		}

		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_BELOW_THRESHOLD, intakeRunPlacement(run, context))
	})

	t.Run("failed run is rejected", func(t *testing.T) {
		run := models.CanvasRun{ID: runID, State: models.CanvasRunStateFinished, Result: models.CanvasRunResultFailed}
		context := intakeRunContext{
			orders: map[uuid.UUID]models.FactoryWorkOrder{runID: order},
			stages: map[uuid.UUID]string{},
		}

		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_REJECTED, intakeRunPlacement(run, context))
	})
}
