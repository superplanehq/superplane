//go:build !licensedev

package server

import "github.com/superplanehq/superplane/pkg/licensing"

func developmentLicense(source licensing.Source) (*licensing.KeySet, licensing.Source) {
	return nil, source
}
