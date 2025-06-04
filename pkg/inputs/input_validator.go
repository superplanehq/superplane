package inputs

import (
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
)

//
// InputValidator checks for improper input and output specs.
//
// Checks:
// - if there is a mapping with no `when`, len(inputMappings) must be 1
// - if input is specified, input value definitions must exist for it in all mappings
// - input value definitions point to existing input definitions
// - connection names in mappings reference existing connections
// - cannot have multiple mappings with the same when.triggeredBy.connection
// - cannot have multiple value definitions for the same input
//

type Validator struct {
	Inputs        []*pb.InputDefinition
	InputMappings []*pb.InputMapping
	Outputs       []*pb.OutputDefinition
	Connections   []*pb.Connection
}

func NewValidator(options ...func(*Validator)) *Validator {
	validator := &Validator{}
	for _, o := range options {
		o(validator)
	}

	return validator
}

func WithInputs(inputs []*pb.InputDefinition) func(*Validator) {
	return func(v *Validator) {
		v.Inputs = inputs
	}
}

func WithInputMappings(inputMappings []*pb.InputMapping) func(*Validator) {
	return func(v *Validator) {
		v.InputMappings = inputMappings
	}
}

func WithOutputs(outputs []*pb.OutputDefinition) func(*Validator) {
	return func(v *Validator) {
		v.Outputs = outputs
	}
}

func WithConnections(connections []*pb.Connection) func(*Validator) {
	return func(v *Validator) {
		v.Connections = connections
	}
}

func (v *Validator) Validate() error {
	return v.executeUntilFirstError(
		func() error { return v.checkInputs() },
		func() error { return v.checkOutputs() },
		func() error { return v.checkInputMappings() },
	)
}

func (v *Validator) checkInputs() error {
	inputs := map[string]bool{}
	for _, input := range v.Inputs {
		if input.Name == "" {
			return fmt.Errorf("empty input name")
		}

		if _, ok := inputs[input.Name]; ok {
			return fmt.Errorf("input %s defined multiple times", input.Name)
		}

		inputs[input.Name] = true
	}

	return nil
}

func (v *Validator) checkOutputs() error {
	outputs := map[string]bool{}
	for _, output := range v.Outputs {
		if output.Name == "" {
			return fmt.Errorf("empty output name")
		}

		if _, ok := outputs[output.Name]; ok {
			return fmt.Errorf("output %s defined multiple times", output.Name)
		}

		outputs[output.Name] = true
	}

	return nil
}

func (v *Validator) checkInputMappings() error {
	return v.executeUntilFirstError(
		func() error { return v.checkInputMappingSpecs() },
		func() error { return v.checkForValidWhenLessMapping() },
		func() error { return v.checkNoDuplicateMappingForConnection() },
		func() error { return v.checkAllInputsAreDefined() },
		func() error { return v.checkValidInputDefinitionReferences() },
		func() error { return v.checkNoDuplicateValueDefinitions() },
		func() error { return v.checkValidConnectionReferences() },
	)
}

// If mapping without when is defined, it must be the only one.
func (v *Validator) checkForValidWhenLessMapping() error {
	if v.HasWhenLessMapping() && len(v.InputMappings) > 1 {
		return fmt.Errorf("mappings > 1, but mapping without when found")
	}

	return nil
}

func (v *Validator) checkNoDuplicateMappingForConnection() error {
	if v.HasWhenLessMapping() {
		return nil
	}

	connections := map[string]bool{}
	for _, mapping := range v.InputMappings {
		connection := mapping.When.TriggeredBy.Connection
		if _, ok := connections[connection]; ok {
			return fmt.Errorf("multiple input mappings for connection %s", connection)
		}

		connections[connection] = true
	}

	return nil
}

// All inputs are defined in all existing mappings
func (v *Validator) checkAllInputsAreDefined() error {
	for _, input := range v.Inputs {
		for i, m := range v.InputMappings {
			defined := slices.ContainsFunc(m.Values, func(def *pb.ValueDefinition) bool {
				return def.Name == input.Name
			})

			if !defined {
				return fmt.Errorf("mapping [%d]: input %s not defined", i, input.Name)
			}
		}
	}

	return nil
}

// All value definitions reference an existing input definition
func (v *Validator) checkValidInputDefinitionReferences() error {
	for mappingIndex, mapping := range v.InputMappings {
		for valueDefIndex, valueDef := range mapping.Values {
			validReference := slices.IndexFunc(v.Inputs, func(input *pb.InputDefinition) bool {
				return input.Name == valueDef.Name
			}) > -1

			if !validReference {
				return fmt.Errorf(
					"mapping [%d]: value definition [%d]: input %s not defined",
					mappingIndex, valueDefIndex, valueDef.Name,
				)
			}
		}
	}

	return nil
}

func (v *Validator) checkNoDuplicateValueDefinitions() error {
	for mappingIndex, mapping := range v.InputMappings {
		defs := map[string]bool{}
		for _, valueDef := range mapping.Values {
			if _, ok := defs[valueDef.Name]; ok {
				return fmt.Errorf("mapping [%d]: input %s defined multiple times", mappingIndex, valueDef.Name)
			}

			defs[valueDef.Name] = true
		}
	}

	return nil
}

func (v *Validator) checkValidConnectionReferences() error {
	for mappingIndex, mapping := range v.InputMappings {

		// Check if triggeredBy.Connection references existing connections
		if mapping.When != nil && mapping.When.TriggeredBy != nil {
			connection := mapping.When.TriggeredBy.Connection
			exists := slices.ContainsFunc(v.Connections, func(conn *pb.Connection) bool {
				return conn.Name == connection
			})

			if !exists {
				return fmt.Errorf("mapping [%d]: connection %s does not exist", mappingIndex, connection)
			}
		}

		// Check if all valueFrom.EventData.Connection references existing connections
		for valueDefIndex, valueDef := range mapping.Values {
			if valueDef.ValueFrom != nil && valueDef.ValueFrom.EventData != nil {
				connection := valueDef.ValueFrom.EventData.Connection
				exists := slices.ContainsFunc(v.Connections, func(conn *pb.Connection) bool {
					return conn.Name == connection
				})

				if !exists {
					return fmt.Errorf(
						"mapping [%d]: value definition [%d]: connection %s does not exist",
						mappingIndex, valueDefIndex, connection,
					)
				}
			}
		}
	}

	return nil
}

func (v *Validator) HasWhenLessMapping() bool {
	return slices.IndexFunc(v.InputMappings, func(mapping *pb.InputMapping) bool {
		return mapping.When == nil
	}) > -1
}

func (v *Validator) executeUntilFirstError(fn ...func() error) error {
	for _, f := range fn {
		err := f()
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) SerializeInputs() []models.InputDefinition {
	inputs := []models.InputDefinition{}
	for _, input := range v.Inputs {
		inputDefinition := models.InputDefinition{
			Name:        input.Name,
			Description: input.Description,
		}

		inputs = append(inputs, inputDefinition)
	}

	return inputs
}

func (v *Validator) SerializeOutputs() []models.OutputDefinition {
	outputs := []models.OutputDefinition{}
	for _, output := range v.Outputs {
		outputDefinition := models.OutputDefinition{
			Name:        output.Name,
			Description: output.Description,
			Required:    output.Required,
		}

		outputs = append(outputs, outputDefinition)
	}

	return outputs
}

func (v *Validator) checkInputMappingSpecs() error {
	for mappingIndex, mapping := range v.InputMappings {
		if len(mapping.Values) == 0 {
			return fmt.Errorf("invalid mapping [%d]: no value definitions", mappingIndex)
		}

		if mapping.When != nil {
			err := validateInputMappingWhen(mapping.When)
			if err != nil {
				return fmt.Errorf("invalid mapping [%d]: %v", mappingIndex, err)
			}
		}

		for _, valueDefinition := range mapping.Values {
			err := validateValueDefinition(valueDefinition)
			if err != nil {
				return fmt.Errorf("invalid mapping [%d]: %v", mappingIndex, err)
			}
		}
	}

	return nil
}

func validateValueDefinition(in *pb.ValueDefinition) error {
	if in.Name == "" {
		return fmt.Errorf("missing input name")
	}

	if in.ValueFrom == nil && in.Value == "" {
		return fmt.Errorf("value is not defined")
	}

	if in.ValueFrom != nil && in.Value != "" {
		return fmt.Errorf("cannot use value and valueFrom at the same time")
	}

	if in.ValueFrom != nil {
		err := validateValueFrom(in.ValueFrom)
		if err != nil {
			return fmt.Errorf("invalid valueFrom for %s: %v", in.Name, err)
		}
	}

	return nil
}

// TODO: should we use an enum here too?
func validateValueFrom(in *pb.ValueFrom) error {
	if in.EventData == nil && in.LastExecution == nil {
		return fmt.Errorf("no source defined")
	}

	if in.EventData != nil && in.LastExecution != nil {
		return fmt.Errorf("cannot use multiple sources at the same time")
	}

	if in.EventData != nil {
		err := validateValueFromEventData(in.EventData)
		if err != nil {
			return fmt.Errorf("invalid event data: %v", err)
		}

		return nil
	}

	err := validateValueFromLastExecution(in.LastExecution)
	if err != nil {
		return fmt.Errorf("invalid last execution: %v", err)
	}

	return nil
}

func validateValueFromEventData(in *pb.ValueFromEventData) error {
	if in.Connection == "" {
		return fmt.Errorf("empty connection")
	}

	if in.Expression == "" {
		return fmt.Errorf("empty expression")
	}

	return nil
}

func validateValueFromLastExecution(in *pb.ValueFromLastExecution) error {
	if len(in.Results) == 0 {
		return fmt.Errorf("empty results")
	}

	for _, result := range in.Results {
		if result == pb.Execution_RESULT_UNKNOWN {
			return fmt.Errorf("invalid execution result %s", result)
		}
	}

	return nil
}

// TODO: should we use an enum here too?
func validateInputMappingWhen(in *pb.InputMapping_When) error {
	if in.TriggeredBy == nil {
		return fmt.Errorf("missing triggered by")
	}

	if in.TriggeredBy.Connection == "" {
		return fmt.Errorf("empty connection name for triggered by condition")
	}

	return nil
}

func (v *Validator) SerializeInputMappings() []models.InputMapping {
	mappings := []models.InputMapping{}
	for _, mapping := range v.InputMappings {
		m := models.InputMapping{
			Values: []models.ValueDefinition{},
		}

		for _, valueDefinition := range mapping.Values {
			def := models.ValueDefinition{
				Name:      valueDefinition.Name,
				ValueFrom: serializeValueFrom(valueDefinition.ValueFrom),
			}

			if valueDefinition.Value != "" {
				def.Value = &valueDefinition.Value
			}

			m.Values = append(m.Values, def)
		}

		if mapping.When != nil {
			m.When = &models.InputMappingWhen{
				TriggeredBy: &models.WhenTriggeredBy{Connection: mapping.When.TriggeredBy.Connection},
			}
		}

		mappings = append(mappings, m)
	}

	return mappings
}

func serializeValueFrom(in *pb.ValueFrom) *models.ValueDefinitionFrom {
	if in == nil {
		return nil
	}

	if in.EventData != nil {
		return &models.ValueDefinitionFrom{
			EventData: &models.ValueDefinitionFromEventData{
				Connection: in.EventData.Connection,
				Expression: in.EventData.Expression,
			},
		}
	}

	if in.LastExecution != nil {
		results := []string{}
		for _, result := range in.LastExecution.Results {
			switch result {
			case pb.Execution_RESULT_PASSED:
				results = append(results, models.StageExecutionResultPassed)
			case pb.Execution_RESULT_FAILED:
				results = append(results, models.StageExecutionResultFailed)
			}
		}

		return &models.ValueDefinitionFrom{
			LastExecution: &models.ValueDefinitionFromLastExecution{
				Results: results,
			},
		}
	}

	return nil
}
