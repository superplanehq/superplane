package e2e

import (
	"os"
	"testing"
)

var ctx *TestContext

func TestMain(m *testing.M) {
	ctx = NewTestContext(m)
	ctx.Start()

	code := m.Run()
	os.Exit(code)
}
