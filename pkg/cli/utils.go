package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func getOneOrAnotherFlag(cmd *cobra.Command, flag1, flag2 string, required bool) string {
	flag1Value, _ := cmd.Flags().GetString(flag1)
	flag2Value, _ := cmd.Flags().GetString(flag2)

	if flag1Value != "" && flag2Value != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --%s and --%s\n", flag1, flag2)
		os.Exit(1)
	}

	if flag1Value != "" {
		return flag1Value
	}

	if flag2Value != "" {
		return flag2Value
	}

	if required {
		fmt.Fprintf(os.Stderr, "Error: must specify either --%s or --%s\n", flag1, flag2)
		os.Exit(1)
	}

	return ""
}

func getDomainOrExit(cmd *cobra.Command) (string, string) {
	organizationId, _ := cmd.Flags().GetString("organization-id")
	canvasIdOrName := getOneOrAnotherFlag(cmd, "canvas-id", "canvas-name", false)

	if organizationId != "" {
		return string(openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION), organizationId
	}

	if canvasIdOrName != "" {
		return string(openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_CANVAS), canvasIdOrName
	}

	fmt.Println("Either organization-id or canvas-id / canvas-name flags must be provided")
	os.Exit(1)
	return "", ""
}
