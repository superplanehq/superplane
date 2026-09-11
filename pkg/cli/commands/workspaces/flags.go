package workspaces

import "github.com/spf13/cobra"

// BindNameFlag registers --workspace and a hidden --factory alias.
func BindNameFlag(cmd *cobra.Command, dest *string) {
	cmd.Flags().StringVar(dest, "workspace", "", "workspace name or UUID (default: active workspace)")
	cmd.Flags().StringVar(dest, "factory", "", "workspace name or UUID (default: active workspace)")
	_ = cmd.Flags().MarkDeprecated("factory", "use --workspace")
	_ = cmd.Flags().MarkHidden("factory")
}
