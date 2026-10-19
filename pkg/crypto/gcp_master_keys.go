package crypto

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"google.golang.org/api/cloudkms/v1"
)

// GCPMasterKeys wraps data keys with the configured Cloud KMS CryptoKeys.
type GCPMasterKeys struct {
	service *cloudkms.Service
	allowed map[string]string
}

func NewGCPMasterKeys(ctx context.Context, configured string) (*GCPMasterKeys, error) {
	allowed := make(map[string]string)
	for uri := range strings.SplitSeq(configured, ",") {
		if uri == "" {
			continue
		}
		name, err := gcpCryptoKeyName(uri)
		if err != nil {
			return nil, err
		}
		allowed[uri] = name
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("at least one KMS key URI is required")
	}
	service, err := cloudkms.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Cloud KMS client: %w", err)
	}
	return &GCPMasterKeys{service: service, allowed: allowed}, nil
}

func gcpCryptoKeyName(uri string) (string, error) {
	name, ok := strings.CutPrefix(uri, "gcp-kms://")
	if !ok {
		return "", fmt.Errorf("invalid Cloud KMS key URI %q", uri)
	}
	parts := strings.Split(name, "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" ||
		parts[4] != "keyRings" || parts[6] != "cryptoKeys" {
		return "", fmt.Errorf("invalid Cloud KMS key URI %q", uri)
	}
	for _, index := range []int{1, 3, 5, 7} {
		if parts[index] == "" {
			return "", fmt.Errorf("invalid Cloud KMS key URI %q", uri)
		}
	}
	return name, nil
}

func (k *GCPMasterKeys) Has(reference string) bool {
	_, ok := k.allowed[reference]
	return ok
}

func (k *GCPMasterKeys) Wrap(ctx context.Context, reference string, plain, associatedData []byte) ([]byte, error) {
	name, ok := k.allowed[reference]
	if !ok {
		return nil, fmt.Errorf("Cloud KMS key %q is not configured", reference)
	}
	request := &cloudkms.EncryptRequest{
		Plaintext:                   base64.StdEncoding.EncodeToString(plain),
		AdditionalAuthenticatedData: base64.StdEncoding.EncodeToString(associatedData),
	}
	response, err := k.service.Projects.Locations.KeyRings.CryptoKeys.Encrypt(name, request).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("wrap data key with Cloud KMS: %w", err)
	}
	wrapped, err := base64.StdEncoding.DecodeString(response.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode Cloud KMS ciphertext: %w", err)
	}
	return wrapped, nil
}

func (k *GCPMasterKeys) Unwrap(ctx context.Context, reference string, wrapped, associatedData []byte) ([]byte, error) {
	name, ok := k.allowed[reference]
	if !ok {
		return nil, fmt.Errorf("Cloud KMS key %q is not configured", reference)
	}
	request := &cloudkms.DecryptRequest{
		Ciphertext:                  base64.StdEncoding.EncodeToString(wrapped),
		AdditionalAuthenticatedData: base64.StdEncoding.EncodeToString(associatedData),
	}
	response, err := k.service.Projects.Locations.KeyRings.CryptoKeys.Decrypt(name, request).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unwrap data key with Cloud KMS: %w", err)
	}
	plain, err := base64.StdEncoding.DecodeString(response.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("decode Cloud KMS plaintext: %w", err)
	}
	return plain, nil
}
