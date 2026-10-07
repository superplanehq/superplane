//go:build licensedev

package server

import (
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/devlicense"
)

func developmentLicense(source licensing.Source) (*licensing.KeySet, licensing.Source) {
	if !devlicense.Enabled() {
		return nil, source
	}

	return devlicense.Wrap(source)
}
