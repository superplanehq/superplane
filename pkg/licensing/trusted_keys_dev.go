//go:build licensedev

package licensing

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// DevRootKeySetPathEnv points to a JWKS file with the root key of a local
// issuer. Only binaries built with the licensedev tag read it. Release images
// never use that tag.
const DevRootKeySetPathEnv = "SUPERPLANE_LICENSE_DEV_ROOT_JWKS_PATH"

// TrustedKeyStore trusts a local issuer instead of the production issuer when
// DevRootKeySetPathEnv is set.
func TrustedKeyStore(extra *KeySet) (*KeyStore, error) {
	path := os.Getenv(DevRootKeySetPathEnv)
	if path == "" {
		return productionKeyStore(extra)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read development root keys: %w", err)
	}

	roots, err := ParseKeySet(data)
	if err != nil {
		return nil, fmt.Errorf("parse development root keys: %w", err)
	}

	log.Warnf("Licensing: trusting development root keys %v. Do not use this build in production.", roots.KeyIDs())
	return NewKeyStore(roots, nil, extra)
}
