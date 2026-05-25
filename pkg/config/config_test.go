package config

import "testing"

func TestLoadCanvasStorageConfigDefaults(t *testing.T) {
	t.Setenv("CANVAS_STORAGE_DRIVER", "")
	t.Setenv("CANVAS_STORAGE_LOCAL_ROOT", "")
	t.Setenv("CANVAS_STORAGE_DEFAULT_BRANCH", "")
	t.Setenv("CANVAS_STORAGE_MAX_FILE_BYTES", "")
	t.Setenv("CANVAS_STORAGE_MAX_COMMIT_BYTES", "")

	cfg := LoadCanvasStorageConfig()

	if cfg.Driver != CanvasStorageDriverDisabled {
		t.Fatalf("expected disabled driver, got %q", cfg.Driver)
	}
	if cfg.LocalRoot != defaultCanvasStorageLocalRoot {
		t.Fatalf("expected default local root, got %q", cfg.LocalRoot)
	}
	if cfg.DefaultBranch != defaultCanvasStorageDefaultBranch {
		t.Fatalf("expected default branch, got %q", cfg.DefaultBranch)
	}
	if cfg.MaxFileBytes != defaultCanvasStorageMaxFileBytes {
		t.Fatalf("expected default max file bytes, got %d", cfg.MaxFileBytes)
	}
	if cfg.MaxCommitBytes != defaultCanvasStorageMaxCommitBytes {
		t.Fatalf("expected default max commit bytes, got %d", cfg.MaxCommitBytes)
	}
}

func TestLoadCanvasStorageConfigEnv(t *testing.T) {
	t.Setenv("CANVAS_STORAGE_DRIVER", "local_git")
	t.Setenv("CANVAS_STORAGE_LOCAL_ROOT", "/tmp/canvas-repos")
	t.Setenv("CANVAS_STORAGE_DEFAULT_BRANCH", "trunk")
	t.Setenv("CANVAS_STORAGE_MAX_FILE_BYTES", "123")
	t.Setenv("CANVAS_STORAGE_MAX_COMMIT_BYTES", "456")
	t.Setenv("CODE_STORAGE_NAME", "acme")
	t.Setenv("CODE_STORAGE_PRIVATE_KEY_PATH", "/keys/code-storage.pem")

	cfg := LoadCanvasStorageConfig()

	if cfg.Driver != "local_git" ||
		cfg.LocalRoot != "/tmp/canvas-repos" ||
		cfg.DefaultBranch != "trunk" ||
		cfg.MaxFileBytes != 123 ||
		cfg.MaxCommitBytes != 456 ||
		cfg.CodeStorageName != "acme" ||
		cfg.CodeStoragePrivateKeyPath != "/keys/code-storage.pem" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadCanvasStorageConfigInvalidLimitsFallback(t *testing.T) {
	t.Setenv("CANVAS_STORAGE_MAX_FILE_BYTES", "-1")
	t.Setenv("CANVAS_STORAGE_MAX_COMMIT_BYTES", "not-a-number")

	cfg := LoadCanvasStorageConfig()

	if cfg.MaxFileBytes != defaultCanvasStorageMaxFileBytes {
		t.Fatalf("expected default max file bytes, got %d", cfg.MaxFileBytes)
	}
	if cfg.MaxCommitBytes != defaultCanvasStorageMaxCommitBytes {
		t.Fatalf("expected default max commit bytes, got %d", cfg.MaxCommitBytes)
	}
}
