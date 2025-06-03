package inputs

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/superplanehq/superplane/pkg/models"
)

var expressionRegex = regexp.MustCompile(`^\$\{\{(.*)\}\}$`)

// ExecutorSpecBuilder takes an executor spec from the stage,
// and a map of inputs, built by InputBuilder,
// and returns the final executor spec used for creating the execution,
// resolving the ${{ inputs.* }} expression that may be present in the executor spec.
type ExecutorSpecBuilder struct {
	spec    models.ExecutorSpec
	inputs  map[string]any
	secrets map[string]string
}

func NewExecutorSpecBuilder(spec models.ExecutorSpec, inputs map[string]any, secrets map[string]string) *ExecutorSpecBuilder {
	return &ExecutorSpecBuilder{
		spec:    spec,
		inputs:  inputs,
		secrets: secrets,
	}
}

func (r *ExecutorSpecBuilder) Build() (*models.ExecutorSpec, error) {
	switch r.spec.Type {
	case models.ExecutorSpecTypeSemaphore:
		return r.resolveSemaphoreExecutorSpec()
	default:
		return nil, fmt.Errorf("resolution of executor spec type %s not supported", r.spec.Type)
	}
}

func (r *ExecutorSpecBuilder) resolveSemaphoreExecutorSpec() (*models.ExecutorSpec, error) {
	t := r.spec.Semaphore
	token, err := r.ResolveExpression(t.APIToken)
	if err != nil {
		return nil, err
	}

	projectID, err := r.ResolveExpression(t.ProjectID)
	if err != nil {
		return nil, err
	}

	branch, err := r.ResolveExpression(t.Branch)
	if err != nil {
		return nil, err
	}

	pipelineFile, err := r.ResolveExpression(t.PipelineFile)
	if err != nil {
		return nil, err
	}

	taskID, err := r.ResolveExpression(t.TaskID)
	if err != nil {
		return nil, err
	}

	parameters := make(map[string]string, len(t.Parameters))
	for k, v := range t.Parameters {
		value, err := r.ResolveExpression(v)
		if err != nil {
			return nil, err
		}

		parameters[k] = value.(string)
	}

	return &models.ExecutorSpec{
		Type: models.ExecutorSpecTypeSemaphore,
		Semaphore: &models.SemaphoreExecutorSpec{
			OrganizationURL: t.OrganizationURL,
			APIToken:        token.(string),
			ProjectID:       projectID.(string),
			Branch:          branch.(string),
			PipelineFile:    pipelineFile.(string),
			TaskID:          taskID.(string),
			Parameters:      parameters,
		},
	}, nil
}

func (r *ExecutorSpecBuilder) ResolveExpression(expression string) (any, error) {
	if expressionRegex.MatchString(expression) {
		matches := expressionRegex.FindStringSubmatch(expression)
		if len(matches) != 2 {
			return "", fmt.Errorf("error resolving expression")
		}

		value, err := r._resolveExpression(matches[1])
		if err != nil {
			return nil, fmt.Errorf("error resolving expression: %v", err)
		}

		//
		// If no error is returned, but value is nil,
		// then user is trying to access an input that is not defined.
		//
		if value == nil {
			parts := strings.Split(strings.Trim(matches[1], " "), ".")
			return nil, fmt.Errorf("input %s not found", parts[1])
		}

		return value, nil
	}

	return expression, nil
}

func (r *ExecutorSpecBuilder) _resolveExpression(expression string) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	variables := map[string]any{
		"ctx":     ctx,
		"inputs":  r.inputs,
		"secrets": r.secrets,
	}

	program, err := expr.Compile(expression,
		expr.Env(variables),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	)

	if err != nil {
		return "", fmt.Errorf("error compiling expression: %v", err)
	}

	output, err := expr.Run(program, variables)
	if err != nil {
		return "", fmt.Errorf("error running expression: %v", err)
	}

	return output, nil
}
