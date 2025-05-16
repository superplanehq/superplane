package resolver

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/expr-lang/expr"
	"github.com/superplanehq/superplane/pkg/models"
)

const expressionRegex = `^\$\{\{(.*)\}\}$`

type Resolver struct {
	execution models.StageExecution
	template  models.RunTemplate
	regex     *regexp.Regexp
}

func NewResolver(execution models.StageExecution, template models.RunTemplate) *Resolver {
	regex := regexp.MustCompile(expressionRegex)
	return &Resolver{
		execution: execution,
		template:  template,
		regex:     regex,
	}
}

func (r *Resolver) Resolve() (*models.RunTemplate, error) {
	switch r.template.Type {
	case models.RunTemplateTypeSemaphore:
		return r.resolveSemaphoreTemplate()
	default:
		return nil, fmt.Errorf("resolution of run template type %s not supported", r.template.Type)
	}
}

func (r *Resolver) resolveSemaphoreTemplate() (*models.RunTemplate, error) {
	t := r.template.Semaphore
	projectID, err := r.resolveExpression(t.ProjectID)
	if err != nil {
		return nil, err
	}

	branch, err := r.resolveExpression(t.Branch)
	if err != nil {
		return nil, err
	}

	pipelineFile, err := r.resolveExpression(t.PipelineFile)
	if err != nil {
		return nil, err
	}

	taskID, err := r.resolveExpression(t.TaskID)
	if err != nil {
		return nil, err
	}

	parameters := make(map[string]string, len(t.Parameters))
	for k, v := range t.Parameters {
		value, err := r.resolveExpression(v)
		if err != nil {
			return nil, err
		}

		parameters[k] = value
	}

	return &models.RunTemplate{
		Type: models.RunTemplateTypeSemaphore,
		Semaphore: &models.SemaphoreRunTemplate{
			OrganizationURL: t.OrganizationURL,
			APIToken:        t.APIToken,
			ProjectID:       projectID,
			Branch:          branch,
			PipelineFile:    pipelineFile,
			TaskID:          taskID,
			Parameters:      parameters,
		},
	}, nil
}

func (r *Resolver) resolveExpression(expression string) (string, error) {
	if r.regex.MatchString(expression) {
		matches := r.regex.FindStringSubmatch(expression)
		if len(matches) != 2 {
			return "", fmt.Errorf("error resolving expression")
		}

		return r._resolveExpression(matches[1])
	}

	return expression, nil
}

func (r *Resolver) _resolveExpression(expression string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	variables := map[string]any{
		"ctx":  ctx,
		"self": Self{execution: r.execution},
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

	//
	// Output of the expression must be a string.
	//
	v, ok := output.(string)
	if !ok {
		return "", fmt.Errorf("expression does not return a string")
	}

	return v, nil
}

type Self struct {
	execution models.StageExecution
}

func (s Self) Conn(name string) (map[string]any, error) {
	sourceName, err := s.execution.FindSource()
	if err != nil {
		return nil, fmt.Errorf("error finding source for execution: %v", err)
	}

	//
	// If the connection wanted is the one that triggered the execution,
	// just use the data on the event itself.
	//
	if name == sourceName {
		data, err := s.execution.GetEventData()
		if err != nil {
			return nil, fmt.Errorf("error finding event data for execution: %v", err)
		}

		return data, nil
	}

	connection, err := models.FindStageConnection(s.execution.StageID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection with name %s: %w", name, err)
	}

	//
	// TODO
	// we'll need to differentiate things here a little bit depending on the type of the connection.
	// For example, for stages, we are only interested in the last stage __completion__ event.
	// We only have that type of stage event now, but we might end up having more.
	//
	// Also, right now, we are erroring if there is no event for the connection yet,
	// but we might want to handle that differently.
	//
	data, err := models.FindLastEventBySourceID(connection.SourceID)
	if err != nil {
		return nil, fmt.Errorf("error finding last event for connection %s: %v", name, err)
	}

	return data, nil
}
