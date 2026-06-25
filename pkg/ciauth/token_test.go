package ciauth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/ciauth"
	"github.com/superplanehq/superplane/pkg/oidc"
)

func TestValidateToken(t *testing.T) {
	t.Parallel()

	provider, err := oidc.NewProviderFromKeyDir("https://superplane.test", "../../test/fixtures/oidc-keys")
	require.NoError(t, err)

	executionID := uuid.NewString()
	token, err := provider.Sign(
		fmt.Sprintf("execution:%s", executionID),
		time.Hour,
		ciauth.ExecutionTokenAudience,
		map[string]any{
			ciauth.ClaimOrgID:       uuid.NewString(),
			ciauth.ClaimCanvasID:    uuid.NewString(),
			ciauth.ClaimNodeID:      "deploy",
			ciauth.ClaimExecutionID: executionID,
			ciauth.ClaimComponent:   "semaphore.runWorkflow",
			"pipeline_file":         ".semaphore/deploy.yml",
		},
	)
	require.NoError(t, err)

	claims, err := ciauth.ValidateToken(provider, token)
	require.NoError(t, err)
	require.Equal(t, "deploy", claims.NodeID)
	require.Equal(t, ".semaphore/deploy.yml", claims.Additional["pipeline_file"])
}

func TestExecutionTokenExpectedMatches(t *testing.T) {
	t.Parallel()

	claims := ciauth.ExecutionTokenClaims{
		CanvasID: "canvas-1",
		Additional: map[string]string{
			"pipeline_file": ".semaphore/deploy.yml",
		},
	}

	err := (ciauth.ExecutionTokenExpected{
		CanvasID: "canvas-1",
		Additional: map[string]string{
			"pipeline_file": ".semaphore/deploy.yml",
		},
	}).Matches(claims)
	require.NoError(t, err)

	err = (ciauth.ExecutionTokenExpected{
		Additional: map[string]string{
			"pipeline_file": ".semaphore/other.yml",
		},
	}).Matches(claims)
	require.Error(t, err)
}
