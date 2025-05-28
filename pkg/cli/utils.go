package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func getOneOrAnotherFlag(cmd *cobra.Command, flag1, flag2 string) string {
	flag1Value, _ := cmd.Flags().GetString(flag1)
	flag2Value, _ := cmd.Flags().GetString(flag2)

	if flag1Value != "" && flag2Value != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --%s and --%s\n", flag1, flag2)
		os.Exit(1)
	} else if flag1Value != "" {
		return flag1Value
	} else if flag2Value != "" {
		return flag2Value
	} else {
		fmt.Fprintf(os.Stderr, "Error: must specify either --%s or --%s\n", flag1, flag2)
		os.Exit(1)
	}

	return ""
}
