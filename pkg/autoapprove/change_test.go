package autoapprove

import "testing"

func TestChangeFromPayload(t *testing.T) {
	t.Run("files as strings", func(t *testing.T) {
		c := ChangeFromPayload(map[string]any{"files": []any{"README.md", "docs/x.md"}})
		if !c.Known || len(c.Paths) != 2 {
			t.Fatalf("expected 2 known paths, got %+v", c)
		}
	})

	t.Run("files as objects with a path key", func(t *testing.T) {
		c := ChangeFromPayload(map[string]any{
			"changed_files": []any{
				map[string]any{"path": "pkg/a.go"},
				map[string]any{"filename": "pkg/b.go"},
			},
		})
		if !c.Known || len(c.Paths) != 2 {
			t.Fatalf("expected 2 paths, got %+v", c)
		}
	})

	t.Run("unreadable payload is not known", func(t *testing.T) {
		if ChangeFromPayload("not a map").Known {
			t.Error("string payload should not be Known")
		}
		if ChangeFromPayload(map[string]any{"unrelated": 1}).Known {
			t.Error("payload with no files should not be Known")
		}
	})

	// Fail-closed: a files list we only partially understand must not be
	// classified on the entries we could read. It goes to a human instead.
	t.Run("partial extraction fails closed", func(t *testing.T) {
		cases := []any{
			map[string]any{"files": []any{"README.md", 12345}},
			map[string]any{"files": []any{"README.md", map[string]any{"nopath": "x"}}},
			map[string]any{"files": []any{"README.md", map[string]any{"path": ""}}},
			map[string]any{"files": "README.md"}, // not a list at all
		}
		for i, payload := range cases {
			if ChangeFromPayload(payload).Known {
				t.Errorf("case %d: an unresolvable entry must fail closed (Known=false)", i)
			}
		}
	})
}
