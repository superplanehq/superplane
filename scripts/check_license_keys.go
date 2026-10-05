package main

// Compares the bundled license verification keys with the issuer JWKS.
// Run it as an explicit release check. Normal builds and CI must not depend on
// the issuer endpoint.

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/superplanehq/superplane/pkg/licensing"
)

const maxJWKSBytes = 64 * 1024

type publicKey struct {
	KeyType   string `json:"kty"`
	Curve     string `json:"crv"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	X         string `json:"x"`
	Y         string `json:"y"`
}

type keySet struct {
	Keys []publicKey `json:"keys"`
}

func main() {
	bundledPath := flag.String("bundled", "pkg/licensing/trustedkeys/production.jwks.json", "bundled JWKS file")
	issuerURL := flag.String("issuer", "https://licensing.superplane.com/.well-known/jwks.json", "issuer JWKS URL")
	flag.Parse()

	if err := run(*bundledPath, *issuerURL); err != nil {
		fmt.Fprintln(os.Stderr, "license key check failed:", err)
		os.Exit(1)
	}
}

func run(bundledPath, issuerURL string) error {
	bundledData, err := os.ReadFile(bundledPath)
	if err != nil {
		return fmt.Errorf("read bundled keys: %w", err)
	}

	if _, err := licensing.ParseKeySet(bundledData); err != nil {
		return fmt.Errorf("bundled keys are not valid: %w", err)
	}

	issuerData, err := fetch(issuerURL)
	if err != nil {
		return err
	}

	bundled, err := decode(bundledData)
	if err != nil {
		return fmt.Errorf("decode bundled keys: %w", err)
	}

	issuer, err := decode(issuerData)
	if err != nil {
		return fmt.Errorf("decode issuer keys: %w", err)
	}

	failed := false
	for keyID, key := range bundled {
		published, ok := issuer[keyID]
		if !ok {
			fmt.Printf("FAIL %s: bundled key is not published by the issuer\n", keyID)
			failed = true
			continue
		}

		if published != key {
			fmt.Printf("FAIL %s: bundled key does not match the issuer key\n", keyID)
			failed = true
			continue
		}

		fmt.Printf("OK   %s\n", keyID)
	}

	for keyID := range issuer {
		if _, ok := bundled[keyID]; !ok {
			fmt.Printf("WARN %s: issuer key is not bundled; review it before the issuer activates it\n", keyID)
		}
	}

	if failed {
		return fmt.Errorf("bundled keys do not match %s", issuerURL)
	}

	return nil
}

func fetch(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch issuer keys: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch issuer keys: unexpected status %d", response.StatusCode)
	}

	return io.ReadAll(io.LimitReader(response.Body, maxJWKSBytes))
}

func decode(data []byte) (map[string]publicKey, error) {
	var set keySet
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, err
	}

	keys := make(map[string]publicKey, len(set.Keys))
	for _, key := range set.Keys {
		keys[key.KeyID] = key
	}

	return keys, nil
}
