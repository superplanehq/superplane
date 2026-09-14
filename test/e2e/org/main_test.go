package org

import (
	"os"
	"testing"

	"github.com/superplanehq/superplane/test/e2e/harness"
	"github.com/superplanehq/superplane/test/e2e/session"
)

var ctx *harness.TestContext

func TestMain(m *testing.M) {
	ctx = harness.NewTestContext()
	ctx.Start()
	if err := session.ResetTestDatabase(); err != nil {
		panic("reset test database: " + err.Error())
	}

	code := m.Run()
	ctx.Shutdown()
	os.Exit(code)
}
