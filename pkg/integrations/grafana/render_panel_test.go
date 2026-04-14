package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test__validateRenderPanelSpec__rejectsNegativeDimensions(t *testing.T) {
	t.Run("negative width", func(t *testing.T) {
		err := validateRenderPanelSpec(RenderPanelSpec{
			DashboardUID: "d1",
			PanelID:      1,
			Width:        -100,
			Height:       500,
		})
		require.ErrorContains(t, err, "width")
	})

	t.Run("negative height", func(t *testing.T) {
		err := validateRenderPanelSpec(RenderPanelSpec{
			DashboardUID: "d1",
			PanelID:      1,
			Width:        1000,
			Height:       -1,
		})
		require.ErrorContains(t, err, "height")
	})

	t.Run("zero dimensions are allowed", func(t *testing.T) {
		err := validateRenderPanelSpec(RenderPanelSpec{
			DashboardUID: "d1",
			PanelID:      1,
			Width:        0,
			Height:       0,
		})
		require.NoError(t, err)
	})
}
