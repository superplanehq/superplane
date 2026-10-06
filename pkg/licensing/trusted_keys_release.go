//go:build !licensedev

package licensing

// TrustedKeyStore returns the keys that verify licenses in this binary.
// Release builds trust only key lists signed by the embedded root keys.
func TrustedKeyStore(extra *KeySet) (*KeyStore, error) {
	return productionKeyStore(extra)
}
