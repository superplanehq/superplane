package logs

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFinalKeyUsesInstallationAndOrganizationHierarchy(t *testing.T) {
	organizationID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	taskID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	key := FinalKey("install-1", organizationID, taskID)

	assert.Equal(
		t,
		"install-1/orgs/11111111-1111-1111-1111-111111111111/runner-tasks/22222222-2222-2222-2222-222222222222/logs/v1/logs.ndjson.gz",
		key,
	)
}
