package opaquetoken

import "testing"

func TestGenerateReturnsUniqueTokensWithStableHashes(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("generated duplicate tokens")
	}
	if len(first) != 43 {
		t.Fatalf("token length = %d, want 43", len(first))
	}
	if Hash(first) != Hash(" "+first+" ") {
		t.Fatal("hash should ignore surrounding whitespace")
	}
	if Hash(first) == Hash(second) {
		t.Fatal("different tokens have the same hash")
	}
}
