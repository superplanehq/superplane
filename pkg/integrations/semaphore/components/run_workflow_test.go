package components

import (
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

func (p *testOIDCProvider) Validate(tokenString string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (p *testOIDCProvider) PublicJWKs() []oidc.PublicJWK {
	return nil
}

func TestRunWorkflowBuildParametersIncludesOIDCToken(t *testing.T) {
	t.Parallel()

	runWorkflow := RunWorkflow{}
	executionID := uuid.New()

	parameters := runWorkflow.buildParameters(core.ExecutionContext{
		ID:             executionID,
		WorkflowID:     "canvas-123",
		OrganizationID: "org-123",
		NodeID:         "deploy",
		OIDC:           &testOIDCProvider{},
	}, RunWorkflowSpec{
		PipelineFile: ".semaphore/deploy.yml",
		Ref:          "refs/heads/main",
		CommitSha:    "abc123",
		Parameters: []Parameter{
			{Name: "FOO", Value: "bar"},
		},
	}, RunWorkflowNodeMetadata{
		Project: &Project{ID: "project-123"},
	})

	require.Equal(t, executionID.String(), parameters["SUPERPLANE_EXECUTION_ID"])
	require.Equal(t, "canvas-123", parameters["SUPERPLANE_CANVAS_ID"])
	require.Equal(t, "bar", parameters["FOO"])
	require.Equal(t, "test-token", parameters["SUPERPLANE_OIDC_TOKEN"])
}
