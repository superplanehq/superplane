package api

import (
	"strings"
	"testing"
)

func TestValidateFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files []TaskFile
		ok    bool
	}{
		{
			name:  "empty ok",
			files: nil,
			ok:    true,
		},
		{
			name:  "valid nested",
			files: []TaskFile{{Path: "steps/01.sh", Content: "#!/bin/bash\n", Mode: "0755"}},
			ok:    true,
		},
		{
			name:  "empty content ok",
			files: []TaskFile{{Path: "empty.txt", Content: ""}},
			ok:    true,
		},
		{
			name:  "missing path",
			files: []TaskFile{{Path: "", Content: "x"}},
			ok:    false,
		},
		{
			name:  "absolute path",
			files: []TaskFile{{Path: "/etc/passwd", Content: "x"}},
			ok:    false,
		},
		{
			name:  "parent traversal",
			files: []TaskFile{{Path: "../secret", Content: "x"}},
			ok:    false,
		},
		{
			name:  "nested parent traversal",
			files: []TaskFile{{Path: "a/../../b", Content: "x"}},
			ok:    false,
		},
		{
			name: "duplicate path",
			files: []TaskFile{
				{Path: "a.txt", Content: "1"},
				{Path: "./a.txt", Content: "2"},
			},
			ok: false,
		},
		{
			name:  "nul content",
			files: []TaskFile{{Path: "a.txt", Content: "a\x00b"}},
			ok:    false,
		},
		{
			name:  "bad mode",
			files: []TaskFile{{Path: "a.txt", Content: "x", Mode: "rwx"}},
			ok:    false,
		},
		{
			name:  "too large file",
			files: []TaskFile{{Path: "big.txt", Content: strings.Repeat("x", MaxTaskFileBytes+1)}},
			ok:    false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateFiles(tt.files)
			if tt.ok && got != "" {
				t.Fatalf("expected ok, got %q", got)
			}
			if !tt.ok && got == "" {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeFiles(t *testing.T) {
	t.Parallel()

	got := NormalizeFiles([]TaskFile{
		{Path: "./steps/../steps/01.sh", Content: "hi", Mode: " 0755 "},
	})
	if len(got) != 1 {
		t.Fatalf("got %#v", got)
	}
	if got[0].Path != "steps/01.sh" {
		t.Fatalf("path = %q", got[0].Path)
	}
	if got[0].Mode != "0755" {
		t.Fatalf("mode = %q", got[0].Mode)
	}
}
