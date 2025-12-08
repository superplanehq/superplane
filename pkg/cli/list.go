package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var listSecretsCmd = &cobra.Command{
	Use:     "secrets",
	Short:   "List secrets for an organization or canvas",
	Long:    `Retrieve a list of all secrets for the specified organization or canvas`,
	Aliases: []string{"secret"},
	Args:    cobra.ExactArgs(0),

	Run: func(cmd *cobra.Command, args []string) {
		c := DefaultClient()
		domainType, domainID := getDomainOrExit(c, cmd)

		response, httpResponse, err := c.SecretAPI.
			SecretsListSecrets(context.Background()).
			DomainId(domainID).
			DomainType(domainType).
			Execute()

		if err != nil {
			b, _ := io.ReadAll(httpResponse.Body)
			fmt.Printf("%s\n", string(b))
			os.Exit(1)
		}

		if len(response.Secrets) == 0 {
			fmt.Println("No secrets found.")
			return
		}

		fmt.Printf("Found %d secrets:\n\n", len(response.Secrets))
		for i, secret := range response.Secrets {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *secret.GetMetadata().Name, *secret.GetMetadata().Id)
			fmt.Printf("   Domain Type: %s\n", *secret.GetMetadata().DomainType)
			fmt.Printf("   Domain ID: %s\n", *secret.GetMetadata().DomainId)
			fmt.Printf("   Provider: %s\n", string(*secret.GetSpec().Provider))
			fmt.Printf("   Created at: %s\n", *secret.GetMetadata().CreatedAt)

			if secret.GetSpec().Local != nil && secret.GetSpec().Local.Data != nil {
				fmt.Println("   Values:")
				for k, v := range *secret.GetSpec().Local.Data {
					fmt.Printf("     %s = %s\n", k, v)
				}
			}

			if i < len(response.Secrets)-1 {
				fmt.Println()
			}
		}
	},
}

var listIntegrationsCmd = &cobra.Command{
	Use:     "integrations",
	Short:   "List all integrations for an organization or canvas",
	Long:    `Retrieve a list of integrations for the specified organization or canvas`,
	Aliases: []string{"integration"},
	Args:    cobra.ExactArgs(0),

	Run: func(cmd *cobra.Command, args []string) {
		c := DefaultClient()
		domainType, domainID := getDomainOrExit(c, cmd)
		response, httpResponse, err := c.IntegrationAPI.
			IntegrationsListIntegrations(context.Background()).
			DomainId(domainID).
			DomainType(domainType).
			Execute()

		if err != nil {
			b, _ := io.ReadAll(httpResponse.Body)
			fmt.Printf("%s\n", string(b))
			os.Exit(1)
		}

		if len(response.Integrations) == 0 {
			fmt.Println("No integrations found.")
			return
		}

		fmt.Printf("Found %d integrations:\n\n", len(response.Integrations))
		for i, integration := range response.Integrations {
			metadata := integration.GetMetadata()
			spec := integration.GetSpec()
			fmt.Printf("%d. %s (ID: %s)\n", i+1, *metadata.Name, *metadata.Id)
			fmt.Printf("   Domain Type: %s\n", *metadata.DomainType)
			fmt.Printf("   Domain ID: %s\n", *metadata.DomainId)
			fmt.Printf("   Type: %s\n", *spec.Type)
			fmt.Printf("   URL: %s\n", spec.GetUrl())

			if i < len(response.Integrations)-1 {
				fmt.Println()
			}
		}
	},
}

// Root list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List Superplane resources",
	Long:    `List multiple Superplane resources.`,
	Aliases: []string{"ls"},
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listSecretsCmd)
	listCmd.AddCommand(listIntegrationsCmd)
}
