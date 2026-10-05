//go:build !licensedev

package licensing

// TrustedKeySet returns the keys that verify licenses in this binary. Release
// builds trust only the bundled production keys.
func TrustedKeySet() (*KeySet, error) {
	return ProductionKeySet()
}
