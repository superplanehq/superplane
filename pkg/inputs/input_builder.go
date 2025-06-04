package inputs

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// InputBuilder assumes that the input mappings are not ambiguous and are properly defined.
// See InputValidator to see how the inputs and outputs are validated.
type InputBuilder struct {
	stage models.Stage
}

func NewBuilder(stage models.Stage) *InputBuilder {
	return &InputBuilder{stage: stage}
}

// Build() assumes that the input definitions and mappings
// were previously validated with InputValidator.Validate().
func (b *InputBuilder) Build(tx *gorm.DB, event *models.Event) (map[string]any, error) {
	//
	// If the stage doesn't define any inputs, there's nothing for us to do here.
	//
	if len(b.stage.Inputs) == 0 {
		return map[string]any{}, nil
	}

	//
	// Find the proper set of value definitions to use for this event.
	//
	valueDefinitions, err := b.getValueDefinitionsForSource(event)
	if err != nil {
		return nil, err
	}

	//
	// Now, go through all the inputs, and calculate their values,
	// with the value definitions found above.
	//
	inputs := map[string]any{}
	for _, inputDefinition := range b.stage.Inputs {
		valueDefinition, err := b.getValueDefinition(valueDefinitions, inputDefinition.Name)
		if err != nil {
			return nil, err
		}

		value, err := b.getValue(tx, valueDefinition, event)

		//
		// Value found, just assign it, and proceed to the next input.
		//
		if err != nil {
			return nil, fmt.Errorf("could not find value for required input %s: %v", inputDefinition.Name, err)
		}

		inputs[valueDefinition.Name] = value
	}

	return inputs, nil
}

func (b *InputBuilder) getValueDefinitionsForSource(event *models.Event) ([]models.ValueDefinition, error) {
	for _, mapping := range b.stage.InputMappings {

		//
		// If when is not defined, we know this is the only mapping we have.
		//
		if mapping.When == nil {
			return mapping.Values, nil
		}

		//
		// If when is defined, we need to find the correct mapping for our source.
		//
		if mapping.When.TriggeredBy.Connection == event.SourceName {
			return mapping.Values, nil
		}
	}

	//
	// If we get here, either the caller didn't Validate() before Build(),
	// or something wrong and unexpected happened, or we have a bug.
	//
	return nil, fmt.Errorf("error finding value definitions for source %s", event.SourceName)
}

func (b *InputBuilder) getValueDefinition(valueDefinitions []models.ValueDefinition, inputName string) (*models.ValueDefinition, error) {
	for _, valueDefinition := range valueDefinitions {
		if valueDefinition.Name == inputName {
			return &valueDefinition, nil
		}
	}

	//
	// If we get here, either the caller didn't Validate() before Build(),
	// or something wrong and unexpected happened, or we have a bug.
	//
	return nil, fmt.Errorf("value definition not found for input %s", inputName)
}

func (b *InputBuilder) getValue(tx *gorm.DB, valueDefinition *models.ValueDefinition, event *models.Event) (any, error) {
	//
	// If value is defined statically, just return it.
	//
	if valueDefinition.Value != nil {
		return *valueDefinition.Value, nil
	}

	//
	// If value is defined from event data, evaluate the expression for it.
	//
	if valueDefinition.ValueFrom.EventData != nil {
		return event.EvaluateStringExpression(valueDefinition.ValueFrom.EventData.Expression)
	}

	//
	// If value is defined from inputs given to the last execution of this stage, find them.
	//
	if valueDefinition.ValueFrom.LastExecution != nil {
		lastInputs, err := b.stage.FindLastExecutionInputs(tx, valueDefinition.ValueFrom.LastExecution.Results)
		if err != nil {
			return nil, fmt.Errorf("error finding last execution inputs: %v", err)
		}

		return b.getValueFromMap(lastInputs, valueDefinition.Name)
	}

	//
	// If we get here, either the caller didn't Validate() before Build(),
	// or something wrong and unexpected happened, or we have a bug.
	//
	return nil, fmt.Errorf("error determining value for %v", valueDefinition)
}

func (b *InputBuilder) getValueFromMap(m map[string]any, inputName string) (any, error) {
	if value, exists := m[inputName]; exists {
		return value, nil
	}

	return nil, fmt.Errorf("value for %s not found in map: %v", inputName, m)
}
