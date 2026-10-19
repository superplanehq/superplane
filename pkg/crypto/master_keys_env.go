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
	configured := os.Getenv("ENCRYPTION_MASTER_KEYS")
	if configured == "" {
		return nil, "", nil
	}
	var localEntries, kmsEntries []string
	var primary string
	seen := make(map[string]bool)
	for entry := range strings.SplitSeq(configured, ",") {
		entry = strings.TrimSpace(entry)
		var reference string
		switch {
		case strings.HasPrefix(entry, "env://"):
			name, value, ok := strings.Cut(strings.TrimPrefix(entry, "env://"), ":")
			if !ok || !validMasterKeyID(name) || value == "" {
				return nil, "", fmt.Errorf("invalid env master key entry")
			}
			reference = "env://" + name
			localEntries = append(localEntries, name+":"+value)
		case strings.HasPrefix(entry, "gcp-kms://"):
			reference = entry
			kmsEntries = append(kmsEntries, entry)
		default:
			return nil, "", fmt.Errorf("unsupported encryption master key entry")
		}
		if seen[reference] {
			return nil, "", fmt.Errorf("duplicate encryption master key reference %q", reference)
		}
		seen[reference] = true
		if primary == "" {
			primary = reference
		}
	}
	var providers []MasterKeys
	if len(localEntries) > 0 {
		local, err := ParseLocalMasterKeys(strings.Join(localEntries, ","))
		if err != nil {
			return nil, "", err
		}
		providers = append(providers, local)
	}
	if len(kmsEntries) > 0 {
		kms, err := NewGCPMasterKeys(ctx, strings.Join(kmsEntries, ","))
		if err != nil {
			return nil, "", err
		}
		providers = append(providers, kms)
	}
	return &ConfiguredMasterKeys{providers: providers}, primary, nil
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
