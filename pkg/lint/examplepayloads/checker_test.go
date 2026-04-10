package examplepayloads

import "testing"

func TestRun(t *testing.T) {
	issues, err := Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, issue := range issues {
		t.Errorf("%s", issue.String())
	}
}
