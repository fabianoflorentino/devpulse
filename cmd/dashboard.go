package cmd

import (
	"context"
	"fmt"

	"github.com/fabianoflorentino/devpulse/internal/storage"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the interactive TUI dashboard",
	Long: `Dashboard launches a terminal-based UI (TUI) built with Bubble Tea
that displays live health metrics for your scanned repositories.
Use arrow keys to navigate; press q or Ctrl+C to quit.`,
	RunE: runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	db, err := storage.Open("")
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer db.Close()

	repos, err := db.ListRepos(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		return fmt.Errorf("no data found — run `devpulse scan --repo owner/name` first")
	}

	// TODO: replace with Bubble Tea TUI model
	fmt.Println("=== DevPulse Dashboard ===")
	for _, r := range repos {
		h, err := db.LatestHealth(ctx, r)
		if err != nil {
			continue
		}
		fmt.Printf("\n[%s]\n  Open PRs      : %d\n  Avg Review    : %s\n  Stale Issues  : %d\n  Security Alerts: %d\n",
			r, h.OpenPRs, h.AvgReviewTime, h.StaleIssues, h.SecurityAlerts)
	}
	return nil
}
