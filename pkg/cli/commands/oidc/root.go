package oidc

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "oidc",
		Short: "Verify SuperPlane OIDC execution tokens",
	}

	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a SuperPlane OIDC execution token",
		Args:  cobra.NoArgs,
	}

	var token string
	var apiURL string
	var expectedClaims []string

	verifyCmd.Flags().StringVar(&token, "token", "", "OIDC token to verify (default: SUPERPLANE_OIDC_TOKEN env var)")
	verifyCmd.Flags().StringVar(&apiURL, "url", "", "SuperPlane API URL (default: configured context URL, or https://app.superplane.com)")
	verifyCmd.Flags().StringArrayVar(&expectedClaims, "claim", nil, "expected claim key=value (repeatable)")

	core.Bind(verifyCmd, &verifyCommand{
		token:          &token,
		apiURL:         &apiURL,
		expectedClaims: &expectedClaims,
	}, options)

	root.AddCommand(verifyCmd)

	return root
}
