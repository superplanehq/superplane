package admincli

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/superplanehq/superplane/pkg/admincli/commands/fleets"
	"github.com/superplanehq/superplane/pkg/admincli/commands/runners"
	"github.com/superplanehq/superplane/pkg/admincli/commands/tasks"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

const (
	DefaultAPIURL   = "http://localhost:8000"
	ConfigKeyURL    = "url"
	ConfigKeyToken  = "token"
	ConfigKeyOutput = "output"
)

var (
	cfgFile      string
	apiURL       string
	apiToken     string
	Verbose      bool
	OutputFormat string
	config       = viper.New()
)

var RootCmd = &cobra.Command{
	Use:   "admin",
	Short: "Manage a SuperPlane installation",
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if !Verbose {
			log.SetOutput(io.Discard)
		}
		if GetAPIToken() == "" {
			return fmt.Errorf(
				"installation admin token is required; set INSTALLATION_ADMIN_TOKEN or use --token",
			)
		}
		return nil
	},
}

func init() {
	config.SetDefault(ConfigKeyURL, DefaultAPIURL)
	config.SetDefault(ConfigKeyOutput, "text")
	_ = config.BindEnv(ConfigKeyURL, "SUPERPLANE_URL")
	_ = config.BindEnv(ConfigKeyToken, "INSTALLATION_ADMIN_TOKEN")
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.superplane-admin.yaml)")
	RootCmd.PersistentFlags().StringVar(&apiURL, "url", "", "SuperPlane API URL (overrides config and SUPERPLANE_URL)")
	RootCmd.PersistentFlags().StringVar(&apiToken, "token", "", "installation admin token (overrides config and INSTALLATION_ADMIN_TOKEN)")
	RootCmd.PersistentFlags().StringVarP(
		&OutputFormat,
		"output",
		"o",
		"",
		"output format: text|json|yaml (overrides config output)",
	)

	options := defaultBindOptions()
	RootCmd.AddCommand(fleets.NewCommand(options))
	RootCmd.AddCommand(tasks.NewCommand(options))
	RootCmd.AddCommand(runners.NewCommand(options))
}

func initConfig() {
	if cfgFile != "" {
		config.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			return
		}

		config.AddConfigPath(home)
		config.SetConfigName(".superplane-admin")

		path := fmt.Sprintf("%s/.superplane-admin.yaml", home)
		// #nosec G304 -- the path is the user's explicit CLI configuration path.
		file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0600)
		if err == nil {
			_ = file.Close()
		}
	}

	if err := config.ReadInConfig(); err == nil && Verbose {
		fmt.Println("Using config file:", config.ConfigFileUsed())
	}
}

func defaultBindOptions() core.BindOptions {
	return core.BindOptions{
		NewAPIClient:        DefaultClient,
		DefaultOutputFormat: GetOutputFormat,
	}
}

func GetAPIURL() string {
	if value := strings.TrimSpace(apiURL); value != "" {
		return value
	}
	return strings.TrimSpace(config.GetString(ConfigKeyURL))
}

func GetAPIToken() string {
	if value := strings.TrimSpace(apiToken); value != "" {
		return value
	}
	return strings.TrimSpace(config.GetString(ConfigKeyToken))
}

func GetOutputFormat() string {
	if value := strings.TrimSpace(OutputFormat); value != "" {
		return value
	}
	return strings.TrimSpace(config.GetString(ConfigKeyOutput))
}
