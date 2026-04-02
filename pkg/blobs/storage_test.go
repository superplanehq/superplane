package blobs

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMemoryStorageRoundTrip(t *testing.T) {
	t.Parallel()
	runRoundTripTest(t, NewMemoryStorage())
}

func TestFilesystemStorageRoundTrip(t *testing.T) {
	t.Parallel()
	runRoundTripTest(t, NewFilesystemStorage(t.TempDir()))
}

func TestMemoryStorageList(t *testing.T) {
	t.Parallel()
	runListTest(t, NewMemoryStorage())
}

func TestFilesystemStorageList(t *testing.T) {
	t.Parallel()
	runListTest(t, NewFilesystemStorage(t.TempDir()))
}

func TestMemoryStorageListPagination(t *testing.T) {
	t.Parallel()
	runListPaginationTest(t, NewMemoryStorage())
}

func TestFilesystemStorageListPagination(t *testing.T) {
	t.Parallel()
	runListPaginationTest(t, NewFilesystemStorage(t.TempDir()))
}

func TestMemoryStorageScopeIsolation(t *testing.T) {
	t.Parallel()
	runScopeIsolationTest(t, NewMemoryStorage())
}

func TestFilesystemStorageScopeIsolation(t *testing.T) {
	t.Parallel()
	runScopeIsolationTest(t, NewFilesystemStorage(t.TempDir()))
}

func TestMemoryStoragePresignReturnsNotSupported(t *testing.T) {
	t.Parallel()
	runPresignNotSupportedTest(t, NewMemoryStorage())
}

func TestFilesystemStoragePresignReturnsNotSupported(t *testing.T) {
	t.Parallel()
	runPresignNotSupportedTest(t, NewFilesystemStorage(t.TempDir()))
}

func TestFilesystemStorageRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	store := NewFilesystemStorage(t.TempDir())
	scope := Scope{Type: ScopeOrganization, OrganizationID: "org-1"}

	err := store.Put(context.Background(), scope, "../../../etc/passwd", strings.NewReader("bad"), PutOptions{})
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestObjectKeyRejectsUnknownScope(t *testing.T) {
	t.Parallel()

	_, err := objectKey(Scope{Type: ScopeType("invalid")}, "file.txt")
	if !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected ErrInvalidScope for unknown scope type, got %v", err)
	}
}

func TestObjectKeyRejectsMissingScopeIDs(t *testing.T) {
	t.Parallel()

	cases := []Scope{
		{Type: ScopeOrganization},
		{Type: ScopeCanvas},
		{Type: ScopeNode, CanvasID: "canvas-only"},
		{Type: ScopeNode, NodeID: "node-only"},
		{Type: ScopeExecution},
	}

	for _, scope := range cases {
		_, err := objectKey(scope, "file.txt")
		if !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("expected ErrInvalidScope for scope %+v, got %v", scope, err)
		}
	}
}

func TestListRejectsInvalidScope(t *testing.T) {
	t.Parallel()

	stores := []Storage{
		NewMemoryStorage(),
		NewFilesystemStorage(t.TempDir()),
	}

	for _, store := range stores {
		_, err := store.List(context.Background(), Scope{Type: ScopeType("invalid")}, ListInput{})
		if !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("expected ErrInvalidScope for %T, got %v", store, err)
		}
	}
}

func TestNewFromEnvDefaultsToMemory(t *testing.T) {
	unsetEnv(t, EnvBackend)

	store, err := NewFromEnv()
	if err != nil {
		t.Fatalf("NewFromEnv failed: %v", err)
	}
	if _, ok := store.(*MemoryStorage); !ok {
		t.Fatal("expected MemoryStorage as default backend")
	}
}

func TestNewFromEnvFilesystem(t *testing.T) {
	setEnv(t, EnvBackend, BackendFilesystem)
	setEnv(t, EnvFilesystemPath, t.TempDir())

	store, err := NewFromEnv()
	if err != nil {
		t.Fatalf("NewFromEnv failed: %v", err)
	}
	if _, ok := store.(*FilesystemStorage); !ok {
		t.Fatal("expected FilesystemStorage")
	}
}

func TestNewFromEnvFilesystemMissingPath(t *testing.T) {
	setEnv(t, EnvBackend, BackendFilesystem)
	unsetEnv(t, EnvFilesystemPath)

	_, err := NewFromEnv()
	if err == nil {
		t.Fatal("expected error when filesystem path is not set")
	}
}

func TestNewFromEnvUnknownBackend(t *testing.T) {
	setEnv(t, EnvBackend, "nosuchbackend")

	_, err := NewFromEnv()
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

// --- shared test helpers ---

func runRoundTripTest(t *testing.T, store Storage) {
	t.Helper()
	ctx := context.Background()
	scope := Scope{Type: ScopeOrganization, OrganizationID: "org-1"}

	err := store.Put(ctx, scope, "test/file.txt", strings.NewReader("hello"), PutOptions{ContentType: "text/plain"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	reader, err := store.Get(ctx, scope, "test/file.txt")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("expected %q, got %q", "hello", string(content))
	}

	err = store.Delete(ctx, scope, "test/file.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(ctx, scope, "test/file.txt")
	if !errors.Is(err, ErrBlobNotFound) {
		t.Fatalf("expected ErrBlobNotFound after delete, got %v", err)
	}
}

func runListTest(t *testing.T, store Storage) {
	t.Helper()
	ctx := context.Background()
	scope := Scope{Type: ScopeCanvas, CanvasID: "canvas-1"}

	for _, name := range []string{"a.txt", "b.txt", "sub/c.txt"} {
		err := store.Put(ctx, scope, name, strings.NewReader("data"), PutOptions{})
		if err != nil {
			t.Fatalf("Put %s failed: %v", name, err)
		}
	}

	out, err := store.List(ctx, scope, ListInput{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(out.Blobs) != 3 {
		t.Fatalf("expected 3 blobs, got %d", len(out.Blobs))
	}
}

func runListPaginationTest(t *testing.T, store Storage) {
	t.Helper()
	ctx := context.Background()
	scope := Scope{Type: ScopeExecution, ExecutionID: "exec-1"}

	for _, name := range []string{"1.txt", "2.txt", "3.txt", "4.txt", "5.txt"} {
		err := store.Put(ctx, scope, name, strings.NewReader("data"), PutOptions{})
		if err != nil {
			t.Fatalf("Put %s failed: %v", name, err)
		}
	}

	page1, err := store.List(ctx, scope, ListInput{MaxResults: 2})
	if err != nil {
		t.Fatalf("List page 1 failed: %v", err)
	}
	if len(page1.Blobs) != 2 {
		t.Fatalf("expected 2 blobs on page 1, got %d", len(page1.Blobs))
	}
	if page1.NextToken == "" {
		t.Fatal("expected non-empty NextToken for page 1")
	}

	page2, err := store.List(ctx, scope, ListInput{MaxResults: 2, ContinuationToken: page1.NextToken})
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}
	if len(page2.Blobs) != 2 {
		t.Fatalf("expected 2 blobs on page 2, got %d", len(page2.Blobs))
	}

	page3, err := store.List(ctx, scope, ListInput{MaxResults: 2, ContinuationToken: page2.NextToken})
	if err != nil {
		t.Fatalf("List page 3 failed: %v", err)
	}
	if len(page3.Blobs) != 1 {
		t.Fatalf("expected 1 blob on page 3, got %d", len(page3.Blobs))
	}
	if page3.NextToken != "" {
		t.Fatalf("expected empty NextToken on last page, got %q", page3.NextToken)
	}
}

func runScopeIsolationTest(t *testing.T, store Storage) {
	t.Helper()
	ctx := context.Background()

	scopeA := Scope{Type: ScopeOrganization, OrganizationID: "org-a"}
	scopeB := Scope{Type: ScopeOrganization, OrganizationID: "org-b"}

	err := store.Put(ctx, scopeA, "shared.txt", strings.NewReader("from A"), PutOptions{})
	if err != nil {
		t.Fatalf("Put to scope A failed: %v", err)
	}

	err = store.Put(ctx, scopeB, "shared.txt", strings.NewReader("from B"), PutOptions{})
	if err != nil {
		t.Fatalf("Put to scope B failed: %v", err)
	}

	readerA, err := store.Get(ctx, scopeA, "shared.txt")
	if err != nil {
		t.Fatalf("Get from scope A failed: %v", err)
	}
	defer readerA.Close()
	contentA, _ := io.ReadAll(readerA)

	readerB, err := store.Get(ctx, scopeB, "shared.txt")
	if err != nil {
		t.Fatalf("Get from scope B failed: %v", err)
	}
	defer readerB.Close()
	contentB, _ := io.ReadAll(readerB)

	if string(contentA) != "from A" {
		t.Fatalf("scope A content: expected %q, got %q", "from A", string(contentA))
	}
	if string(contentB) != "from B" {
		t.Fatalf("scope B content: expected %q, got %q", "from B", string(contentB))
	}

	listA, _ := store.List(ctx, scopeA, ListInput{})
	listB, _ := store.List(ctx, scopeB, ListInput{})

	if len(listA.Blobs) != 1 {
		t.Fatalf("scope A should have 1 blob, got %d", len(listA.Blobs))
	}
	if len(listB.Blobs) != 1 {
		t.Fatalf("scope B should have 1 blob, got %d", len(listB.Blobs))
	}
}

func runPresignNotSupportedTest(t *testing.T, store Storage) {
	t.Helper()
	ctx := context.Background()
	scope := Scope{Type: ScopeOrganization, OrganizationID: "org-1"}

	_, err := store.PresignPut(ctx, scope, "file.txt", PutOptions{}, 0)
	if !errors.Is(err, ErrPresignedURLNotSupported) {
		t.Fatalf("PresignPut: expected ErrPresignedURLNotSupported, got %v", err)
	}

	_, err = store.PresignGet(ctx, scope, "file.txt", 0)
	if !errors.Is(err, ErrPresignedURLNotSupported) {
		t.Fatalf("PresignGet: expected ErrPresignedURLNotSupported, got %v", err)
	}
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
	_ = os.Setenv(key, value)
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
	_ = os.Unsetenv(key)
}
