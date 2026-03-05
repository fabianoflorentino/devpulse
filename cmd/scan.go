package cmd

import (
	"context"
	"fmt"

	"github.com/fabianoflorentino/devpulse/internal/github"
	"github.com/fabianoflorentino/devpulse/internal/metrics"
	"github.com/fabianoflorentino/devpulse/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan a GitHub repository and collect health metrics",
	Long: `Scan fetches data from the GitHub API for the given repository,
calculates health metrics (PR velocity, open issues, stale PRs, etc.)
and persists the results locally so they can be used by report and dashboard.`,
	Example: `  devpulse scan --repo fabianoflorentino/devpulse
  devpulse scan --repo owner/repo --token ghp_xxxx`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringP("repo", "r", "", "Repository in owner/name format (required)")
	_ = scanCmd.MarkFlagRequired("repo")
}

func runScan(cmd *cobra.Command, _ []string) error {
	repo, _ := cmd.Flags().GetString("repo")
	// Priority: --token flag > config file (github.token) > GITHUB_TOKEN env var.
	// All three are resolved automatically by Viper via BindPFlag + BindEnv.
	token := viper.GetString("github.token")

	ctx := context.Background()

	fmt.Printf("Scanning repository: %s\n", repo)

	client, err := github.NewClient(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	collector := metrics.NewCollector(client)
	health, err := collector.Collect(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	db, err := storage.Open("")
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer db.Close()

	if err := db.SaveHealth(ctx, health); err != nil {
		return fmt.Errorf("failed to save health data: %w", err)
	}

	fmt.Printf("Scan complete. Open PRs: %d | Avg review time: %s | Stale issues: %d\n",
		health.OpenPRs, health.AvgReviewTime, health.StaleIssues)
	return nil
}
