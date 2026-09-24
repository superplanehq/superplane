package runner

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/jwt"
)

func TestAttachArtifactUploadEnvRequiresEnabledWorkOrderRun(t *testing.T) {
	t.Parallel()

	existing := []BrokerEnvironmentVariable{
		{Name: "EXISTING", Value: "value"},
		{Name: EnvSuperplaneArtifactToken, Value: "external-token"},
	}
	want := []BrokerEnvironmentVariable{{Name: "EXISTING", Value: "value"}}
	ctx := core.ExecutionContext{RunID: uuid.New(), ID: uuid.New()}
	assert.Equal(t, want, AttachArtifactUploadEnv(ctx, existing, 60, false))
	assert.Equal(t, want, AttachArtifactUploadEnv(core.ExecutionContext{}, existing, 60, true))
}

func TestAttachArtifactUploadEnvSkipsPlanningSessions(t *testing.T) {
	t.Parallel()

	environment := []BrokerEnvironmentVariable{
		{Name: EnvSuperplanePlanningID, Value: uuid.NewString()},
		{Name: EnvSuperplaneArtifactToken, Value: "external-token"},
	}
	want := []BrokerEnvironmentVariable{{Name: EnvSuperplanePlanningID, Value: environment[0].Value}}
	ctx := core.ExecutionContext{RunID: uuid.New(), ID: uuid.New()}
	assert.Equal(t, want, AttachArtifactUploadEnv(ctx, environment, 60, true))
}

func TestArtifactUploadTokenRoundTrip(t *testing.T) {
	signer := jwt.NewSigner("secret")
	want := ArtifactUploadScope{
		OrganizationID:  uuid.New(),
		FactoryID:       uuid.New(),
		WorkOrderID:     uuid.New(),
		CanvasID:        uuid.New(),
		CanvasRunID:     uuid.New(),
		NodeExecutionID: uuid.New(),
		NodeID:          "implement",
	}
	token, err := MintArtifactUploadToken(signer, want, time.Hour)
	require.NoError(t, err)

	got, err := ParseArtifactUploadToken(signer, token)
	require.NoError(t, err)
	assert.Equal(t, want, *got)
}

func TestArtifactUploadTokenRejectsPlanningPurpose(t *testing.T) {
	signer := jwt.NewSigner("secret")
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{"purpose": PlanningSessionTokenPurpose})
	require.NoError(t, err)

	_, err = ParseArtifactUploadToken(signer, token)
	assert.ErrorContains(t, err, "purpose")
}

func TestArtifactUploadTokenRejectsExpiredToken(t *testing.T) {
	signer := jwt.NewSigner("secret")
	token, err := signer.GenerateWithClaims(-time.Second, map[string]string{"purpose": ArtifactUploadTokenPurpose})
	require.NoError(t, err)

	_, err = ParseArtifactUploadToken(signer, token)
	assert.Error(t, err)
}

func TestAppendTaskArtifactMCPRequiresArtifactToken(t *testing.T) {
	files := []BrokerTaskFile{{Path: "run.js"}}
	assert.Equal(t, files, AppendTaskArtifactMCP(nil, files))

	environment := []BrokerEnvironmentVariable{{Name: EnvSuperplaneArtifactToken, Value: "token"}}
	withMCP := AppendTaskArtifactMCP(environment, files)
	require.Len(t, withMCP, 2)
	assert.Equal(t, "task_artifact_mcp.js", withMCP[1].Path)
}
