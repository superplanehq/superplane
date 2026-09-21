package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__TrimWorkOrderEventsPage(t *testing.T) {
	events := func(n int) []models.FactoryWorkOrderEvent {
		result := make([]models.FactoryWorkOrderEvent, n)
		return result
	}

	t.Run("full page with no extra row is the last page", func(t *testing.T) {
		page, hasNext := trimWorkOrderEventsPage(events(2), 2)
		assert.Len(t, page, 2)
		assert.False(t, hasNext)
	})

	t.Run("extra row means more events follow and is not returned", func(t *testing.T) {
		page, hasNext := trimWorkOrderEventsPage(events(3), 2)
		assert.Len(t, page, 2)
		assert.True(t, hasNext)
	})

	t.Run("partial page", func(t *testing.T) {
		page, hasNext := trimWorkOrderEventsPage(events(1), 2)
		assert.Len(t, page, 1)
		assert.False(t, hasNext)
	})

	t.Run("empty page", func(t *testing.T) {
		page, hasNext := trimWorkOrderEventsPage(events(0), 2)
		assert.Empty(t, page)
		assert.False(t, hasNext)
	})
}
