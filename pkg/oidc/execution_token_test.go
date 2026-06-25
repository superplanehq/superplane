package oidc_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/oidc"
)

func TestSignAndValidateExecutionToken(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(t, "https://superplane.test")

	input := oidc.ExecutionTokenInput{
		OrganizationID: uuid.NewString(),
		CanvasID:       uuid.NewString(),
		NodeID:         "deploy",
		ExecutionID:    uuid.NewString(),
		Component:      "semaphore.runWorkflow",
		ProjectID:      "project-123",
		PipelineFile:   ".semaphore/deploy.yml",
		Ref:            "refs/heads/main",
		CommitSha:      "abc123",
	}

	token, err := oidc.SignExecutionToken(provider, input)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := oidc.ValidateExecutionToken(provider, token)
	require.NoError(t, err)
	require.Equal(t, input.OrganizationID, claims.OrgID)
	require.Equal(t, input.CanvasID, claims.CanvasID)
	require.Equal(t, input.NodeID, claims.NodeID)
	require.Equal(t, input.ExecutionID, claims.ExecutionID)
	require.Equal(t, input.Component, claims.Component)
	require.Equal(t, input.ProjectID, claims.ProjectID)
	require.Equal(t, input.PipelineFile, claims.PipelineFile)
	require.Equal(t, input.Ref, claims.Ref)
	require.Equal(t, input.CommitSha, claims.CommitSha)
}

func TestExecutionTokenExpectedMatches(t *testing.T) {
	t.Parallel()

	claims := oidc.ExecutionTokenClaims{
		CanvasID:     "canvas-1",
		PipelineFile: ".semaphore/deploy.yml",
	}

	err := (oidc.ExecutionTokenExpected{
		CanvasID:     "canvas-1",
		PipelineFile: ".semaphore/deploy.yml",
	}).Matches(claims)
	require.NoError(t, err)

	err = (oidc.ExecutionTokenExpected{
		PipelineFile: ".semaphore/other.yml",
	}).Matches(claims)
	require.Error(t, err)
}

func TestValidateExecutionTokenRejectsWrongAudience(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(t, "https://superplane.test")
	token, err := provider.Sign("execution:test", oidc.ExecutionTokenDuration, "other-audience", map[string]any{
		oidc.ClaimOrgID:       uuid.NewString(),
		oidc.ClaimCanvasID:    uuid.NewString(),
		oidc.ClaimNodeID:      "node",
		oidc.ClaimExecutionID: uuid.NewString(),
	})
	require.NoError(t, err)

	_, err = oidc.ValidateExecutionToken(provider, token)
	require.Error(t, err)
}

func newTestProvider(t *testing.T, issuer string) oidc.Provider {
	t.Helper()

	dir := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPath := filepath.Join(dir, "test.pem")
	require.NoError(t, os.WriteFile(keyPath, pemEncodeKey(key), 0o600))

	provider, err := oidc.NewProviderFromKeyDir(issuer, dir)
	require.NoError(t, err)
	return provider
}

func pemEncodeKey(key *rsa.PrivateKey) []byte {
	der := x509.MarshalPKCS1PrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	})
}
