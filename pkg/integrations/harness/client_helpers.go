package harness

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func readAnyPath(input map[string]any, path ...string) any {
	current := any(input)
	for _, key := range path {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}

		next, ok := currentMap[key]
		if !ok {
			return nil
		}

		current = next
	}

	return current
}

func readStringPath(input map[string]any, path ...string) string {
	value := readAnyPath(input, path...)
	return readString(value)
}

func readString(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		if math.Mod(typed, 1) == 0 {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case float32:
		f := float64(typed)
		if math.Mod(f, 1) == 0 {
			return strconv.FormatInt(int64(f), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(f, 'f', -1, 32))
	case int:
		return strconv.Itoa(typed)
	case int8:
		return strconv.FormatInt(int64(typed), 10)
	case int16:
		return strconv.FormatInt(int64(typed), 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	}

	if text, ok := value.(fmt.Stringer); ok {
		return strings.TrimSpace(text.String())
	}

	return ""
}

func arrayOfMaps(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return []map[string]any{}
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		mapItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mapItem)
	}

	return result
}

func firstMapFromArray(value any) map[string]any {
	items := arrayOfMaps(value)
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
