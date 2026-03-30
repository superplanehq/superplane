package blobstorage

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestInMemoryBlobStorageRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewInMemoryBlobStorage()

	_, err := store.Put(context.Background(), PutInput{
		Key:         "blobs/execution/test/file.txt",
		Body:        stringsReader("hello"),
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	output, err := store.Get(context.Background(), "blobs/execution/test/file.txt")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer output.Body.Close()

	content, err := io.ReadAll(output.Body)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestFilesystemBlobStorageRoundTrip(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	store := NewFilesystemBlobStorage(basePath)

	_, err := store.Put(context.Background(), PutInput{
		Key:         "blobs/canvas/canvas-id/file.txt",
		Body:        stringsReader("content"),
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	output, err := store.Get(context.Background(), "blobs/canvas/canvas-id/file.txt")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer output.Body.Close()

	content, err := io.ReadAll(output.Body)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(content) != "content" {
		t.Fatalf("unexpected content: %q", string(content))
	}

	err = store.Delete(context.Background(), "blobs/canvas/canvas-id/file.txt")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = store.Get(context.Background(), "blobs/canvas/canvas-id/file.txt")
	if err != ErrBlobNotFound {
		t.Fatalf("expected ErrBlobNotFound, got %v", err)
	}
}

func TestFilesystemBlobStorageRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	store := NewFilesystemBlobStorage(basePath)

	_, err := store.Put(context.Background(), PutInput{
		Key:  "../outside.txt",
		Body: stringsReader("bad"),
	})
	if err == nil {
		t.Fatalf("expected error for path traversal")
	}

}

func TestNewFromEnvDefaultsToMemory(t *testing.T) {
	unsetEnv(t, EnvBlobStorageBackend)

	store, err := NewFromEnv()
	if err != nil {
		t.Fatalf("new from env failed: %v", err)
	}
	if _, ok := store.(*InMemoryBlobStorage); !ok {
		t.Fatalf("expected in-memory store")
	}
}

func stringsReader(value string) io.Reader {
	return strings.NewReader(value)
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	currentValue, hasValue := os.LookupEnv(key)
	if hasValue {
		t.Cleanup(func() {
			_ = os.Setenv(key, currentValue)
		})
	} else {
		t.Cleanup(func() {
			_ = os.Unsetenv(key)
		})
	}

	_ = os.Unsetenv(key)
}
