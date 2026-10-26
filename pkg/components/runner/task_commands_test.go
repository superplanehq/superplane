package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskCommandsFromLines(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []TaskCommand{
		{Command: "echo a", Kind: LiveLogKindBash, Preview: "echo a"},
		{Command: "echo b", Kind: LiveLogKindBash, Preview: "echo b"},
	}, TaskCommandsFromLines([]string{"  echo a  ", "", "echo b"}))
}
