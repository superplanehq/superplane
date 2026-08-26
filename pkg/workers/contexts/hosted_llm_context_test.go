package contexts

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__HostedLLMContext__ReservesCreditOutsideExecutorTransaction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	first := pendingHostedExecution(t, r)
	second := pendingHostedExecution(t, r)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		hosted := NewHostedLLMContext(tx, nil, r.Organization.ID, first.ID, nil)
		require.NoError(t, hosted.AssertCreditAvailable())

		done := make(chan error, 2)
		go func() {
			bps := 0
			done <- models.UpsertOrganizationLLMMarkup(database.Conn(), r.Organization.ID, &bps)
		}()
		go func() {
			other := NewHostedLLMContext(database.Conn(), nil, r.Organization.ID, second.ID, nil)
			done <- other.AssertCreditAvailable()
		}()

		var markupErr, secondErr error
		for i := 0; i < 2; i++ {
			select {
			case err := <-done:
				if errors.Is(err, models.ErrHostedRunInFlight) {
					secondErr = err
					continue
				}
				markupErr = err
			case <-time.After(2 * time.Second):
				return errors.New("blocked on organization LLM settings row lock")
			}
		}
		if markupErr != nil {
			return markupErr
		}
		if secondErr == nil {
			return errors.New("expected in-flight hosted run error")
		}
		return errors.New("executor rolled back")
	})
	require.Error(t, err)
	require.EqualError(t, err, "executor rolled back")

	var count int64
	require.NoError(t, database.Conn().Model(&models.OrganizationLLMCreditHold{}).
		Where("node_execution_id = ?", first.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func Test__HostedLLMContext__AssertModelSelectable(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	hosted := NewHostedLLMContext(db, nil, r.Organization.ID, uuid.New(), nil)
	err := hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")

	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{
		"gpt-4.1",
	})
	require.NoError(t, err)

	require.NoError(t, hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "gpt-4.1"))
	err = hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "gpt-4o")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selected-model list")
}

func pendingHostedExecution(t *testing.T, r *support.ResourceRegistry) *models.CanvasNodeExecution {
	t.Helper()
	nodeID := support.RandomName("hosted")
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: nodeID, Type: models.NodeTypeComponent},
	}, []models.Edge{})
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	return support.CreateCanvasNodeExecution(t, canvas.ID, nodeID, rootEvent.ID, rootEvent.ID)
}
