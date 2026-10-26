package llm

import (
	"crypto/sha256"
	"encoding/hex"
)

// CredentialIdentity rejects account replacement when a provider does not
// expose a stable account identifier. Plaintext keys never leave the hash.
func CredentialIdentity(apiKey, managementKey, baseURL string) string {
	digest := sha256.Sum256([]byte(apiKey + "\x00" + managementKey + "\x00" + baseURL))
	return hex.EncodeToString(digest[:])
}
