package executors

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

var expressionRegex = regexp.MustCompile(`^\$\{\{(.*)\}\}$`)

type Executor interface {
	Name() string
	BuildSpec(models.ExecutorSpec, map[string]any, map[string]string) (*models.ExecutorSpec, error)
	Execute(models.ExecutorSpec) (Response, error)
	Check(models.ExecutorSpec, string) (Response, error)
}

type Response interface {
	Finished() bool
	Successful() bool
	Id() string
}

func NewExecutor(specType string, execution models.StageExecution, jwtSigner *jwt.Signer) (Executor, error) {
	switch specType {
	case models.ExecutorSpecTypeSemaphore:
		return NewSemaphoreExecutor(execution, jwtSigner)
	case models.ExecutorSpecTypeHTTP:
		return NewHTTPExecutor(execution, jwtSigner)
	default:
		return nil, fmt.Errorf("executor type %s not supported", specType)
	}
}

func resolveExpression(expression string, inputs map[string]any, secrets map[string]string) (any, error) {
	if expressionRegex.MatchString(expression) {
		matches := expressionRegex.FindStringSubmatch(expression)
		if len(matches) != 2 {
			return "", fmt.Errorf("error resolving expression")
		}

		value, err := _resolveExpression(matches[1], inputs, secrets)
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

func _resolveExpression(expression string, inputs map[string]any, secrets map[string]string) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	variables := map[string]any{
		"ctx":     ctx,
		"inputs":  inputs,
		"secrets": secrets,
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
