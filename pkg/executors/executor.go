package executors

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

var expressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

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
	if !expressionRegex.MatchString(expression) {
		return expression, nil
	}

	var err error

	result := expressionRegex.ReplaceAllStringFunc(expression, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		value, e := _resolveExpression(matches[1], inputs, secrets)
		if e != nil {
			err = e
			return ""
		}

		return fmt.Sprintf("%v", value)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func _resolveExpression(expression string, inputs map[string]any, secrets map[string]string) (any, error) {
	expression = strings.TrimSpace(expression)

	// Handle direct secret access: secrets.SECRET_NAME
	if strings.HasPrefix(expression, "secrets.") {
		key := strings.TrimSpace(strings.TrimPrefix(expression, "secrets."))
		if key == "" {
			return nil, fmt.Errorf("empty secret key")
		}
		if value, exists := secrets[key]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("secret %s not found", key)
	}

	// Handle direct input access: inputs.INPUT_NAME
	if strings.HasPrefix(expression, "inputs.") {
		key := strings.TrimSpace(strings.TrimPrefix(expression, "inputs."))
		if key == "" {
			return nil, fmt.Errorf("empty input key")
		}
		if value, exists := inputs[key]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("input %s not found", key)
	}

	return nil, fmt.Errorf("invalid expression format")
}
