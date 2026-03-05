# DevPulse

**CLI + Repository Health Dashboard with AI** — written in Go.

DevPulse monitors the health of your GitHub repositories in real time and generates intelligent summaries using LLMs (OpenAI or local Ollama) about team velocity, technical debt, security alerts and actionable next steps.

---

## Features

| Feature | Description |
| --------- | ------------- |
| 📊 **Team velocity** | Open PRs, PRs without reviewer, avg review time |
| 🐛 **Technical debt** | Stale issues (>30 d, no label) |
| 🔒 **Security alerts** | Dependabot open alerts |
| 🤖 **AI summaries** | OpenAI GPT-4o or local Ollama (100 % private) |
| 💾 **Local storage** | SQLite — no cloud, no SaaS |
| ⚡ **Single binary** | Cross-platform, zero runtime dependencies |

---

## Installation

```bash
go install github.com/fabianoflorentino/devpulse@latest
```

Or build from source:

```bash
git clone https://github.com/fabianoflorentino/devpulse
cd devpulse
go build -o devpulse .
```

---

## Quick start

```bash
# 1. Export your GitHub token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# 2. Scan a repository
devpulse scan --repo fabianoflorentino/devpulse

# 3. Generate an AI report (OpenAI)
export DEVPULSE_OPENAI_API_KEY=sk-xxxxxxxxxxxx
devpulse report --repo fabianoflorentino/devpulse

# 4. Generate an AI report (Ollama — fully offline)
devpulse report --repo fabianoflorentino/devpulse --provider ollama --model llama3

# 5. Open the TUI dashboard
devpulse dashboard
```

---

## Configuration

DevPulse reads `~/.devpulse.yaml` (created automatically on first run).

```yaml
github:
  token: ghp_xxxxxxxxxxxx   # or use GITHUB_TOKEN env var

openai:
  api_key: sk-xxxxxxxxxxxx  # or use DEVPULSE_OPENAI_API_KEY

ollama:
  base_url: http://localhost:11434
```

All values can also be set via environment variables prefixed with `DEVPULSE_`:

```shell
DEVPULSE_GITHUB_TOKEN
DEVPULSE_OPENAI_API_KEY
DEVPULSE_OLLAMA_BASE_URL
```

---

## Commands

```shell
devpulse scan       --repo owner/name   Collect health metrics
devpulse report     --repo owner/name   Generate AI health report
devpulse dashboard                      Open TUI dashboard
```

---

## Architecture

```shell
devpulse/
├── cmd/
│   ├── root.go         # Cobra root + config init
│   ├── scan.go         # devpulse scan
│   ├── report.go       # devpulse report
│   └── dashboard.go    # devpulse dashboard (TUI)
├── internal/
│   ├── github/         # GitHub API client (go-github)
│   ├── ai/             # LLM summarizer (OpenAI / Ollama)
│   ├── metrics/        # Health metric calculations
│   └── storage/        # SQLite persistence (modernc/sqlite)
└── main.go
```

---

## Why Go?

- **Single binary** — trivial distribution, no runtime dependencies
- **Goroutines** — parallel API requests for speed
- **Low footprint** — runs in CI, Raspberry Pi, minimal containers
- **Rich ecosystem** — cobra, go-github, bubbletea are battle-tested

---

## Roadmap

- [ ] Bubble Tea TUI with live refresh
- [ ] GitHub Actions / CI integration
- [ ] Slack / Discord notifications
- [ ] Multi-repo scanning in one command
- [ ] Historical trend charts

---

## License

MIT
