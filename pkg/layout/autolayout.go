package layout

import (
	"fmt"
	"strings"
)

const (
	AlgorithmHorizontal     = "ALGORITHM_HORIZONTAL"
	ScopeFullCanvas         = "SCOPE_FULL_CANVAS"
	ScopeConnectedComponent = "SCOPE_CONNECTED_COMPONENT"
)

// AutoLayout configures optional automatic node positioning.
type AutoLayout struct {
	Algorithm string   `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	Scope     string   `json:"scope,omitempty" yaml:"scope,omitempty"`
	NodeIDs   []string `json:"nodeIds,omitempty" yaml:"nodeIds,omitempty"`
}

func (a *AutoLayout) resolvedAlgorithm() (string, error) {
	if a == nil {
		return "", fmt.Errorf("layout algorithm is required")
	}

	algorithm := strings.ToUpper(strings.TrimSpace(a.Algorithm))
	switch algorithm {
	case "", AlgorithmHorizontal, "HORIZONTAL":
		return AlgorithmHorizontal, nil
	case "ALGORITHM_UNSPECIFIED", "UNSPECIFIED":
		return "", fmt.Errorf("layout algorithm is required")
	default:
		return "", fmt.Errorf("unsupported layout algorithm: %s", a.Algorithm)
	}
}

func ParseAutoLayout(value string, scopeValue string, nodeIDs []string) (*AutoLayout, error) {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	switch normalizedValue {
	case "disable", "disabled", "none", "off":
		if strings.TrimSpace(scopeValue) != "" || len(nodeIDs) > 0 {
			return nil, fmt.Errorf("--auto-layout-scope and --auto-layout-node cannot be used when --auto-layout disables layout")
		}
		return nil, nil
	}

	autoLayout := &AutoLayout{}

	switch normalizedValue {
	case "", "horizontal":
		autoLayout.Algorithm = AlgorithmHorizontal
	default:
		return nil, fmt.Errorf("unsupported auto layout %q (supported: horizontal, disable)", value)
	}

	normalizedNodeIDs := make([]string, 0, len(nodeIDs))
	seen := make(map[string]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		trimmed := strings.TrimSpace(nodeID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalizedNodeIDs = append(normalizedNodeIDs, trimmed)
	}
	if len(normalizedNodeIDs) > 0 {
		autoLayout.NodeIDs = normalizedNodeIDs
	}

	if strings.TrimSpace(scopeValue) == "" {
		return autoLayout, nil
	}

	switch strings.ToLower(strings.TrimSpace(scopeValue)) {
	case "full-canvas", "full_canvas", "full":
		autoLayout.Scope = ScopeFullCanvas
	case "connected-component", "connected_component", "connected":
		autoLayout.Scope = ScopeConnectedComponent
	default:
		return nil, fmt.Errorf("unsupported auto layout scope %q (supported: full-canvas, connected-component)", scopeValue)
	}

	return autoLayout, nil
}

func DefaultAutoLayout() AutoLayout {
	return AutoLayout{
		Algorithm: AlgorithmHorizontal,
		Scope:     ScopeFullCanvas,
	}
}
