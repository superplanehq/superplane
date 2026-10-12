package main

// Downloads the license key list, verifies it with the embedded root keys, and
// writes it as the bootstrap list for the next release. Run it as an explicit
// release step. Normal builds and CI must not depend on the issuer endpoint.

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/superplanehq/superplane/pkg/licensing"
)

func main() {
	keysURL := flag.String("url", licensing.DefaultKeysURL, "license key list URL")
	output := flag.String("output", "pkg/licensing/trustedkeys/license-keys.jws", "bootstrap key list file")
	flag.Parse()

	if err := run(*keysURL, *output); err != nil {
		fmt.Fprintln(os.Stderr, "license key update failed:", err)
		os.Exit(1)
	}
}

func run(keysURL, output string) error {
	roots, err := licensing.ProductionRootKeySet()
	if err != nil {
		return fmt.Errorf("embedded root keys are not valid: %w", err)
	}

	if roots.Len() == 0 {
		return fmt.Errorf("no root keys are embedded")
	}

	raw, err := fetch(keysURL)
	if err != nil {
		return err
	}

	list, err := licensing.VerifyKeyList(raw, roots)
	if err != nil {
		return err
	}

	if current, err := os.ReadFile(output); err == nil && len(current) > 0 {
		previous, err := licensing.VerifyKeyList(current, roots)
		if err == nil && previous.Version > list.Version {
			return fmt.Errorf("downloaded version %d is older than bundled version %d", list.Version, previous.Version)
		}
	}

	if err := os.WriteFile(output, []byte(list.Document+"\n"), 0o644); err != nil {
		return fmt.Errorf("write bootstrap key list: %w", err)
	}

	fmt.Printf("Wrote key list version %d with keys %v to %s\n", list.Version, list.Keys.KeyIDs(), output)
	return nil
}

func fetch(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download license keys: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download license keys: unexpected status %d", response.StatusCode)
	}

	return io.ReadAll(io.LimitReader(response.Body, licensing.MaxKeyListBytes+1))
}
