package core

import (
	"fmt"
	"strings"
)

// IsNewerVersion returns true if latest is a newer semver than current.
// Both may optionally have a "v" prefix.
func IsNewerVersion(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	if current == latest {
		return false
	}

	var cMajor, cMinor, cPatch int
	var lMajor, lMinor, lPatch int

	fmt.Sscanf(current, "%d.%d.%d", &cMajor, &cMinor, &cPatch)
	fmt.Sscanf(latest, "%d.%d.%d", &lMajor, &lMinor, &lPatch)

	if lMajor != cMajor {
		return lMajor > cMajor
	}

	if lMinor != cMinor {
		return lMinor > cMinor
	}

	return lPatch > cPatch
}
