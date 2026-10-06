// Package devlicense grants every Enterprise feature to the development
// server. Only binaries built with the licensedev tag import this package.
// Release binaries do not contain it.
package devlicense

import (
	"context"
	"errors"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

// EnterpriseEnv must be "true" to grant the development license. The
// development compose file sets it.
const EnterpriseEnv = "SUPERPLANE_LICENSE_DEV_ENTERPRISE"

const keyID = "development-ephemeral"

func Enabled() bool {
	return os.Getenv(EnterpriseEnv) == "true"
}

// Wrap returns a signing key that exists only in this process, to trust in
// addition to the issuer keys, and a source that supplies a license for every
// recognized feature when no license is installed. An installed license has
// priority. A license file stays the only source, so the function returns no
// key and does not change a file source.
func Wrap(source licensing.Source) (*licensing.KeySet, licensing.Source) {
	writable, ok := source.(licensing.WritableSource)
	if !ok {
		log.Infof("Licensing: %s is ignored because a license file is configured", EnterpriseEnv)
		return nil, source
	}

	issuer := licensingtest.NewIssuer(keyID)
	log.Warn("Licensing: development build grants every Enterprise feature when no license is installed. Do not use this build in production.")
	return licensingtest.KeySet(issuer), &developmentSource{
		WritableSource: writable,
		license:        issuer.License(licensing.RecognizedFeatures()...),
	}
}

type developmentSource struct {
	licensing.WritableSource
	license []byte
}

func (s *developmentSource) Read(ctx context.Context) ([]byte, error) {
	raw, err := s.WritableSource.Read(ctx)
	if errors.Is(err, licensing.ErrNotInstalled) {
		return s.license, nil
	}

	return raw, err
}
