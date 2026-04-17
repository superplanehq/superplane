package core

import (
	"fmt"
	"strings"
)

// SplitUserIdentifier picks between a user id and a user email based on
// either a positional argument or a --email flag. A positional containing
// "@" is treated as an email, so CLI commands can accept either form in the
// positional slot.
//
// Returns (userID, userEmail, error). Exactly one of userID/userEmail is
// non-empty on success when an identifier was provided. Both empty means no
// identifier was given. An error is returned when both a positional and an
// explicit --email flag were supplied (ambiguous).
func SplitUserIdentifier(positional string, emailFlag string) (string, string, error) {
	positional = strings.TrimSpace(positional)
	emailFlag = strings.TrimSpace(emailFlag)

	if positional != "" && emailFlag != "" {
		return "", "", fmt.Errorf("pass either a positional user id or --email, not both")
	}

	if positional != "" {
		if strings.Contains(positional, "@") {
			return "", positional, nil
		}
		return positional, "", nil
	}
	if emailFlag != "" {
		return "", emailFlag, nil
	}
	return "", "", nil
}
