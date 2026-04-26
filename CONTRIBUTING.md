# Contributing to HerbieGo

This guide covers the current local development workflow for HerbieGo.

## Prerequisites

- Go `1.26.2` or newer
- Git

The repository tracks its preferred toolchain in [go.mod](go.mod), so a recent Go install can automatically select the pinned toolchain version.

## First-time setup

1. Clone the repository.
2. Download module dependencies:

   ```bash
   go mod download
   ```

3. Review the checked-in runtime config in [herbiego.yaml](herbiego.yaml) and the provider/model catalog in [llm.yaml](llm.yaml).

## Run the app

Start the current bootstrap executable with:

```bash
go run ./cmd/herbiego
```

Useful flags:

- `-config` selects a YAML config file. The default is `herbiego.yaml`.
- `-human-players` overrides how many canonical roles are human-controlled. Use `0` to force an AI-only test run.
- `-seed` overrides the deterministic random seed for a run.

Example:

```bash
go run ./cmd/herbiego -config herbiego.yaml -human-players 0 -seed 42
```

The current executable validates runtime configuration and prints the initialized role summary. The full TUI gameplay loop has not landed yet, so a successful local run is currently a bootstrap check rather than an interactive match.

## Test and validate changes

Run the contributor quality suite with:

```bash
go run ./cmd/quality
```

That command runs:

- `gofmt -w` on repository Go files
- `go test ./...`
- `go tool staticcheck ./...`

When you only want verification without rewriting files, run:

```bash
go run ./cmd/quality verify
```

You can also run one task at a time while iterating:

```bash
go run ./cmd/quality fmt
go run ./cmd/quality test
go run ./cmd/quality lint
```

CI also runs `go vet ./...` and `go build ./...`, so those are useful extra checks before opening a PR.

## Environment variables

HerbieGo does not currently require any environment variables for local bootstrap, tests, or the quality workflow.

Today, runtime provider selection is driven by YAML configuration, not by environment-variable switches inside the application. If future provider adapters add credential requirements, document those alongside the adapter implementation before relying on them in contributor workflows.

## How AI providers are configured

AI role assignments live in [herbiego.yaml](herbiego.yaml). Provider/model connection details live in [llm.yaml](llm.yaml).

Each entry under `roles` configures one canonical role with:

- `role_id`: the role being configured
- `provider`: the named provider entry to use from `llm.yaml`

Example:

```yaml
roles:
  - role_id: sales_manager
    provider: ollama-localhost
```

Each entry under `models` in `llm.yaml` defines:

- `provider_name`: the named provider referenced by `herbiego.yaml`
- `model_name`: the concrete model identifier
- `url`: the full OpenAI-compatible base URL, including the provider-specific path prefix such as `/v1/` or `/api/v1/`
- `api_sdk_type`: the transport family, currently `openai`
- `api_key`: the configured API key value, if any

Example:

```yaml
models:
  - provider_name: openrouter
    model_name: openai/gpt-5-mini
    url: https://openrouter.ai/api/v1/
    api_sdk_type: openai
    api_key: ""
```

`human_players` determines which roles stay human-controlled during startup. The application assigns human control in a fixed order and keeps the provider label on every role so contributors can switch to AI-only runs with a CLI override instead of rewriting config.

All configured providers currently flow through the shared OpenAI-compatible chat-completions adapter. Local Ollama usually works with an empty `api_key`, while Ollama Cloud requires one in the corresponding `llm.yaml` entry.
