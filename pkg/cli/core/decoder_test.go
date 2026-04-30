package core

import (
	"strings"
	"testing"
)

func TestDecoderDecodeYAML(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	t.Run("accepts known fields", func(t *testing.T) {
		raw := []byte("name: valid")
		var out payload

		err := NewDecoder(raw).DecodeYAML(&out)
		if err != nil {
			t.Fatalf("DecodeYAML returned error: %v", err)
		}
		if out.Name != "valid" {
			t.Fatalf("expected name=valid, got %q", out.Name)
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		raw := []byte("name: valid\nunknown: true\n")
		var out payload

		err := NewDecoder(raw).DecodeYAML(&out)
		if err == nil {
			t.Fatal("expected DecodeYAML to fail for unknown field")
		}
		if !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("expected unknown field error, got %v", err)
		}
	})

}
