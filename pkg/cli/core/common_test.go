package core_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/test/support/cli"
)

func TestResolveAppID(t *testing.T) {
	cases := []struct {
		name        string
		appID       string
		activeApp   string
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:  "explicit app id",
			appID: "app-123",
			want:  "app-123",
		},
		{
			name:  "trims explicit app id",
			appID: "  app-123  ",
			want:  "app-123",
		},
		{
			name:      "uses active app from config",
			activeApp: "active-app",
			want:      "active-app",
		},
		{
			name:      "trims active app from config",
			activeApp: "  active-app  ",
			want:      "active-app",
		},
		{
			name:      "prefers explicit app id over active app",
			appID:     "explicit-app",
			activeApp: "active-app",
			want:      "explicit-app",
		},
		{
			name:        "missing app id and active app",
			wantErr:     true,
			errContains: "app id is required",
		},
		{
			name:        "whitespace app id with empty active app",
			appID:       "   ",
			wantErr:     true,
			errContains: "app id is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := core.CommandContext{
				Config: &cli.FakeConfig{ActiveApp: tc.activeApp},
			}

			got, err := core.ResolveAppID(ctx, tc.appID)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBindAppIDFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var appID string

	core.BindAppIDFlag(cmd, &appID, "app ID")

	require.NoError(t, cmd.Flags().Set("app-id", "app-from-flag"))
	require.Equal(t, "app-from-flag", appID)

	require.NoError(t, cmd.Flags().Set("canvas-id", "app-from-legacy-flag"))
	require.Equal(t, "app-from-legacy-flag", appID)

	appIDFlag := cmd.Flags().Lookup("app-id")
	require.NotNil(t, appIDFlag)
	require.Equal(t, "app ID", appIDFlag.Usage)

	canvasIDFlag := cmd.Flags().Lookup("canvas-id")
	require.NotNil(t, canvasIDFlag)
	require.Equal(t, "app ID", canvasIDFlag.Usage)
	require.Equal(t, "use --app-id", canvasIDFlag.Deprecated)
}
