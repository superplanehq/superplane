package cli

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	canvases "github.com/superplanehq/superplane/pkg/cli/commands/canvases"
	config "github.com/superplanehq/superplane/pkg/cli/commands/config"
	index "github.com/superplanehq/superplane/pkg/cli/commands/index"
	integrations "github.com/superplanehq/superplane/pkg/cli/commands/integrations"
	queue "github.com/superplanehq/superplane/pkg/cli/commands/queue"
	secrets "github.com/superplanehq/superplane/pkg/cli/commands/secrets"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

const (
	DefaultAPIURL     = "http://localhost:8000"
	ConfigKeyAPIURL   = "api_url"
	ConfigKeyAPIToken = "api_token"
	ConfigKeyFormat   = "output_format"
)

var cfgFile string
var Verbose bool
var OutputFormat string

var RootCmd = &cobra.Command{
	Use:   "superplane",
	Short: "SuperPlane command line interface",
	Long:  `SuperPlane CLI - Command line interface for the SuperPlane API`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !Verbose {
			log.SetOutput(io.Discard)
		}
	},
}

func init() {
	viper.SetDefault(ConfigKeyAPIURL, DefaultAPIURL)
	viper.SetDefault(ConfigKeyFormat, "text")
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.superplane.yaml)")
	RootCmd.PersistentFlags().StringVarP(&OutputFormat, "output", "o", "", "output format: text|json|yaml (overrides config output_format)")

	options := defaultBindOptions()
	RootCmd.AddCommand(canvases.NewCommand(options))
	RootCmd.AddCommand(index.NewCommand(options))
	RootCmd.AddCommand(integrations.NewCommand(options))
	RootCmd.AddCommand(queue.NewCommand(options))
	RootCmd.AddCommand(secrets.NewCommand(options))
	RootCmd.AddCommand(config.NewCommand(options))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		CheckWithMessage(err, "failed to find home directory")

		viper.AddConfigPath(home)
		viper.SetConfigName(".superplane")

		path := fmt.Sprintf("%s/.superplane.yaml", home)

		// #nosec
		_, err = os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Warning: could not ensure config file exists:", err)
		}
	}

	viper.SetEnvPrefix("SUPERPLANE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if Verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}

func defaultBindOptions() core.BindOptions {
	return core.BindOptions{
		NewAPIClient:        DefaultClient,
		DefaultOutputFormat: GetOutputFormat,
	}
}

func GetAPIURL() string {
	if viper.IsSet(ConfigKeyAPIURL) {
		return viper.GetString(ConfigKeyAPIURL)
	}

	return DefaultAPIURL
}

func GetAPIToken() string {
	return viper.GetString(ConfigKeyAPIToken)
}

func GetOutputFormat() string {
	if viper.IsSet(ConfigKeyFormat) {
		return viper.GetString(ConfigKeyFormat)
	}

	return "text"
}

// Checks if an error is present.
//
// If it is present, it displays the provided message and exits with status 1.
func CheckWithMessage(err error, message string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", message)

		Exit(1)
	}
}

func Exit(code int) {
	if flag.Lookup("test.v") == nil {
		os.Exit(1)
	} else {
		panic(fmt.Sprintf("exit %d", code))
	}
}
