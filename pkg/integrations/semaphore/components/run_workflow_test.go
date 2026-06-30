package components

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/oidc"
)

type testOIDCProvider struct{}

func (p *testOIDCProvider) Sign(subject string, duration time.Duration, audience string, additionalClaims map[string]any) (string, error) {
	return "test-token", nil
}

func (p *testOIDCProvider) PublicJWKs() []oidc.PublicJWK {
	return nil
}

type failingOIDCProvider struct{}

func (p *failingOIDCProvider) Sign(subject string, duration time.Duration, audience string, additionalClaims map[string]any) (string, error) {
	return "", fmt.Errorf("signing unavailable")
}

func (p *failingOIDCProvider) PublicJWKs() []oidc.PublicJWK {
	return nil
}

func TestRunWorkflowBuildParametersIncludesOIDCToken(t *testing.T) {
	t.Parallel()

	runWorkflow := RunWorkflow{}
	executionID := uuid.New()

	parameters, err := runWorkflow.buildParameters(core.ExecutionContext{
		ID:             executionID,
		WorkflowID:     "canvas-123",
		OrganizationID: "org-123",
		NodeID:         "deploy",
		OIDC:           &testOIDCProvider{},
	}, RunWorkflowSpec{
		InjectOidcToken: true,
		PipelineFile:    ".semaphore/deploy.yml",
		Ref:             "refs/heads/main",
		CommitSha:       "abc123",
		Parameters: []Parameter{
			{Name: "FOO", Value: "bar"},
		},
	}, RunWorkflowNodeMetadata{
		Project: &Project{ID: "project-123"},
	})
	require.NoError(t, err)

	require.Equal(t, executionID.String(), parameters["SUPERPLANE_EXECUTION_ID"])
	require.Equal(t, "canvas-123", parameters["SUPERPLANE_CANVAS_ID"])
	require.Equal(t, "bar", parameters["FOO"])
	require.Equal(t, "test-token", parameters["SUPERPLANE_OIDC_TOKEN"])
}

func TestRunWorkflowBuildParametersSkipsOIDCWhenDisabled(t *testing.T) {
	t.Parallel()

	runWorkflow := RunWorkflow{}
	executionID := uuid.New()

	parameters, err := runWorkflow.buildParameters(core.ExecutionContext{
		ID:   executionID,
		OIDC: &testOIDCProvider{},
	}, RunWorkflowSpec{}, RunWorkflowNodeMetadata{})
	require.NoError(t, err)
	require.Equal(t, executionID.String(), parameters["SUPERPLANE_EXECUTION_ID"])
	require.NotContains(t, parameters, "SUPERPLANE_OIDC_TOKEN")
}

func TestRunWorkflowBuildParametersFailsWhenOIDCUnavailable(t *testing.T) {
	t.Parallel()

	runWorkflow := RunWorkflow{}

	_, err := runWorkflow.buildParameters(core.ExecutionContext{
		ID: uuid.New(),
	}, RunWorkflowSpec{
		InjectOidcToken: true,
	}, RunWorkflowNodeMetadata{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "OIDC provider is not configured")
}

func TestRunWorkflowBuildParametersFailsWhenSigningFails(t *testing.T) {
	t.Parallel()

	runWorkflow := RunWorkflow{}

	_, err := runWorkflow.buildParameters(core.ExecutionContext{
		ID:             uuid.New(),
		WorkflowID:     "canvas-123",
		OrganizationID: "org-123",
		NodeID:         "deploy",
		OIDC:           &failingOIDCProvider{},
	}, RunWorkflowSpec{
		InjectOidcToken: true,
		PipelineFile:    ".semaphore/deploy.yml",
	}, RunWorkflowNodeMetadata{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to sign OIDC execution token")
}
