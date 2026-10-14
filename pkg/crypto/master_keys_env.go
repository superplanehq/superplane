package crypto

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type ConfiguredMasterKeys struct {
	providers []MasterKeys
}

func MasterKeysFromEnv(ctx context.Context) (MasterKeys, string, error) {
	var providers []MasterKeys
	var references []string
	if configured := os.Getenv("ENCRYPTION_LOCAL_MASTER_KEYS"); configured != "" {
		local, err := ParseLocalMasterKeys(configured)
		if err != nil {
			return nil, "", err
		}
		providers = append(providers, local)
		for entry := range strings.SplitSeq(configured, ",") {
			name, _, _ := strings.Cut(entry, ":")
			references = append(references, name)
		}
	}
	kmsURIs := os.Getenv("ENCRYPTION_KMS_KEY_URIS")
	if kmsURIs == "" {
		kmsURIs = os.Getenv("ENCRYPTION_KMS_KEY_URI")
	}
	if kmsURIs != "" {
		kms, err := NewGCPMasterKeys(ctx, kmsURIs)
		if err != nil {
			return nil, "", err
		}
		providers = append(providers, kms)
		for uri := range strings.SplitSeq(kmsURIs, ",") {
			if uri != "" {
				references = append(references, uri)
			}
		}
	}
	if len(providers) == 0 {
		return nil, "", nil
	}
	keys := &ConfiguredMasterKeys{providers: providers}
	initial := os.Getenv("ENCRYPTION_INITIAL_MASTER_KEY_REF")
	if initial == "" && len(references) == 1 {
		initial = references[0]
	}
	if initial != "" && !keys.Has(initial) {
		return nil, "", fmt.Errorf("ENCRYPTION_INITIAL_MASTER_KEY_REF must identify a configured master key")
	}
	return keys, initial, nil
}

func (k *ConfiguredMasterKeys) Has(reference string) bool {
	for _, provider := range k.providers {
		if provider.Has(reference) {
			return true
		}
	}
	return false
}

func (k *ConfiguredMasterKeys) Wrap(ctx context.Context, reference string, plain, associatedData []byte) ([]byte, error) {
	for _, provider := range k.providers {
		if provider.Has(reference) {
			return provider.Wrap(ctx, reference, plain, associatedData)
		}
	}
	return nil, fmt.Errorf("master key %q is not configured", reference)
}

func (k *ConfiguredMasterKeys) Unwrap(ctx context.Context, reference string, wrapped, associatedData []byte) ([]byte, error) {
	for _, provider := range k.providers {
		if provider.Has(reference) {
			return provider.Unwrap(ctx, reference, wrapped, associatedData)
		}
	}
	return nil, fmt.Errorf("master key %q is not configured", reference)
}
