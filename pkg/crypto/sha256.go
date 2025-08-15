package crypto

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

func SHA256ForMap(m map[string]string) (string, error) {
	//
	// Maps are not ordered, so we need to sort the key/value
	// pairs before hashing it. We do that by creating an array of key=value
	// pairs and sorting it.
	//
	var keyValues []string
	for k, v := range m {
		keyValues = append(keyValues, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(keyValues)

	//
	// Now, we join our list of key/value pairs and hash it.
	//
	h := sha256.New()
	_, err := h.Write([]byte(strings.Join(keyValues, ",")))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
