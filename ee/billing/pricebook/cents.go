package pricebook

import (
	"math"
	"strconv"
	"strings"
)

const centsPerDollar = 100

// CentsPerMillionFromUSDPerToken converts a USD-per-token catalog price
// into cents per million tokens. An empty string is 0.
func CentsPerMillionFromUSDPerToken(usdPerToken string) (int64, bool) {
	trimmed := strings.TrimSpace(usdPerToken)
	if trimmed == "" {
		return 0, true
	}

	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}

	cents := math.Round(value * float64(tokensPerMillion) * float64(centsPerDollar))
	if cents > math.MaxInt64 {
		return 0, false
	}
	return int64(cents), true
}
