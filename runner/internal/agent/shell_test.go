package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtyExitAliasBootstrap(t *testing.T) {
	t.Parallel()

	assert.Contains(t, ptyExitAliasBootstrap, "shopt -s expand_aliases")
	assert.Contains(t, ptyExitAliasBootstrap, "alias exit=return")
	assert.NotContains(t, ptyExitAliasBootstrap, "\n")
}

func TestWrapSourcedDirectiveErrexitTrap(t *testing.T) {
	t.Parallel()

	got := wrapSourcedDirective("echo hello\nfalse\necho there\n")
	assert.True(t, strings.HasPrefix(got, "set +e\n"))
	assert.Contains(t, got, "trap 'trap - ERR RETURN; set +e' RETURN\n")
	assert.Contains(t, got, "trap '_sp_runner_err=$?; trap - ERR; set +e; return \"$_sp_runner_err\"' ERR\n")
	assert.Contains(t, got, "set -e\n")
	assert.Contains(t, got, "echo hello\nfalse\necho there\n")
	assert.True(t, strings.HasSuffix(got, "trap - ERR RETURN\nset +e\n"))
	assert.NotContains(t, got, "mktemp")
	assert.NotContains(t, got, "(\n")
}
