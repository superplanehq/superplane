package configuration

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	// The timezone database is embedded so IANA lookups work in container
	// images that do not ship the system tzdata package.
	_ "time/tzdata"
)

const (
	minTimezoneOffsetHours = -12
	maxTimezoneOffsetHours = 14
)

// LoadTimezone resolves a timezone field value to a *time.Location.
//
// IANA identifiers such as "America/New_York" carry daylight saving rules, so
// they resolve to the correct offset for whichever instant is being evaluated.
//
// Numeric offsets such as "-5" or "5.5" are still accepted, because
// configurations saved before identifiers were supported store them. They
// resolve to a fixed offset and therefore do not follow daylight saving.
func LoadTimezone(value string) (*time.Location, error) {
	if value == "" {
		return nil, fmt.Errorf("timezone cannot be empty")
	}

	if offsetHours, err := strconv.ParseFloat(strings.TrimPrefix(value, "+"), 64); err == nil {
		if offsetHours < minTimezoneOffsetHours || offsetHours > maxTimezoneOffsetHours {
			return nil, fmt.Errorf("timezone offset must be between -12 and +14 hours, got: %g", offsetHours)
		}

		if offsetHours != float64(int(offsetHours)) && offsetHours != float64(int(offsetHours))+0.5 {
			return nil, fmt.Errorf("timezone offset must be a whole number or half hour (e.g., 5.5), got: %g", offsetHours)
		}

		return time.FixedZone(fmt.Sprintf("GMT%+.1f", offsetHours), int(offsetHours*3600)), nil
	}

	//
	// "Local" resolves to whatever timezone the server runs in, which makes the
	// same configuration behave differently across deployments.
	//
	if value == "Local" {
		return nil, fmt.Errorf("invalid timezone %q: use an IANA identifier like 'America/New_York'", value)
	}

	location, err := time.LoadLocation(value)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: must be an IANA identifier like 'America/New_York' or a numeric offset like '-5'", value)
	}

	return location, nil
}
