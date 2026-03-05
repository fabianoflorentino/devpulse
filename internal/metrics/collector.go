// Package metrics collects and calculates repository health metrics.
package metrics

import (
	"context"
	"fmt"
	"time"

	ghclient "github.com/fabianoflorentino/devpulse/internal/github"
	gh "github.com/google/go-github/v69/github"
)

// Health holds the computed health snapshot for a repository.
type Health struct {
	Repo               string
	ScannedAt          time.Time
	OpenPRs            int
	PRsWithoutReviewer int
	AvgReviewTime      string        // human-readable, e.g. "3h42m"
	avgReviewTime      time.Duration // internal
	StaleIssues        int           // open issues older than 30 days without a label
	SecurityAlerts     int
}

// Collector gathers raw data from the GitHub API and derives Health metrics.
type Collector struct {
	client *ghclient.Client
}

// NewCollector creates a Collector that uses the supplied GitHub client.
func NewCollector(client *ghclient.Client) *Collector {
	return &Collector{client: client}
}

// Collect fetches data for the given repo and returns a Health snapshot.
func (c *Collector) Collect(ctx context.Context, repo string) (*Health, error) {
	h := &Health{
		Repo:      repo,
		ScannedAt: time.Now().UTC(),
	}

	// --- Open PRs ---
	openPRs, err := c.client.ListOpenPRs(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("collecting open PRs: %w", err)
	}
	h.OpenPRs = len(openPRs)
	h.PRsWithoutReviewer = countPRsWithoutReviewer(openPRs)

	// --- Avg review time (from recent closed PRs) ---
	closedPRs, err := c.client.ListClosedPRs(ctx, repo, 3)
	if err != nil {
		return nil, fmt.Errorf("collecting closed PRs: %w", err)
	}
	h.avgReviewTime = avgReviewTime(closedPRs)
	h.AvgReviewTime = fmtDuration(h.avgReviewTime)

	// --- Stale issues ---
	openIssues, err := c.client.ListOpenIssues(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("collecting open issues: %w", err)
	}
	h.StaleIssues = countStaleIssues(openIssues)

	// --- Security alerts (best-effort; ignored on permission error) ---
	alerts, err := c.client.ListDependabotAlerts(ctx, repo)
	if err == nil {
		h.SecurityAlerts = len(alerts)
	}

	return h, nil
}

// countPRsWithoutReviewer returns the number of open PRs that have no
// requested reviewers and no reviews.
func countPRsWithoutReviewer(prs []*gh.PullRequest) int {
	count := 0
	for _, pr := range prs {
		if len(pr.RequestedReviewers) == 0 && len(pr.RequestedTeams) == 0 {
			count++
		}
	}
	return count
}

// avgReviewTime computes the average elapsed time between PR creation and merge
// for closed (merged) PRs.
func avgReviewTime(prs []*gh.PullRequest) time.Duration {
	var total time.Duration
	n := 0
	for _, pr := range prs {
		if pr.MergedAt != nil && pr.CreatedAt != nil {
			total += pr.MergedAt.Time.Sub(pr.CreatedAt.Time)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return total / time.Duration(n)
}

// countStaleIssues returns the number of open issues that are older than
// 30 days and have no labels.
func countStaleIssues(issues []*gh.Issue) int {
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
	count := 0
	for _, iss := range issues {
		if iss.CreatedAt != nil && iss.CreatedAt.Time.Before(cutoff) && len(iss.Labels) == 0 {
			count++
		}
	}
	return count
}

// fmtDuration formats a duration as "Xh Ym" for human reading.
func fmtDuration(d time.Duration) string {
	if d == 0 {
		return "n/a"
	}
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}
