package cli

import (
	"context"
	"fmt"
	"io"
	"os"

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
		domainType, _ := cmd.Flags().GetString("domain-type")
		domainID, _ := cmd.Flags().GetString("domain-id")
		if domainID == "" {
			fmt.Println("Domain ID not provided")
			os.Exit(1)
		}

		c := DefaultClient()
		_, httpResponse, err := c.SecretAPI.
			SecretsDeleteSecret(context.Background(), idOrName).
			DomainId(domainID).
			DomainType(domainType).
			Execute()

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
	Short: "Delete SuperPlane resources",
	Long:  `Delete a SuperPlane resource by ID or name.`,
}

func init() {
	RootCmd.AddCommand(deleteCmd)

	// Secret command
	deleteCmd.AddCommand(deleteSecretCmd)
	deleteSecretCmd.Flags().String("domain-type", "DOMAIN_TYPE_ORGANIZATION", "Domain to list secrets from (organization, canvas)")
	deleteSecretCmd.Flags().String("domain-id", "", "ID of the domain (organization ID, canvas ID)")
}
