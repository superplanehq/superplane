package contexts

import (
	"context"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

// ResolveRuntimeConfig walks config and re-evaluates any string value that looks
// like an expression (contains {{ and }}), using the given builder's env plus
// a secrets() function. Returns a new config map; the original is not modified.
// Resolved config must not be persisted (secret values only in memory).
func ResolveRuntimeConfig(
	config map[string]any,
	builder *NodeConfigurationBuilder,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	orgID uuid.UUID,
) (map[string]any, error) {
	resolveString := func(s string) (any, error) {
		return resolveStringAtRuntime(s, builder, tx, encryptor, orgID)
	}
	resolved, err := resolveRuntimeValue(config, resolveString)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, nil
	}
	return resolved.(map[string]any), nil
}

func resolveRuntimeValue(value any, resolveString func(string) (any, error)) (any, error) {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "{{") && strings.Contains(v, "}}") {
			return resolveString(v)
		}
		return v, nil
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			resolved, err := resolveRuntimeValue(val, resolveString)
			if err != nil {
				return nil, err
			}
			result[key] = resolved
		}
		return result, nil
	case map[string]string:
		result := make(map[string]any, len(v))
		for key, val := range v {
			resolved, err := resolveRuntimeValue(val, resolveString)
			if err != nil {
				return nil, err
			}
			result[key] = resolved
		}
		return result, nil
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			resolved, err := resolveRuntimeValue(item, resolveString)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil
	default:
		return value, nil
	}
}

func resolveStringAtRuntime(
	s string,
	builder *NodeConfigurationBuilder,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	orgID uuid.UUID,
) (any, error) {
	if !expressionRegex.MatchString(s) {
		return s, nil
	}

	var err error
	result := expressionRegex.ReplaceAllStringFunc(s, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		inner := strings.TrimSpace(matches[1])
		value, e := runExpressionAtRuntime(inner, builder, tx, encryptor, orgID)
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

// runExpressionAtRuntime evaluates a single expression with the same env and
// options as build-time (builder.buildExprOptions) plus the secrets() function
// so deferred secret expressions can be resolved. Secret values must never be persisted.
func runExpressionAtRuntime(
	expression string,
	builder *NodeConfigurationBuilder,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	orgID uuid.UUID,
) (any, error) {
	env, err := builder.BuildExpressionEnv(expression)
	if err != nil {
		return nil, err
	}

	secretsFunc := func(name string) (map[string]string, error) {
		provider, err := secrets.NewProvider(tx, encryptor, name, models.DomainTypeOrganization, orgID)
		if err != nil {
			return nil, fmt.Errorf("secret not found: %s", name)
		}
		return provider.Load(context.Background())
	}

	exprOptions := append(
		builder.buildExprOptions(env),
		expr.Function("secrets", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("secrets() takes exactly one argument (secret name)")
			}
			name, ok := params[0].(string)
			if !ok {
				return nil, fmt.Errorf("secrets() argument must be a string")
			}
			return secretsFunc(name)
		}),
	)

	vm, err := expr.Compile(expression, exprOptions...)
	if err != nil {
		return nil, err
	}
	output, err := expr.Run(vm, env)
	if err != nil {
		return nil, fmt.Errorf("expression evaluation failed: %w", err)
	}
	return output, nil
}
