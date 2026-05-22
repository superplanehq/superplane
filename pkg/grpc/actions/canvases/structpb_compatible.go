package canvases

import (
	"encoding/json"
	"math"
	"math/big"

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
	if err != nil || math.IsInf(number, 0) {
		return raw
	}

	if jsonNumberLosesFloat64Precision(raw, number) {
		return raw
	}

	return number
}

func jsonNumberLosesFloat64Precision(raw string, number float64) bool {
	if !isJSONIntegerToken(raw) {
		return false
	}

	original, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return true
	}

	converted := new(big.Int)
	new(big.Float).SetFloat64(number).Int(converted)
	return original.Cmp(converted) != 0
}

func isJSONIntegerToken(raw string) bool {
	if raw == "" || raw == "-" {
		return false
	}

	start := 0
	if raw[0] == '-' {
		if len(raw) == 1 {
			return false
		}
		start = 1
	}

	for i := start; i < len(raw); i++ {
		if raw[i] < '0' || raw[i] > '9' {
			return false
		}
	}

	return true
}
