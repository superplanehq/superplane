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
			Width:        intPtr(-100),
			Height:       intPtr(500),
		})
		require.ErrorContains(t, err, "width")
	})

	t.Run("negative height", func(t *testing.T) {
		err := validateRenderPanelSpec(RenderPanelSpec{
			DashboardUID: "d1",
			PanelID:      1,
			Width:        intPtr(1000),
			Height:       intPtr(-1),
		})
		require.ErrorContains(t, err, "height")
	})

	t.Run("zero dimensions are rejected when explicitly set", func(t *testing.T) {
		err := validateRenderPanelSpec(RenderPanelSpec{
			DashboardUID: "d1",
			PanelID:      1,
			Width:        intPtr(0),
			Height:       intPtr(0),
		})
		require.ErrorContains(t, err, "width")
	})

	t.Run("omitted dimensions are valid", func(t *testing.T) {
		err := validateRenderPanelSpec(RenderPanelSpec{
			DashboardUID: "d1",
			PanelID:      1,
		})
		require.NoError(t, err)
	})
}
