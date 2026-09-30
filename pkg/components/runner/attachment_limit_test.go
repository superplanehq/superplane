package runner

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAttachmentLimitMatchesSharedMaxFileBytes(t *testing.T) {
	t.Parallel()

	runnerLimit := attachmentLimitMiB(t, AttachmentLimitFile().Content, `const MAX_ATTACHMENT_BYTES = (\d+) \* 1024 \* 1024;`)
	assert.Equal(t, int64(models.MaxFileBytes), runnerLimit<<20)

	uiSource, err := os.ReadFile("../../../web_src/src/lib/workOrderFiles.ts")
	require.NoError(t, err)
	uiLimit := attachmentLimitMiB(t, string(uiSource), `export const MAX_WORK_ORDER_FILE_BYTES = (\d+) \* 1024 \* 1024;`)
	assert.Equal(t, int64(models.MaxFileBytes), uiLimit<<20)
}

func TestPlanningFilesShipOneAttachmentLimit(t *testing.T) {
	t.Parallel()

	files := AppendPlanningSessionMCPFiles(AppendAttachmentLimitFile(nil))
	var copies int
	for _, file := range files {
		if file.Path == "attachment_limit.js" {
			copies++
			assert.Equal(t, AttachmentLimitFile().Content, file.Content)
		}
	}
	assert.Equal(t, 1, copies)
	assert.Contains(t, FollowUpLoopFile().Content, `require("./attachment_limit")`)
	assert.Contains(t, PlanningSessionMCPScript(), `require("./attachment_limit")`)
}

func attachmentLimitMiB(t *testing.T, source, pattern string) int64 {
	t.Helper()
	match := regexp.MustCompile(pattern).FindStringSubmatch(source)
	require.Len(t, match, 2)
	value, err := strconv.ParseInt(match[1], 10, 64)
	require.NoError(t, err)
	return value
}
