package digitalocean

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var errInvalidDropletID = errors.New("must be a number")

func parseDropletID(raw string) (int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, errInvalidDropletID
	}

	maxInt := int64(^uint(0) >> 1)

	// First try a strict integer parse.
	if parsed, err := strconv.ParseInt(s, 10, 64); err == nil {
		if parsed <= 0 {
			return 0, fmt.Errorf("must be a positive integer")
		}
		if parsed > maxInt {
			return 0, fmt.Errorf("is too large")
		}
		return int(parsed), nil
	}

	// Expressions can stringify float64 values using %v, which yields scientific notation.
	// Example: 557784760 -> "5.5778476e+08".
	if !strings.ContainsAny(s, ".eE") {
		return 0, errInvalidDropletID
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, errInvalidDropletID
	}

	if f <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}

	if math.Trunc(f) != f {
		return 0, fmt.Errorf("must be an integer")
	}

	// Do not convert before range-checking, otherwise we risk undefined behavior / overflow.
	if strconv.IntSize == 64 {
		// float64(maxInt64) rounds up to exactly 2^63, so treat 2^63 as the exclusive upper bound.
		if f >= math.Exp2(63) {
			return 0, fmt.Errorf("is too large")
		}
	} else {
		if f > float64(maxInt) {
			return 0, fmt.Errorf("is too large")
		}
	}

	parsed := int64(f)
	if parsed <= 0 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	if parsed > maxInt {
		return 0, fmt.Errorf("is too large")
	}

	return int(parsed), nil
}
