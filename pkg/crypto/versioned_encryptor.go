package crypto

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var envelopeMarker = []byte("SPENC1")

type DataKey struct {
	ID       uuid.UUID
	Material []byte
}

type DataKeys interface {
	Active(context.Context) (DataKey, error)
	Find(context.Context, uuid.UUID) (DataKey, error)
}

type VersionedEncryptor struct {
	keys DataKeys
}

func NewVersionedEncryptor(keys DataKeys) *VersionedEncryptor {
	return &VersionedEncryptor{keys: keys}
}

func (e *VersionedEncryptor) WithKeys(keys DataKeys) *VersionedEncryptor {
	copy := *e
	copy.keys = keys
	return &copy
}

func (e *VersionedEncryptor) Encrypt(ctx context.Context, plain, associatedData []byte) ([]byte, error) {
	key, err := e.keys.Active(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active data key: %w", err)
	}
	aead, err := dataAEAD(key.Material)
	if err != nil {
		return nil, err
	}
	header := make([]byte, 0, len(envelopeMarker)+len(key.ID))
	header = append(header, envelopeMarker...)
	header = append(header, key.ID[:]...)
	ciphertext := aead.Seal(nil, nil, plain, envelopeAssociatedData(header, associatedData))
	return append(header, ciphertext...), nil
}

func (e *VersionedEncryptor) Decrypt(ctx context.Context, value, associatedData []byte) ([]byte, error) {
	if !bytes.HasPrefix(value, envelopeMarker) {
		return nil, errors.New("encrypted value has no versioned envelope")
	}
	headerLength := len(envelopeMarker) + len(uuid.UUID{})
	if len(value) < headerLength+28 {
		return nil, errors.New("encrypted value is too short")
	}
	header := value[:headerLength]
	keyID, err := uuid.FromBytes(header[len(envelopeMarker):])
	if err != nil {
		return nil, fmt.Errorf("read data key ID: %w", err)
	}
	key, err := e.keys.Find(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("load data key %s: %w", keyID, err)
	}
	aead, err := dataAEAD(key.Material)
	if err != nil {
		return nil, err
	}
	plain, err := aead.Open(nil, nil, value[headerLength:], envelopeAssociatedData(header, associatedData))
	if err != nil {
		return nil, fmt.Errorf("decrypt value with data key %s: %w", keyID, err)
	}
	return plain, nil
}

func dataAEAD(material []byte) (cipher.AEAD, error) {
	if len(material) != 32 {
		return nil, errors.New("data key must contain 32 bytes")
	}
	block, err := aes.NewCipher(material)
	if err != nil {
		return nil, fmt.Errorf("create data cipher: %w", err)
	}
	return cipher.NewGCMWithRandomNonce(block)
}

func envelopeAssociatedData(header, associatedData []byte) []byte {
	result := make([]byte, 0, len(header)+len(associatedData))
	result = append(result, header...)
	return append(result, associatedData...)
}

func EncryptedDataKeyID(value []byte) (uuid.UUID, bool, error) {
	if !bytes.HasPrefix(value, envelopeMarker) {
		return uuid.Nil, false, nil
	}
	if len(value) < len(envelopeMarker)+len(uuid.UUID{})+28 {
		return uuid.Nil, true, errors.New("encrypted value is too short")
	}
	id, err := uuid.FromBytes(value[len(envelopeMarker) : len(envelopeMarker)+len(uuid.UUID{})])
	return id, true, err
}
