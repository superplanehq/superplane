package manualrun

import "fmt"

// WalkControl tells WalkPayload whether to keep visiting leaves.
type WalkControl int

const (
	// WalkContinue keeps visiting remaining leaves.
	WalkContinue WalkControl = iota
	// WalkStop ends the walk without visiting further leaves.
	WalkStop
)

// WalkPayload visits every leaf in v until visit returns WalkStop.
// path is the dot/bracket path from the payload root.
func WalkPayload(v any, path string, visit func(path string, value any) WalkControl) WalkControl {
	switch val := v.(type) {
	case map[string]any:
		for key, child := range val {
			if WalkPayload(child, joinPath(path, key), visit) == WalkStop {
				return WalkStop
			}
		}
	case map[string]string:
		for key, child := range val {
			if WalkPayload(child, joinPath(path, key), visit) == WalkStop {
				return WalkStop
			}
		}
	case []any:
		for i, child := range val {
			if WalkPayload(child, joinPath(path, fmt.Sprintf("[%d]", i)), visit) == WalkStop {
				return WalkStop
			}
		}
	default:
		return visit(path, v)
	}
	return WalkContinue
}

func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	if len(segment) > 0 && segment[0] == '[' {
		return prefix + segment
	}
	return prefix + "." + segment
}
