package git

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/git/codestorage"
	"github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/git/supergit"
)

const testECPrivateKey = "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgy3DPdzzsP6tOOvmorjbx6L7mpFmKKL2hNWNW3urkN8ehRANCAAQ7/DPhGH3kaWl0YEIO+W9WmhyCclDGyTh6suablSura7ZDG8hpm3oNsq/ykC3Scfsw6ZTuuVuLlXKV/be/Xr0d\n-----END PRIVATE KEY-----\n"

func TestNewProviderRequiresEnv(t *testing.T) {
	t.Setenv("GIT_STORAGE_PROVIDER", "")
	_, err := NewProvider()
	require.Error(t, err)
	require.Contains(t, err.Error(), "GIT_STORAGE_PROVIDER is not set")
}

func TestNewProviderCodestorage(t *testing.T) {
	t.Setenv("GIT_STORAGE_PROVIDER", provider.CodeStorageProvider)
	t.Setenv("GIT_STORAGE_CODE_STORAGE_NAME", "test-org")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", testECPrivateKey)

	p, err := NewProvider()
	require.NoError(t, err)
	require.IsType(t, &codestorage.Provider{}, p)
	require.Equal(t, provider.CodeStorageProvider, p.Name())
}

func TestNewProviderSupergit(t *testing.T) {
	t.Setenv("GIT_STORAGE_PROVIDER", provider.SuperGitProvider)
	t.Setenv("GIT_STORAGE_SUPERGIT_BASE_URL", "http://localhost:8080")

	p, err := NewProvider()
	require.NoError(t, err)
	require.IsType(t, &supergit.Provider{}, p)
	require.Equal(t, provider.SuperGitProvider, p.Name())
}

func TestNewProviderUnsupported(t *testing.T) {
	t.Setenv("GIT_STORAGE_PROVIDER", "bogus")
	_, err := NewProvider()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported git storage provider")
}
