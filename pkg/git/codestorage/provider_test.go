package codestorage

import (
	"context"
	"strings"
	"testing"

	codestorage "github.com/pierrecomputer/sdk/packages/code-storage-go"
	"github.com/stretchr/testify/require"
)

// testKey is the code-storage-go SDK test ECDSA private key.
const testKey = "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgy3DPdzzsP6tOOvmorjbx6L7mpFmKKL2hNWNW3urkN8ehRANCAAQ7/DPhGH3kaWl0YEIO+W9WmhyCclDGyTh6suablSura7ZDG8hpm3oNsq/ykC3Scfsw6ZTuuVuLlXKV/be/Xr0d\n-----END PRIVATE KEY-----\n"

func Test__Provider__RepositoryURL(t *testing.T) {
	client, err := codestorage.NewClient(codestorage.Options{
		Name:           "acme",
		Key:            testKey,
		StorageBaseURL: "acme.code.storage",
	})
	require.NoError(t, err)

	provider := &Provider{
		client:        client,
		defaultBranch: "main",
	}

	repoID := "orgs/org-1/my-app"
	remote, err := provider.RepositoryURL(context.Background(), repoID, "My App")
	require.NoError(t, err)
	require.Contains(t, remote, "acme.code.storage/"+repoID+".git")
	require.True(t, strings.HasPrefix(remote, "https://t:"))
}
