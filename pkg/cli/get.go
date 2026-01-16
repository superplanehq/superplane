package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var getSecretCmd = &cobra.Command{
	Use:     "secret [ID_OR_NAME]",
	Short:   "Get secret details",
	Long:    `Get details about a specific secret`,
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
		response, httpResponse, err := c.SecretAPI.
			SecretsDescribeSecret(context.Background(), idOrName).
			DomainId(domainID).
			DomainType(domainType).
			Execute()

		if err != nil {
			b, _ := io.ReadAll(httpResponse.Body)
			fmt.Printf("%s\n", string(b))
			os.Exit(1)
		}

		out, err := yaml.Marshal(response.Secret)
		Check(err)
		fmt.Printf("%s", string(out))
	},
}

// Root describe command
var getCmd = &cobra.Command{
	Use:     "get",
	Short:   "Show details of SuperPlane resources",
	Long:    `Get detailed information about SuperPlane resources.`,
	Aliases: []string{"desc", "get"},
}

func init() {
	RootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getSecretCmd)
}
