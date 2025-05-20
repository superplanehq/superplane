package cli

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/utils"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var Verbose bool
var apiURL string
var authToken string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "superplane",
	Short: "Superplane command line interface",
	Long: `Superplane CLI - Command line interface for the Superplane API.
	
Allows you to manage Canvases, Event Sources, and Stages.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !Verbose {
			log.SetOutput(io.Discard)
		}

		if apiURL != "" {
			viper.Set(ConfigKeyAPIURL, apiURL)
		}
		if authToken != "" {
			viper.Set(ConfigKeyAuthToken, authToken)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.superplane.yaml)")
	RootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "API URL (overrides config file)")
	RootCmd.PersistentFlags().StringVar(&authToken, "token", "", "authentication token (overrides config file)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		utils.CheckWithMessage(err, "failed to find home directory")

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

	viper.SetDefault(ConfigKeyAPIURL, DefaultAPIURL)
	viper.SetDefault(ConfigKeyFormat, "text")
}
