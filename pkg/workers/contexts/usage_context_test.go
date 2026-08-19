package contexts

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__UsageContext__KeepsSpendWhenExecutorTransactionRollsBack(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	err = database.Conn().Transaction(func(_ *gorm.DB) error {
		usage := NewUsageContext(r.Organization.ID, nodeExecution)
		require.NoError(t, usage.Record(core.UsageRecord{
			Provider:     models.UsageProviderAnthropic,
			Model:        "claude-sonnet-4-6",
			InputTokens:  100,
			OutputTokens: 20,
			TotalTokens:  120,
		}))
		return errors.New("emit failed")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, database.Conn().Model(&models.LLMUsageEvent{}).
		Where("node_execution_id = ?", nodeExecution.ID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
