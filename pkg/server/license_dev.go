//go:build licensedev

package server

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/devlicense"
)

func developmentLicense(keys *licensing.KeySet, source licensing.Source) (*licensing.KeySet, licensing.Source) {
	if !devlicense.Enabled() {
		return keys, source
	}

	keys, source, err := devlicense.Wrap(keys, source)
	if err != nil {
		panic(fmt.Sprintf("failed to set up the development license: %v", err))
	}

	return keys, source
}
