package crypto

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
)

type testDataKeys struct {
	active DataKey
	keys   map[uuid.UUID]DataKey
}

func (k *testDataKeys) Active(_ context.Context) (DataKey, error) {
	return k.active, nil
}

func (k *testDataKeys) Find(_ context.Context, id uuid.UUID) (DataKey, error) {
	return k.keys[id], nil
}

func TestVersionedEncryptorReadsOldDataKeyAfterRotation(t *testing.T) {
	first := DataKey{ID: uuid.New(), Material: bytes.Repeat([]byte{1}, 32)}
	second := DataKey{ID: uuid.New(), Material: bytes.Repeat([]byte{2}, 32)}
	keys := &testDataKeys{active: first, keys: map[uuid.UUID]DataKey{first.ID: first, second.ID: second}}
	encryptor := NewVersionedEncryptor(keys, NewAESGCMEncryptor(bytes.Repeat([]byte{3}, 32)))
	ctx := context.Background()

	oldValue, err := encryptor.Encrypt(ctx, []byte("secret"), []byte("record-1"))
	if err != nil {
		t.Fatal(err)
	}
	keys.active = second
	newValue, err := encryptor.Encrypt(ctx, []byte("new secret"), []byte("record-2"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		value []byte
		aad   []byte
		want  []byte
	}{{oldValue, []byte("record-1"), []byte("secret")}, {newValue, []byte("record-2"), []byte("new secret")}} {
		plain, err := encryptor.Decrypt(ctx, item.value, item.aad)
		if err != nil || !bytes.Equal(plain, item.want) {
			t.Fatalf("decrypt: %q, %v", plain, err)
		}
	}
	if _, err := encryptor.Decrypt(ctx, oldValue, []byte("record-2")); err == nil {
		t.Fatal("different associated data decrypted the value")
	}
}

func TestVersionedEncryptorReadsLegacyCiphertext(t *testing.T) {
	legacy := NewAESGCMEncryptor(bytes.Repeat([]byte{3}, 32))
	oldValue, err := legacy.Encrypt(context.Background(), []byte("legacy"), []byte("record"))
	if err != nil {
		t.Fatal(err)
	}
	encryptor := NewVersionedEncryptor(&testDataKeys{}, legacy)
	plain, err := encryptor.Decrypt(context.Background(), oldValue, []byte("record"))
	if err != nil || !bytes.Equal(plain, []byte("legacy")) {
		t.Fatalf("decrypt legacy: %q, %v", plain, err)
	}
}

func TestVersionedEncryptorLegacyWriteModeReadsBothFormats(t *testing.T) {
	ctx := context.Background()
	key := DataKey{ID: uuid.New(), Material: bytes.Repeat([]byte{1}, 32)}
	keys := &testDataKeys{active: key, keys: map[uuid.UUID]DataKey{key.ID: key}}
	legacy := NewAESGCMEncryptor(bytes.Repeat([]byte{2}, 32))
	versioned := NewVersionedEncryptor(keys, legacy)
	compatible := versioned.WithLegacyWrites()
	aad := []byte("record")

	legacyValue, err := compatible.Encrypt(ctx, []byte("old"), aad)
	if err != nil {
		t.Fatal(err)
	}
	if _, versionedValue, err := EncryptedDataKeyID(legacyValue); err != nil || versionedValue {
		t.Fatalf("legacy write has versioned envelope: %v", err)
	}
	newValue, err := versioned.Encrypt(ctx, []byte("new"), aad)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		value []byte
		want  string
	}{{legacyValue, "old"}, {newValue, "new"}} {
		plain, err := compatible.Decrypt(ctx, item.value, aad)
		if err != nil || string(plain) != item.want {
			t.Fatalf("decrypt compatible value: %q, %v", plain, err)
		}
	}
}
