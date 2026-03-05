package cmd

import (
	"context"
	"fmt"

	"github.com/fabianoflorentino/devpulse/internal/ai"
	"github.com/fabianoflorentino/devpulse/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate an AI-powered health report for a repository",
	Long: `Report reads the latest scan data from local storage and sends it to
an LLM (OpenAI or Ollama) to generate a human-readable health summary
with actionable next-step suggestions.`,
	Example: `  devpulse report --repo fabianoflorentino/devpulse
  devpulse report --repo owner/repo --provider ollama`,
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringP("repo", "r", "", "Repository in owner/name format (required)")
	reportCmd.Flags().String("provider", "openai", "LLM provider: openai | ollama")
	reportCmd.Flags().String("model", "", "Model name (default: gpt-4o for OpenAI, llama3 for Ollama)")
	_ = reportCmd.MarkFlagRequired("repo")
}

func runReport(cmd *cobra.Command, _ []string) error {
	repo, _ := cmd.Flags().GetString("repo")
	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")

	ctx := context.Background()

	db, err := storage.Open("")
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}
	defer db.Close()

	health, err := db.LatestHealth(ctx, repo)
	if err != nil {
		return fmt.Errorf("no scan data found for %s — run `devpulse scan` first: %w", repo, err)
	}

	cfg := ai.Config{
		Provider: provider,
		Model:    model,
		APIKey:   viper.GetString("openai.api_key"),
		BaseURL:  viper.GetString("ollama.base_url"),
	}

	summarizer, err := ai.NewSummarizer(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize AI summarizer: %w", err)
	}

	report, err := summarizer.Summarize(ctx, health)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	fmt.Println(report)
	return nil
}
