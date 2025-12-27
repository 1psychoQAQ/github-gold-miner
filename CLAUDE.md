# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GitHub Gold Miner is an automated AI programming tool discovery system that finds high-potential AI coding tools on GitHub and sends notifications via Feishu (Lark) webhooks. The system fetches trending repos and topic-based repos, filters them, analyzes them with Gemini AI, and notifies users of promising projects.

## Common Commands

```bash
# Single mining run
go run cmd/app/main.go -mode=mine

# Mining with custom concurrency
go run cmd/app/main.go -mode=mine -concurrency=5

# Scheduled mining (every N minutes)
go run cmd/app/main.go -mode=mine -interval=30 -concurrency=5

# Semantic search across stored projects
go run cmd/app/main.go -mode=search -q="code generation tool"

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/adapter/analyzer/...

# Build binary
go build -o bin/github-gold-miner cmd/app/main.go
```

## Required Environment Variables

- `GITHUB_TOKEN`: GitHub Personal Access Token
- `GEMINI_API_KEY`: Google Gemini API Key
- `FEISHU_WEBHOOK`: Feishu webhook URL (optional, skips notification if unset)
- `DATABASE_URL`: PostgreSQL connection string (or uses hardcoded default)

## Architecture

The project follows a hexagonal (ports and adapters) architecture:

```
cmd/app/main.go          # Entry point, CLI parsing, dependency wiring
internal/
├── domain/model.go      # Core Repo entity with LLM analysis fields
├── port/interfaces.go   # Port interfaces defining boundaries
├── service/             # Business logic orchestration
└── adapter/             # Interface implementations
    ├── github/          # Scouter: fetches repos from GitHub API
    ├── filter/          # Filter: time-based and activity filtering
    ├── analyzer/        # Analyzer: star growth + concurrent LLM analysis
    ├── gemini/          # Appraiser: Gemini AI integration
    ├── repository/      # Repository: PostgreSQL storage via GORM
    └── feishu/          # Notifier: Feishu webhook notifications
```

### Port Interfaces (internal/port/interfaces.go)

- **Scouter**: Fetches repos from GitHub (trending + topic-based)
- **Filter**: Applies hard rules (creation date, recent commits)
- **Analyzer**: Calculates star growth rate + runs concurrent LLM analysis
- **Appraiser**: AI evaluation interface (Gemini implementation)
- **Repository**: Persistence layer (PostgreSQL/GORM)
- **Notifier**: Push notifications (Feishu webhooks)

### Mining Pipeline

```
Fetcher → Filter → Analyzer → Storage → Notifier
```

1. **Fetcher** collects repos from GitHub Trending and topics (ai-coding, ide-extension, dev-tools)
2. **Filter** removes repos older than 10 days and those without recent commits
3. **Analyzer** calculates star growth rate, then runs concurrent LLM analysis via worker pool
4. **Storage** saves qualifying projects (AI tool + score >= 50), prevents duplicates
5. **Notifier** sends Feishu messages for new discoveries

### Concurrency Model

The LLM analysis phase uses a worker pool pattern (analyzer.go):
- Jobs channel distributes repos to workers
- Configurable worker count via `-concurrency` flag
- Each analysis has 30-second timeout
- Failed analyses don't block the pipeline

### Testing Pattern

Tests use interface-based mocking. The `ContentGenerator` interface in gemini/appraiser.go allows mocking AI responses. Filter and analyzer tests inject `nowFunc` for time-based testing.
