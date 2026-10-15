package contexts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeJSONNumberKeepsInvalidToken(t *testing.T) {
	token := json.Number("not-a-number")
	result := normalizeJSONNumber(token)
	assert.Equal(t, token, result)
}

func TestNormalizeJSONNumberLargeIntegerTokenUsesFloat64(t *testing.T) {
	result := normalizeJSONNumber(json.Number("9007199254740993"))
	assert.IsType(t, float64(0), result)
}
