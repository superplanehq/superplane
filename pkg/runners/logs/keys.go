package logs

import (
	"fmt"

	"github.com/google/uuid"
)

func ChunkKey(organizationID, taskID uuid.UUID, sequence int64) string {
	return fmt.Sprintf(
		"runner-logs/v1/%s/%s/chunks/%d.ndjson",
		organizationID,
		taskID,
		sequence,
	)
}

func FinalKey(organizationID, taskID uuid.UUID) string {
	return fmt.Sprintf("runner-logs/v1/%s/%s/logs.ndjson.gz", organizationID, taskID)
}
