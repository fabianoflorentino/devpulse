// Package ai provides a unified interface to LLM providers (OpenAI, Ollama).
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/fabianoflorentino/devpulse/internal/metrics"
)

// Config holds the configuration for the AI summarizer.
type Config struct {
	Provider string // "openai" | "ollama"
	Model    string // model name; defaults applied per provider
	APIKey   string // required for OpenAI
	BaseURL  string // custom base URL (e.g. http://localhost:11434 for Ollama)
}

// Summarizer generates AI-powered health reports.
type Summarizer struct {
	cfg Config
}

// NewSummarizer validates the config and returns a Summarizer.
func NewSummarizer(cfg Config) (*Summarizer, error) {
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch cfg.Provider {
	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai provider requires DEVPULSE_OPENAI_API_KEY or --token flag")
		}
		if cfg.Model == "" {
			cfg.Model = "gpt-4o"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.openai.com/v1"
		}
	case "ollama":
		if cfg.Model == "" {
			cfg.Model = "llama3"
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "http://localhost:11434"
		}
	default:
		return nil, fmt.Errorf("unknown provider %q — supported: openai, ollama", cfg.Provider)
	}
	return &Summarizer{cfg: cfg}, nil
}

var promptTmpl = template.Must(template.New("prompt").Parse(`
You are a senior software engineering coach reviewing repository health data.

Repository: {{.Repo}}
Scanned at: {{.ScannedAt}}

Metrics:
  - Open PRs:              {{.OpenPRs}}
  - PRs without reviewer:  {{.PRsWithoutReviewer}}
  - Avg PR review time:    {{.AvgReviewTime}}
  - Stale issues (>30d, no label): {{.StaleIssues}}
  - Open security alerts:  {{.SecurityAlerts}}

Please provide:
1. A concise health assessment (2-3 sentences).
2. The top 3 risks or bottlenecks based on these metrics.
3. Three actionable next steps the team should take this sprint.

Be direct and practical. Use Markdown formatting.
`))

// Summarize generates a health report for the given Health snapshot.
func (s *Summarizer) Summarize(ctx context.Context, h *metrics.Health) (string, error) {
	var buf bytes.Buffer
	if err := promptTmpl.Execute(&buf, h); err != nil {
		return "", fmt.Errorf("building prompt: %w", err)
	}
	prompt := buf.String()

	switch s.cfg.Provider {
	case "openai":
		return s.callOpenAI(ctx, prompt)
	case "ollama":
		return s.callOllama(ctx, prompt)
	}
	return "", fmt.Errorf("unreachable")
}

// --- OpenAI ---

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *Summarizer) callOpenAI(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(openAIRequest{
		Model: s.cfg.Model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out openAIResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("parsing openai response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("openai error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}

// --- Ollama ---

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func (s *Summarizer) callOllama(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(ollamaRequest{
		Model:  s.cfg.Model,
		Prompt: prompt,
		Stream: false,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out ollamaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("parsing ollama response: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("ollama error: %s", out.Error)
	}
	return out.Response, nil
}
