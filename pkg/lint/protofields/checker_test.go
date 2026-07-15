package protofields_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/superplanehq/superplane/pkg/lint/protofields"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		proto   string
		wantMsg string // qualified message name expected to have a gap ("" = none)
		missing []int
		have    []int
	}{
		{
			name: "contiguous message is OK",
			proto: `syntax = "proto3";
message Foo {
  string id = 1;
  string name = 2;
  string description = 3;
}`,
		},
		{
			name: "gap fails",
			proto: `syntax = "proto3";
message Foo {
  string id = 1;
  string name = 3;
}`,
			wantMsg: "Foo",
			missing: []int{2},
			have:    []int{1, 3},
		},
		{
			name: "gap does not need to start at 1",
			proto: `syntax = "proto3";
message Foo {
  string a = 2;
  string b = 4;
}`,
			wantMsg: "Foo",
			missing: []int{3},
			have:    []int{2, 4},
		},
		{
			name: "nested message is reported with a qualified name",
			proto: `syntax = "proto3";
message Outer {
  message Inner {
    string a = 1;
    string b = 3;
  }
  Inner inner = 1;
}`,
			wantMsg: "Outer.Inner",
			missing: []int{2},
			have:    []int{1, 3},
		},
		{
			name: "oneof fields count toward the enclosing message",
			proto: `syntax = "proto3";
message Foo {
  string id = 1;
  oneof value {
    string text = 2;
    int32 number = 4;
  }
}`,
			wantMsg: "Foo",
			missing: []int{3},
			have:    []int{1, 2, 4},
		},
		{
			name: "map fields count",
			proto: `syntax = "proto3";
message Foo {
  string id = 1;
  map<string, string> labels = 3;
}`,
			wantMsg: "Foo",
			missing: []int{2},
			have:    []int{1, 3},
		},
		{
			name: "empty message is OK",
			proto: `syntax = "proto3";
message Foo {
}`,
		},
		{
			name: "enum values are ignored",
			proto: `syntax = "proto3";
message Foo {
  enum State {
    STATE_UNSPECIFIED = 0;
    STATE_READY = 2;
  }
  State state = 1;
}`,
		},
		{
			name: "reserved numbers are treated as gaps",
			proto: `syntax = "proto3";
message Foo {
  reserved 2;
  string id = 1;
  string name = 3;
}`,
			wantMsg: "Foo",
			missing: []int{2},
			have:    []int{1, 3},
		},
		{
			name: "comments and service blocks are ignored",
			proto: `syntax = "proto3";
service Svc {
  rpc Do(Foo) returns (Foo);
}
message Foo {
  // leading comment = 99
  string id = 1; /* inline = 99 */
  string name = 2;
}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "test.proto"), tc.proto)

			issues, err := protofields.Run(dir)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			if tc.wantMsg == "" {
				if len(issues) != 0 {
					t.Fatalf("Run() = %+v, want no issues", issues)
				}
				return
			}

			if len(issues) != 1 {
				t.Fatalf("Run() = %+v, want exactly one issue", issues)
			}

			got := issues[0]
			if got.Message != tc.wantMsg {
				t.Errorf("Message = %q, want %q", got.Message, tc.wantMsg)
			}
			if !equalInts(got.Missing, tc.missing) {
				t.Errorf("Missing = %v, want %v", got.Missing, tc.missing)
			}
			if !equalInts(got.Have, tc.have) {
				t.Errorf("Have = %v, want %v", got.Have, tc.have)
			}
		})
	}
}

func TestRunExcludesNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	include := filepath.Join(dir, "include")
	if err := os.Mkdir(include, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(include, "vendored.proto"), `syntax = "proto3";
message Vendored {
  string a = 1;
  string b = 3;
}`)

	issues, err := protofields.Run(dir)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("Run() = %+v, want no issues for nested directories", issues)
	}
}

func TestIssueString(t *testing.T) {
	issue := protofields.Issue{
		Path:    "protos/canvases.proto",
		Message: "CanvasSummary",
		Missing: []int{4, 11, 13},
		Have:    []int{1, 2, 3, 5, 6, 7, 8, 9, 10, 12, 14},
	}

	want := "protos/canvases.proto: CanvasSummary: field numbers have gaps: missing 4, 11, 13 (have 1–3, 5–10, 12, 14)"
	if got := issue.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
