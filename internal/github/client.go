// Package github wraps the go-github client and exposes the subset of
// GitHub API calls that DevPulse needs.
package github

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"
)

// Client is a thin wrapper around the upstream go-github client.
type Client struct {
	inner *gh.Client
}

// NewClient creates an authenticated (or unauthenticated when token == "")
// GitHub API client.
func NewClient(ctx context.Context, token string) (*Client, error) {
	var inner *gh.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		inner = gh.NewClient(oauth2.NewClient(ctx, ts))
	} else {
		inner = gh.NewClient(nil)
	}
	return &Client{inner: inner}, nil
}

// splitRepo splits "owner/repo" into two strings.
func splitRepo(repo string) (owner, name string, err error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo format %q — expected owner/name", repo)
	}
	return parts[0], parts[1], nil
}

// ListOpenPRs returns all open pull requests for the given repository.
func (c *Client) ListOpenPRs(ctx context.Context, repo string) ([]*gh.PullRequest, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return nil, err
	}

	var all []*gh.PullRequest
	opts := &gh.PullRequestListOptions{
		State:       "open",
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	for {
		prs, resp, err := c.inner.PullRequests.List(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("listing open PRs: %w", err)
		}
		all = append(all, prs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// ListClosedPRs returns closed pull requests (up to maxPages pages of 100 each).
func (c *Client) ListClosedPRs(ctx context.Context, repo string, maxPages int) ([]*gh.PullRequest, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return nil, err
	}

	var all []*gh.PullRequest
	opts := &gh.PullRequestListOptions{
		State:       "closed",
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	pages := 0
	for {
		prs, resp, err := c.inner.PullRequests.List(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("listing closed PRs: %w", err)
		}
		all = append(all, prs...)
		pages++
		if resp.NextPage == 0 || pages >= maxPages {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// ListOpenIssues returns open issues (excluding pull requests) for the repo.
func (c *Client) ListOpenIssues(ctx context.Context, repo string) ([]*gh.Issue, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return nil, err
	}

	var all []*gh.Issue
	opts := &gh.IssueListByRepoOptions{
		State:       "open",
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	for {
		issues, resp, err := c.inner.Issues.ListByRepo(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("listing open issues: %w", err)
		}
		for _, iss := range issues {
			if !iss.IsPullRequest() {
				all = append(all, iss)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// ListDependabotAlerts returns open Dependabot security alerts for the repo.
// Requires the repo to have Dependabot enabled and the token to have the
// `security_events` scope.
func (c *Client) ListDependabotAlerts(ctx context.Context, repo string) ([]*gh.DependabotAlert, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return nil, err
	}

	stateOpen := "open"
	var all []*gh.DependabotAlert
	opts := &gh.ListAlertsOptions{
		State:       &stateOpen,
		ListOptions: gh.ListOptions{PerPage: 100},
	}
	for {
		alerts, resp, err := c.inner.Dependabot.ListRepoAlerts(ctx, owner, name, opts)
		if err != nil {
			// Treat 403/404 as "feature not available" and return empty list
			return nil, fmt.Errorf("listing Dependabot alerts: %w", err)
		}
		all = append(all, alerts...)
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}
	return all, nil
}
