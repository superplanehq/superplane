package eventdistributer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__RunStateToWsEvent(t *testing.T) {
	assert.Equal(t, RunStartedEvent, runStateToWsEvent(models.CanvasRunStateStarted))
	assert.Equal(t, RunFinishedEvent, runStateToWsEvent(models.CanvasRunStateFinished))
	assert.Empty(t, runStateToWsEvent("unknown"))
}
