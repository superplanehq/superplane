package contexts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeJSONNumberUsesFloat64ForIntegers(t *testing.T) {
	result := normalizeJSONNumber(json.Number("10"))
	assert.IsType(t, float64(0), result)
	assert.Equal(t, float64(10), result)
}

func TestNormalizeJSONNumberUsesFloat64ForFractions(t *testing.T) {
	result := normalizeJSONNumber(json.Number("10.5"))
	assert.IsType(t, float64(0), result)
	assert.Equal(t, float64(10.5), result)
}

func TestNormalizeJSONNumberKeepsInvalidToken(t *testing.T) {
	token := json.Number("not-a-number")
	result := normalizeJSONNumber(token)
	assert.Equal(t, token, result)
}

func TestNormalizeJSONNumberLargeIntegerTokenUsesFloat64(t *testing.T) {
	result := normalizeJSONNumber(json.Number("9007199254740993"))
	assert.IsType(t, float64(0), result)
}
