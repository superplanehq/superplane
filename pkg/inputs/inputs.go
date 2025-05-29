package inputs

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
)

type InputResolver struct {
	definitions []models.InputDefinition
	assignments []models.InputAssignment
}

func NewInputResolver(definitions []models.InputDefinition, assignments []models.InputAssignment) *InputResolver {
	return &InputResolver{
		definitions: definitions,
		assignments: assignments,
	}
}

func (r *InputResolver) findAssignment(inputName string) (*models.InputAssignment, error) {
	for _, assignment := range r.assignments {
		if assignment.Name == inputName {
			return &assignment, nil
		}
	}

	return nil, fmt.Errorf("assignment for input %s not found", inputName)
}

func (r *InputResolver) Resolve(event *models.Event) (map[string]any, error) {
	inputs := map[string]any{}

	for _, definition := range r.definitions {
		assignment, err := r.findAssignment(definition.Name)

		//
		// If assignment was not found, we check if this input is required and has a default.
		//
		if err != nil {
			if definition.Required {
				return nil, fmt.Errorf("%s is not assigned", definition.Name)
			}

			if definition.Default == nil {
				return nil, fmt.Errorf("%s is not assigned, is not required, but does not have a default", definition.Name)
			}

			inputs[definition.Name] = definition.Default
			continue
		}

		//
		// If the assignment for the input exists,
		// we use it to find the value for this input.
		//
		value, err := assignment.GetValue(event)
		if err != nil {
			return nil, fmt.Errorf("error assigning value to %s: %v", definition.Name, err)
		}

		inputs[definition.Name] = value
	}

	return inputs, nil
}
