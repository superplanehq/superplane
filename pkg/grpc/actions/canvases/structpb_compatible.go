package canvases

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

const maxExactJavaScriptInteger = 1<<53 - 1

func newStructpbStruct(value map[string]any) (*structpb.Struct, error) {
	converted := toStructpbCompatible(value)
	return structpb.NewStruct(converted.(map[string]any))
}

func newStructpbValue(value any) (*structpb.Value, error) {
	return structpb.NewValue(toStructpbCompatible(value))
}

func toStructpbCompatible(value any) any {
	switch typed := value.(type) {
	case json.Number:
		return jsonNumberForStructpb(typed)
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = toStructpbCompatible(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = toStructpbCompatible(item)
		}
		return out
	default:
		return value
	}
}

func jsonNumberForStructpb(value json.Number) any {
	raw := value.String()
	if shouldSerializeJSONNumberAsString(raw) {
		return raw
	}

	number, err := value.Float64()
	if err != nil {
		return raw
	}

	return number
}

func shouldSerializeJSONNumberAsString(raw string) bool {
	if raw == "" {
		return true
	}

	if strings.ContainsAny(raw, ".eE") {
		number, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return true
		}

		abs := math.Abs(number)
		return abs != 0 && (abs < 1e-6 || abs >= 1e21)
	}

	integer, err := strconv.ParseInt(raw, 10, 64)
	if err == nil {
		return integer > maxExactJavaScriptInteger || integer < -maxExactJavaScriptInteger
	}

	unsigned, err := strconv.ParseUint(raw, 10, 64)
	if err == nil {
		return unsigned > maxExactJavaScriptInteger
	}

	return true
}
