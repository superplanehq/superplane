package cli

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var Verbose bool

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
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.superplane.yaml)")
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
