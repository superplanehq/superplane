package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtyExitAliasBootstrap(t *testing.T) {
	t.Parallel()

	assert.Contains(t, ptyExitAliasBootstrap, "shopt -s expand_aliases")
	assert.Contains(t, ptyExitAliasBootstrap, "alias exit=return")
	assert.NotContains(t, ptyExitAliasBootstrap, "\n")
}
