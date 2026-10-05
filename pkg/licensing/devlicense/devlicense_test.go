package devlicense_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/devlicense"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

type memorySource struct {
	raw []byte
}

func (m *memorySource) Kind() licensing.SourceKind {
	return licensing.SourceDatabase
}

func (m *memorySource) Read(context.Context) ([]byte, error) {
	if m.raw == nil {
		return nil, licensing.ErrNotInstalled
	}

	return m.raw, nil
}

func (m *memorySource) Write(_ context.Context, raw []byte, _ uuid.UUID) error {
	m.raw = raw
	return nil
}

func (m *memorySource) Clear(context.Context) error {
	m.raw = nil
	return nil
}

func wrappedService(t *testing.T, keys *licensing.KeySet, base licensing.Source) *licensing.Service {
	t.Helper()
	keys, source, err := devlicense.Wrap(keys, base)
	require.NoError(t, err)

	service := licensing.NewService(licensing.NewVerifier(keys), source)
	require.NoError(t, service.Refresh(context.Background()))
	return service
}

func TestEnabledRequiresExplicitTrue(t *testing.T) {
	t.Setenv(devlicense.EnterpriseEnv, "")
	assert.False(t, devlicense.Enabled())

	t.Setenv(devlicense.EnterpriseEnv, "1")
	assert.False(t, devlicense.Enabled())

	t.Setenv(devlicense.EnterpriseEnv, "true")
	assert.True(t, devlicense.Enabled())
}

func TestWrapGrantsEveryFeatureWithoutInstalledLicense(t *testing.T) {
	keys, err := licensing.ProductionKeySet()
	require.NoError(t, err)

	service := wrappedService(t, keys, &memorySource{})

	status := service.Status()
	assert.Equal(t, licensing.EditionEnterprise, status.Edition)
	assert.True(t, status.Writable, "administrators can still install a license")
	for _, feature := range licensing.RecognizedFeatures() {
		assert.True(t, service.IsEntitled(feature), feature)
	}
}

func TestWrapPrefersInstalledLicense(t *testing.T) {
	issuer := licensingtest.NewIssuer("local-issuer")
	installed := issuer.License(licensing.FeatureGroups)

	service := wrappedService(t, licensingtest.KeySet(issuer), &memorySource{raw: installed})

	assert.True(t, service.IsEntitled(licensing.FeatureGroups))
	assert.False(t, service.IsEntitled(licensing.FeatureCustomRoles))
}

func TestWrapKeepsLicenseFileAuthoritative(t *testing.T) {
	path := filepath.Join(t.TempDir(), "license.jws")
	require.NoError(t, os.WriteFile(path, []byte("not-a-license"), 0o600))
	file := licensing.NewFileSource(path)

	keys, err := licensing.ProductionKeySet()
	require.NoError(t, err)
	wrappedKeys, source, err := devlicense.Wrap(keys, file)
	require.NoError(t, err)
	assert.Same(t, keys, wrappedKeys)
	assert.Same(t, licensing.Source(file), source)
}

func TestReleaseServerDoesNotContainDevelopmentLicense(t *testing.T) {
	output, err := exec.Command("go", "list", "-deps", "-buildvcs=false", "github.com/superplanehq/superplane/cmd/server").CombinedOutput()
	require.NoError(t, err, string(output))

	dependencies := strings.Fields(string(output))
	assert.NotContains(t, dependencies, "github.com/superplanehq/superplane/pkg/licensing/devlicense")
	assert.NotContains(t, dependencies, "github.com/superplanehq/superplane/pkg/licensing/licensingtest")
}

func TestProductionKeysRejectDevelopmentLicense(t *testing.T) {
	keys, err := licensing.ProductionKeySet()
	require.NoError(t, err)
	_, source, err := devlicense.Wrap(keys, &memorySource{})
	require.NoError(t, err)

	raw, err := source.Read(context.Background())
	require.NoError(t, err)

	_, err = licensing.NewVerifier(keys).Verify(raw)
	require.Error(t, err)
}
