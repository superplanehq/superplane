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
	encryptor := NewVersionedEncryptor(keys)
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

func TestVersionedEncryptorRejectsUnmarkedCiphertext(t *testing.T) {
	encryptor := NewVersionedEncryptor(&testDataKeys{})
	if _, err := encryptor.Decrypt(context.Background(), []byte("unmarked"), []byte("record")); err == nil {
		t.Fatal("unmarked ciphertext was accepted")
	}
}
