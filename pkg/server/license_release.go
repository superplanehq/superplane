//go:build !licensedev

package server

import "github.com/superplanehq/superplane/pkg/licensing"

func developmentLicense(keys *licensing.KeySet, source licensing.Source) (*licensing.KeySet, licensing.Source) {
	return keys, source
}
