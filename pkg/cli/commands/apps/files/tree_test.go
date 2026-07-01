package files

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test__FileTree__WriteNestedPaths(t *testing.T) {
	tree := NewFileTree([]string{
		"docs/guide.md",
		"README.md",
		"docs/setup/install.md",
	})

	var output bytes.Buffer
	require.NoError(t, tree.Write(&output))
	require.Equal(t, `├── README.md
└── docs/
    ├── guide.md
    └── setup/
        └── install.md
`, output.String())
}
