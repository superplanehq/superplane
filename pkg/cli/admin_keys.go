package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func newAdminCommand() *cobra.Command {
	admin := &cobra.Command{Use: "admin", Short: "Manage this SuperPlane installation"}
	keys := &cobra.Command{Use: "keys", Short: "Manage application encryption keys"}
	keys.AddCommand(newAdminKeysListCommand(), newAdminKeysRotateCommand(), newAdminKeysMigrateCommand(), newAdminKeysRewrapCommand())
	admin.AddCommand(keys)
	return admin
}

func newAdminKeysListCommand() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List data encryption keys",
		RunE: func(cmd *cobra.Command, _ []string) error {
			response, _, err := DefaultClient().KeysAPI.KeysListKeys(cmd.Context()).Execute()
			if err != nil {
				return fmt.Errorf("list encryption keys: %w", err)
			}
			if strings.EqualFold(OutputFormat, "json") {
				return printAdminJSON(cmd, response)
			}
			for _, key := range response.GetKeys() {
				state := ""
				if key.GetId() == response.GetActiveKeyId() {
					state = " (active)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s%s\n", key.GetId(), key.GetMasterKeyRef(), state)
			}
			return nil
		},
	}
}

func newAdminKeysRotateCommand() *cobra.Command {
	return &cobra.Command{
		Use: "rotate", Short: "Create and activate a data encryption key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			response, _, err := DefaultClient().KeysAPI.KeysRotateKey(cmd.Context()).Body(map[string]any{}).Execute()
			if err != nil {
				return fmt.Errorf("rotate data key: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), response.GetActiveKeyId())
			return nil
		},
	}
}

func newAdminKeysRewrapCommand() *cobra.Command {
	var reference string
	command := &cobra.Command{
		Use: "rewrap", Short: "Wrap data keys with the selected master key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			request := openapi_client.NewKeysRewrapKeysRequest()
			request.SetMasterKeyRef(reference)
			_, _, err := DefaultClient().KeysAPI.KeysRewrapKeys(cmd.Context()).Body(*request).Execute()
			if err != nil {
				return fmt.Errorf("rewrap data keys: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Data keys rewrapped")
			return nil
		},
	}
	command.Flags().StringVar(&reference, "master-key-ref", "", "target master key ID or KMS URI; omit to rewrap with the current key")
	return command
}

func newAdminKeysMigrateCommand() *cobra.Command {
	command := &cobra.Command{
		Use: "migrate", Short: "Re-encrypt stored values with the active data key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			response, _, err := DefaultClient().KeysAPI.KeysStartMigration(cmd.Context()).Body(map[string]any{}).Execute()
			if err != nil {
				return fmt.Errorf("start encryption migration: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), response.GetId())
			return nil
		},
	}
	command.AddCommand(&cobra.Command{
		Use: "status <id>", Short: "Show encryption migration progress", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			response, _, err := DefaultClient().KeysAPI.KeysGetMigration(cmd.Context(), args[0]).Execute()
			if err != nil {
				return fmt.Errorf("get encryption migration: %w", err)
			}
			return printAdminJSON(cmd, response)
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "retry <id>", Short: "Retry a failed encryption migration", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			response, _, err := DefaultClient().KeysAPI.KeysRetryMigration(cmd.Context(), args[0]).Body(map[string]any{}).Execute()
			if err != nil {
				return fmt.Errorf("retry encryption migration: %w", err)
			}
			return printAdminJSON(cmd, response)
		},
	})
	return command
}

func printAdminJSON(cmd *cobra.Command, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(payload))
	return err
}
