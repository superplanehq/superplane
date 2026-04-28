package expressionvalidation

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
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
func ValidateCanvasExpressions(reg *registry.Registry, nodes []*componentpb.Node) map[string][]ExpressionError {
	results := map[string][]ExpressionError{}
	if len(nodes) == 0 {
		return results
	}

	knownNodeNames := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		if n == nil || n.Name == "" {
			continue
		}
		knownNodeNames[n.Name] = struct{}{}
	}

	for _, node := range nodes {
		if node == nil || node.Configuration == nil {
			continue
		}

		fields := schemaForNode(reg, node)
		errs := validateNodeExpressions(node.Id, node.Name, node.Configuration.AsMap(), fields, knownNodeNames)
		if len(errs) > 0 {
			results[node.Id] = errs
		}
	}

	return results
}

func schemaForNode(reg *registry.Registry, node *componentpb.Node) []configuration.Field {
	if reg == nil || node.Component == "" {
		return nil
	}
	configurable, err := reg.FindConfigurableComponent(node.Component)
	if err != nil {
		return nil
	}
	return configurable.Configuration()
}

func validateNodeExpressions(nodeID, nodeName string, config map[string]any, fields []configuration.Field, knownNodeNames map[string]struct{}) []ExpressionError {
	var errs []ExpressionError
	walkConfiguration(config, fields, func(fieldPath, fieldType, value string) {
		for _, e := range validateString(fieldPath, fieldType, value, knownNodeNames) {
			e.NodeID = nodeID
			e.NodeName = nodeName
			errs = append(errs, e)
		}
	})
	return errs
}

func validateString(fieldPath, fieldType, value string, knownNodeNames map[string]struct{}) []ExpressionError {
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
	for _, match := range matches {
		body := match[2 : len(match)-2]
		if err := ValidateExpression(body, knownNodeNames); err != nil {
			errs = append(errs, ExpressionError{
				FieldPath:  fieldPath,
				Expression: match,
				Err:        err,
			})
		}
	}
	return errs
}
