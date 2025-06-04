package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var deleteSecretCmd = &cobra.Command{
	Use:     "secret [ID_OR_NAME]",
	Short:   "Delete a canvas secret",
	Long:    `Delete a canvas secret by ID or name.`,
	Aliases: []string{"secrets"},
	Args:    cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		idOrName := args[0]
		canvasIDOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name")

		c := DefaultClient()
		_, httpResponse, err := c.SecretAPI.SuperplaneDeleteSecret(
			context.Background(),
			canvasIDOrName,
			idOrName,
		).RequesterId(uuid.NewString()).Execute()

		if err != nil {
			b, _ := io.ReadAll(httpResponse.Body)
			fmt.Printf("%s\n", string(b))
			os.Exit(1)
		}

		fmt.Printf("Secret %s deleted successfully\n", idOrName)
	},
}

// Root describe command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete Superplane resources",
	Long:  `Delete a Superplane resource by ID or name.`,
}

func init() {
	RootCmd.AddCommand(deleteCmd)

	// Secret command
	deleteCmd.AddCommand(deleteSecretCmd)
	deleteSecretCmd.Flags().String("canvas-id", "", "ID of the canvas (alternative to --canvas-name)")
	deleteSecretCmd.Flags().String("canvas-name", "", "Name of the canvas (alternative to --canvas-id)")
}
