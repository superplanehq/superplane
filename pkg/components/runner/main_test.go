package runner

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	runModelCallOutcomeInsert = func(fn func()) { fn() }
	os.Exit(m.Run())
}
