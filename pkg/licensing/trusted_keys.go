package licensing

import _ "embed"

// rootJWKS holds the root public keys that sign license key lists. Change it
// only through a reviewed root-rotation pull request, and keep the previous
// root until no installation needs it. Never add a development key.
//
//go:embed trustedkeys/root.jwks.json
var rootJWKS []byte

// bootstrapKeyList is the newest key list at release time. Installations use
// it until they download a newer one. Update it with make license.keys.update.
//
//go:embed trustedkeys/license-keys.jws
var bootstrapKeyList []byte

func ProductionRootKeySet() (*KeySet, error) {
	return ParseOptionalKeySet(rootJWKS)
}

func productionKeyStore(extra *KeySet) (*KeyStore, error) {
	roots, err := ProductionRootKeySet()
	if err != nil {
		return nil, err
	}

	return NewKeyStore(roots, bootstrapKeyList, extra)
}
