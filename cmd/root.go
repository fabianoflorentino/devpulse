package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "devpulse",
	Short: "DevPulse — Repository Health CLI with AI",
	Long: `DevPulse monitors the health of your GitHub repositories in real time
and generates intelligent summaries using AI about team velocity,
technical debt, security alerts and automated next-step suggestions.`,
}

// Execute is the entry point for all commands.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.devpulse.yaml or ./devpulse.yaml / .json)")
	rootCmd.PersistentFlags().String("token", "", "GitHub personal access token (or set GITHUB_TOKEN / DEVPULSE_GITHUB_TOKEN)")

	_ = viper.BindPFlag("github.token", rootCmd.PersistentFlags().Lookup("token"))

	// Accept both GITHUB_TOKEN (standard) and DEVPULSE_GITHUB_TOKEN (prefixed).
	_ = viper.BindEnv("github.token", "GITHUB_TOKEN", "DEVPULSE_GITHUB_TOKEN")

	viper.SetEnvPrefix("DEVPULSE")
	viper.AutomaticEnv()
}

func initConfig() {
	if cfgFile != "" {
		// Explicit path: honour the extension to detect format (yaml/json/toml…).
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// Search order: current directory first, then $HOME.
		// Viper will try all supported extensions (.yaml, .json, .toml, …)
		// automatically when SetConfigType is NOT set.
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.SetConfigName(".devpulse") // matches .devpulse.yaml, .devpulse.json …
		// Also look for "devpulse" (without dot) so devpulse.yaml / devpulse.json
		// placed in the project root work without renaming.
		viper.SetConfigName("devpulse")
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
