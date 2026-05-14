package api

import "testing"

func TestValidateExecutionTimeoutSeconds(t *testing.T) {
	if got := ValidateExecutionTimeoutSeconds(nil); got != "" {
		t.Fatalf("nil: %q", got)
	}
	one := 1
	if got := ValidateExecutionTimeoutSeconds(&one); got != "" {
		t.Fatalf("1: %q", got)
	}
	max := MaxExecutionTimeoutSecondsRequest
	if got := ValidateExecutionTimeoutSeconds(&max); got != "" {
		t.Fatalf("max: %q", got)
	}
	zero := 0
	if got := ValidateExecutionTimeoutSeconds(&zero); got == "" {
		t.Fatal("expected error for 0")
	}
	tooHigh := MaxExecutionTimeoutSecondsRequest + 1
	if got := ValidateExecutionTimeoutSeconds(&tooHigh); got == "" {
		t.Fatal("expected error for above max")
	}
}
