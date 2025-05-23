package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DefaultAPIURL = "http://localhost:8000"
)

// Configuration keys
const (
	ConfigKeyAPIURL    = "api_url"
	ConfigKeyAuthToken = "auth_token"
	ConfigKeyFormat    = "output_format"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Get and set configuration options",
	Long:  `Get and set CLI configuration options like API URL and authentication token.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [KEY]",
	Short: "Display a configuration value",
	Long:  `Display the value of a specific configuration key.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		if viper.IsSet(key) {
			value := viper.GetString(key)
			fmt.Println(value)
		} else {
			fmt.Printf("Configuration key '%s' not found\n", key)
			os.Exit(1)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [KEY] [VALUE]",
	Short: "Set a configuration value",
	Long:  `Set the value of a specific configuration key.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		viper.Set(key, value)
		err := viper.WriteConfig()
		CheckWithMessage(err, "Failed to write configuration")

		fmt.Printf("Configuration '%s' set to '%s'\n", key, value)
	},
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View all configuration values",
	Long:  `Display all configuration values currently set.`,
	Run: func(cmd *cobra.Command, args []string) {
		allSettings := viper.AllSettings()

		if len(allSettings) == 0 {
			fmt.Println("No configuration values set")
			return
		}

		fmt.Println("Current configuration:")
		for key, value := range allSettings {
			fmt.Printf("  %s: %v\n", key, value)
		}
	},
}

// GetAPIURL returns the configured API URL or the default if not set
func GetAPIURL() string {
	if viper.IsSet(ConfigKeyAPIURL) {
		return viper.GetString(ConfigKeyAPIURL)
	}
	return DefaultAPIURL
}

// GetAuthToken returns the configured authentication token or empty string if not set
func GetAuthToken() string {
	return viper.GetString(ConfigKeyAuthToken)
}

// GetOutputFormat returns the configured output format or "text" if not set
func GetOutputFormat() string {
	if viper.IsSet(ConfigKeyFormat) {
		return viper.GetString(ConfigKeyFormat)
	}
	return "text"
}

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configViewCmd)

	// Set default configuration values
	viper.SetDefault(ConfigKeyAPIURL, DefaultAPIURL)
	viper.SetDefault(ConfigKeyFormat, "text")
}
