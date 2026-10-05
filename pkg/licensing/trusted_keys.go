package licensing

import _ "embed"

// productionJWKS is the reviewed copy of the issuer's public signing keys.
// Change it only through a reviewed key-rotation pull request. Never add a
// development key to this file.
//
//go:embed trustedkeys/production.jwks.json
var productionJWKS []byte

func ProductionKeySet() (*KeySet, error) {
	return ParseKeySet(productionJWKS)
}
