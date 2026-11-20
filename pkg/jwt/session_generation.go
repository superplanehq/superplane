package jwt

import (
	"fmt"
	"strconv"

	gojwt "github.com/golang-jwt/jwt/v4"
)

const SessionGenerationClaim = "session_generation"

// SessionGenerationFromClaims distinguishes legacy tokens from invalid claims.
func SessionGenerationFromClaims(claims gojwt.MapClaims) (*int64, error) {
	value, present := claims[SessionGenerationClaim]
	if !present {
		return nil, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid session generation")
	}
	generation, err := strconv.ParseInt(text, 10, 64)
	if err != nil || generation < 0 {
		return nil, fmt.Errorf("invalid session generation")
	}
	return &generation, nil
}
