package organizations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_initialSyncRunAt_gcpWIFUsesDelay(t *testing.T) {
	sak := &models.Integration{
		AppName: "gcp",
		Properties: []core.IntegrationPropertyDefinition{
			{Name: "connectionMethod", Value: "serviceAccountKey"},
		},
	}
	wif := &models.Integration{
		AppName: "gcp",
		Properties: []core.IntegrationPropertyDefinition{
			{Name: "connectionMethod", Value: "workloadIdentityFederation"},
		},
	}

	t0 := time.Now()
	require.False(t, initialSyncRunAt(sak).After(t0.Add(2*time.Second)), "SAK path should sync immediately")

	runWIF := initialSyncRunAt(wif)
	assert.GreaterOrEqual(t, runWIF.Sub(t0), gcpWIFInitialSyncDelay-time.Second)
	assert.LessOrEqual(t, runWIF.Sub(t0), gcpWIFInitialSyncDelay+time.Second)
	assert.Equal(t, postSetupSyncDescriptionGCPWIFDelayed, initialSyncStateDescription(wif))
	assert.Equal(t, postSetupSyncDescription, initialSyncStateDescription(sak))
}
