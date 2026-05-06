package contexts

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

type IntegrationSetupContext struct {
	Integration *models.Integration
}

func NewIntegrationSetupContext(integration *models.Integration) *IntegrationSetupContext {
	return &IntegrationSetupContext{
		Integration: integration,
	}
}

func (c *IntegrationSetupContext) SetStep(step core.SetupStep) error {
	if c.Integration.SetupState != nil {
		return fmt.Errorf("setup state already set")
	}

	setupState := datatypes.NewJSONType(models.SetupState{
		CurrentStep:   &step,
		PreviousSteps: []core.SetupStep{},
	})

	c.Integration.SetupState = &setupState
	return nil
}
