package organizations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/models"
)

func Test_initialSyncRunAt_gcpUsesDelay(t *testing.T) {
	gcp := &models.Integration{AppName: "gcp"}
	other := &models.Integration{AppName: "github"}

	t0 := time.Now()

	runGCP := initialSyncRunAt(gcp)
	assert.GreaterOrEqual(t, runGCP.Sub(t0), gcpWIFInitialSyncDelay-time.Second)
	assert.LessOrEqual(t, runGCP.Sub(t0), gcpWIFInitialSyncDelay+time.Second)
	assert.Equal(t, postSetupSyncDescriptionGCPWIFDelayed, initialSyncStateDescription(gcp))

	runOther := initialSyncRunAt(other)
	assert.LessOrEqual(t, runOther.Sub(t0), 2*time.Second, "non-GCP integrations sync immediately")
	assert.Equal(t, postSetupSyncDescription, initialSyncStateDescription(other))
}
