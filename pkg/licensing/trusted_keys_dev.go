//go:build licensedev

package licensing

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// DevKeySetPathEnv points to a JWKS file with local issuer keys. Only binaries
// built with the licensedev tag read it. Release images never use that tag.
const DevKeySetPathEnv = "SUPERPLANE_LICENSE_DEV_JWKS_PATH"

func TrustedKeySet() (*KeySet, error) {
	production, err := ProductionKeySet()
	if err != nil {
		return nil, err
	}

	path := os.Getenv(DevKeySetPathEnv)
	if path == "" {
		return production, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read development license keys: %w", err)
	}

	development, err := ParseKeySet(data)
	if err != nil {
		return nil, fmt.Errorf("parse development license keys: %w", err)
	}

	log.Warnf("Licensing: trusting development license keys %v. Do not use this build in production.", development.KeyIDs())
	return production.Merge(development)
}
