package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func TestSerializeArtifactIncludesPlanningKey(t *testing.T) {
	key := models.PlanningSpecArtifactKey + ":" + uuid.NewString()
	artifact := &models.FactoryWorkOrderArtifact{
		ID:        uuid.New(),
		Type:      models.FactoryWorkOrderArtifactTypeMarkdown,
		Data:      datatypes.JSON(`{"name":"plan.md","title":"plan.md","body":"# Plan"}`),
		Key:       &key,
		CreatedAt: time.Now(),
	}

	serialized, err := serializeArtifact(artifact)
	require.NoError(t, err)
	assert.Equal(t, key, serialized.GetKey())
	assert.Equal(t, "# Plan", serialized.GetData().AsMap()["body"])
}
