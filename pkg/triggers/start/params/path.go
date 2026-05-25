package params

import (
	"fmt"
	"strconv"
)

// setValueAtPath assigns value to the leaf at path
// (dot and bracket notation, e.g. body.name or items[0].id).
// The path must already exist in root; intermediate objects are not created.
func setValueAtPath(root map[string]any, path string, value any) error {
	segments, err := pathSegments(path)
	if err != nil {
		return err
	}

	cur := any(root)
	for i := 0; i < len(segments)-1; i++ {
		cur, err = stepInto(cur, segments[i])
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return setLeaf(cur, segments[len(segments)-1], value)
}

// getValueAtPath returns the value at path.
// The second result is false when the path does not exist.
func getValueAtPath(root map[string]any, path string) (any, bool, error) {
	segments, err := pathSegments(path)
	if err != nil {
		return nil, false, err
	}

	cur := any(root)
	for _, seg := range segments {
		cur, err = stepInto(cur, seg)
		if err != nil {
			return nil, false, nil
		}
	}
	return cur, true, nil
}

func pathSegments(path string) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	var segments []string
	var buffer string
	i := 0
	for i < len(path) {
		ch := path[i]
		switch ch {
		case '.':
			if buffer != "" {
				segments = append(segments, buffer)
				buffer = ""
			}
			i++
		case '[':
			if buffer != "" {
				segments = append(segments, buffer)
				buffer = ""
			}
			end := -1
			for j := i + 1; j < len(path); j++ {
				if path[j] == ']' {
					end = j
					break
				}
			}
			if end < 0 {
				return nil, fmt.Errorf("unterminated '[' in path %q", path)
			}
			segment := path[i+1 : end]
			if len(segment) >= 2 && ((segment[0] == '\'' && segment[len(segment)-1] == '\'') ||
				(segment[0] == '"' && segment[len(segment)-1] == '"')) {
				segment = segment[1 : len(segment)-1]
			}
			segments = append(segments, segment)
			i = end + 1
		default:
			buffer += string(ch)
			i++
		}
	}
	if buffer != "" {
		segments = append(segments, buffer)
	}
	return segments, nil
}

func stepInto(cur any, seg string) (any, error) {
	switch node := cur.(type) {
	case map[string]any:
		child, ok := node[seg]
		if !ok {
			return nil, fmt.Errorf("missing key %q", seg)
		}
		return child, nil
	case []any:
		idx, err := strconv.Atoi(seg)
		if err != nil {
			return nil, fmt.Errorf("expected numeric index, got %q", seg)
		}
		if idx < 0 || idx >= len(node) {
			return nil, fmt.Errorf("index %d out of range", idx)
		}
		return node[idx], nil
	default:
		return nil, fmt.Errorf("cannot traverse into %T", cur)
	}
}

func setLeaf(cur any, seg string, value any) error {
	switch node := cur.(type) {
	case map[string]any:
		if _, ok := node[seg]; !ok {
			return fmt.Errorf("missing key %q", seg)
		}
		node[seg] = value
		return nil
	case []any:
		idx, err := strconv.Atoi(seg)
		if err != nil {
			return fmt.Errorf("expected numeric index, got %q", seg)
		}
		if idx < 0 || idx >= len(node) {
			return fmt.Errorf("index %d out of range", idx)
		}
		node[idx] = value
		return nil
	default:
		return fmt.Errorf("cannot set leaf on %T", cur)
	}
}
