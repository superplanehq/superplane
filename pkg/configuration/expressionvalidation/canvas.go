package expressionvalidation

import (
	"fmt"
	"regexp"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

// ExpressionError describes a single static validation failure for one
// expression inside a node's configuration.
type ExpressionError struct {
	NodeID     string
	NodeName   string
	FieldPath  string
	Expression string
	Err        error
}

func (e *ExpressionError) Error() string {
	return fmt.Sprintf("field %q: expression %q: %s", e.FieldPath, e.Expression, e.Err.Error())
}

// ValidateCanvasExpressions runs static expression validation across every
// node's configuration. Errors are returned per node ID; an empty result means
// nothing failed.
func ValidateCanvasExpressions(reg *registry.Registry, nodes []models.Node) map[string][]ExpressionError {
	results := map[string][]ExpressionError{}
	if len(nodes) == 0 {
		return results
	}

	knownNodeNames := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		if n.Name == "" {
			continue
		}
		knownNodeNames[n.Name] = struct{}{}
	}

	for _, node := range nodes {
		if node.Configuration == nil {
			continue
		}

		fields := schemaForNode(reg, node)
		errs := validateNodeExpressions(node.ID, node.Name, node.Configuration, fields, knownNodeNames)
		if len(errs) > 0 {
			results[node.ID] = errs
		}
	}

	return results
}

func schemaForNode(reg *registry.Registry, node models.Node) []configuration.Field {
	componentName := node.ComponentName()
	if componentName == "" {
		return nil
	}

	configurable, err := reg.FindConfigurableComponent(componentName)
	if err != nil {
		return nil
	}
	return configurable.Configuration()
}

func validateNodeExpressions(nodeID, nodeName string, config map[string]any, fields []configuration.Field, knownNodeNames map[string]struct{}) []ExpressionError {
	var errs []ExpressionError
	walkConfiguration(config, fields, func(fieldPath string, field configuration.Field, value string) {
		for _, e := range validateString(fieldPath, field, value, knownNodeNames) {
			e.NodeID = nodeID
			e.NodeName = nodeName
			errs = append(errs, e)
		}
	})
	return errs
}

func validateString(fieldPath string, field configuration.Field, value string, knownNodeNames map[string]struct{}) []ExpressionError {
	if field.Type == configuration.FieldTypeString &&
		field.TypeOptions != nil &&
		field.TypeOptions.String != nil &&
		field.TypeOptions.String.AllowExpressions != nil &&
		!*field.TypeOptions.String.AllowExpressions {
		if configuration.ExpressionPlaceholderRegex.MatchString(value) {
			return []ExpressionError{{
				FieldPath:  fieldPath,
				Expression: value,
				Err:        fmt.Errorf("expressions are not supported for this field"),
			}}
		}
		return nil
	}

	// Text fields can disable expression resolution without forbidding literal
	// {{ }} content (e.g. runner scripts). Skip SuperPlane expression checks.
	if field.Type == configuration.FieldTypeText &&
		field.TypeOptions != nil &&
		field.TypeOptions.Text != nil &&
		field.TypeOptions.Text.AllowExpressions != nil &&
		!*field.TypeOptions.Text.AllowExpressions {
		return nil
	}

	fieldType := field.Type
	matches := configuration.ExpressionPlaceholderRegex.FindAllString(value, -1)

	// FieldTypeExpression values without {{ }} framing flow through
	// NodeConfigurationBuilder.Build() as literal strings — bare identifiers
	// like "default" or "ok" never reach expr.Compile at runtime — so we skip
	// the strict identifier check here. Syntax errors and unknown node
	// references are still caught.
	if len(matches) == 0 {
		if fieldType == configuration.FieldTypeExpression {
			if err := ValidateBareExpression(value, knownNodeNames); err != nil {
				return []ExpressionError{{FieldPath: fieldPath, Expression: value, Err: err}}
			}
		}
		return nil
	}

	var errs []ExpressionError
	extraEnv := expressionValidationExtraEnv(fieldPath)
	for _, match := range matches {
		body := match[2 : len(match)-2]
		if err := ValidateExpressionWithExtraEnv(body, knownNodeNames, extraEnv); err != nil {
			errs = append(errs, ExpressionError{
				FieldPath:  fieldPath,
				Expression: match,
				Err:        err,
			})
		}
	}
	return errs
}

var startTemplatePayloadFieldPattern = regexp.MustCompile(`^templates\[\d+\]\.payload(\.|$)`)

func expressionValidationExtraEnv(fieldPath string) map[string]any {
	if startTemplatePayloadFieldPattern.MatchString(fieldPath) {
		return map[string]any{"parameters": map[string]any{}}
	}
	return nil
}
