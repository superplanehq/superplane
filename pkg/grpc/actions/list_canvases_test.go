package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListCanvases(t *testing.T) {
	r := support.Setup(t)

	res, err := ListCanvases(context.Background(), &protos.ListCanvasesRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Canvases, 1)
	assert.Equal(t, r.Canvas.ID.String(), res.Canvases[0].Id)
	assert.Equal(t, r.Canvas.Name, res.Canvases[0].Name)
	assert.Equal(t, r.Canvas.CreatedBy.String(), res.Canvases[0].CreatedBy)
	assert.NotNil(t, res.Canvases[0].CreatedAt)
}
