package provider

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test__RepositorySlug(t *testing.T) {
	t.Parallel()

	slug, err := RepositorySlug("My App")
	require.NoError(t, err)
	require.Equal(t, "My-App", slug)

	_, err = RepositorySlug("  ")
	require.Error(t, err)
}

func Test__RepositoryPath(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	path, err := RepositoryPath(orgID, "My App")
	require.NoError(t, err)
	require.Equal(t, "orgs/11111111-1111-1111-1111-111111111111/My-App", path)
}
