package runner

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
)

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
