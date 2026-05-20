package canvases

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
)

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
	number, err := value.Float64()
	if err != nil {
		return raw
	}

	return number
}
