package contexts

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// NewFactoryExecutionStateContext composes owner-prepared data with ordinary persistence.
func NewFactoryExecutionStateContext(tx *gorm.DB, execution *models.CanvasNodeExecution, onNewEvents func([]models.CanvasEvent)) *ExecutionStateContext {
	state := NewExecutionStateContext(tx, execution, onNewEvents)
	state.outputType, state.outputs, state.outputError = mergeconfidence.PrepareOutputs(tx, execution)
	return state
}

func PrepareFactoryAgentContext(tx *gorm.DB, ctx *core.ExecutionContext, component string) error {
	if !models.IsAgentRunnerComponent(component) {
		return nil
	}
	return mergeconfidence.PrepareRunner(tx, ctx)
}
