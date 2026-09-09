package pricesync

import (
	"math"
	"strconv"
	"strings"
)

const usdPerTokenToCentsPerMillion = 100_000_000

// USDPerTokenToCentsPerMillion converts vendor USD-per-token prices into
// SuperPlane cents per million tokens.
func USDPerTokenToCentsPerMillion(usdPerToken float64) int64 {
	if usdPerToken <= 0 || math.IsNaN(usdPerToken) || math.IsInf(usdPerToken, 0) {
		return 0
	}
	return int64(math.Round(usdPerToken * usdPerTokenToCentsPerMillion))
}

func parseUSDPerToken(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return USDPerTokenToCentsPerMillion(parsed)
}
