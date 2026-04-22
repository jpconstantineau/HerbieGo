# Configure, Launch, and Play

This guide covers the current MVP runtime path for HerbieGo: a Bubble Tea shell with one or more human-controlled roles and the remaining roles handled by AI providers.

## Files

- `herbiego.yaml`: runtime settings for environment, seed, human player count, UI timing, and role-to-provider mapping
- `llm.yaml`: provider catalog that maps provider names to model names, URLs, API protocol families, and optional API keys

## Configure

The repo ships with a local-first default setup:

- `human_players: 1` makes one role human-controlled by default
- `random.seed: 1` keeps startup deterministic unless you override it
- `ui.ai_reveal_delay_seconds: 15` pauses revealed AI-only rounds briefly before advancing
- each canonical role is mapped to a provider name in `herbiego.yaml`
- each provider name is resolved through `llm.yaml`

Default `herbiego.yaml`:

```yaml
environment: local
random:
  seed: 1
human_players: 1
ui:
  ai_reveal_delay_seconds: 15
roles:
  - role_id: procurement_manager
    provider: ollama-localhost
  - role_id: production_manager
    provider: ollama-localhost
  - role_id: sales_manager
    provider: ollama-localhost
  - role_id: finance_controller
    provider: ollama-localhost
```

Default `llm.yaml` provider catalog:

```yaml
models:
  - provider_name: ollama-localhost
    model_name: gemma4:e4b
    url: http://localhost:11434/
    api_sdk_type: ollama
    api_key: ""
  - provider_name: ollama-cloud
    model_name: gemma4:e4b
    url: https://ollama.com/api/
    api_sdk_type: ollama
    api_key: ""
  - provider_name: openrouter
    model_name: openai/gpt-5-mini
    url: https://openrouter.ai/api/v1/
    api_sdk_type: openai
    api_key: ""
```

If you want a different local profile:

- change `human_players` in `herbiego.yaml`
- swap a role's `provider` label in `herbiego.yaml`
- update the matching provider entry in `llm.yaml`
- pass `-human-players` or `-seed` at launch time for a one-off override

## Launch

Verify the repo before running:

```bash
go run ./cmd/quality verify
```

Launch the default mixed human-plus-AI game:

```bash
go run ./cmd/herbiego -rounds 2 -human-players 1
```

Useful launch variants:

```bash
go run ./cmd/herbiego
go run ./cmd/herbiego -rounds 5
go run ./cmd/herbiego -human-players 0
go run ./cmd/herbiego -seed 7
go run ./cmd/herbiego -config custom-herbiego.yaml
```

Current CLI flags:

- `-config`: path to the runtime YAML file, default `herbiego.yaml`
- `-human-players`: override the number of human-controlled roles, use `0` for an AI-only run
- `-rounds`: number of rounds to play before exiting, default `3`
- `-seed`: override the deterministic runtime seed

## Play

The Bubble Tea shell is the primary gameplay surface.

At startup:

- the left pane shows departments and the selected role
- the center workspace starts in action entry for the current human-controlled role
- the right pane shows plant stats and current targets
- the command bar shows navigation and editing hints

The current human turn flow:

1. Edit the action-entry fields for the selected role.
2. Add commentary describing the reasoning for the turn.
3. Enter review mode.
4. Submit the turn.
5. Wait while AI-controlled roles finish the same hidden simultaneous round.
6. Inspect the revealed feed once the round resolves.
7. Move into the next round without restarting the shell.

What to verify during a manual smoke test:

- the Bubble Tea shell opens immediately without a separate terminal-prompt gameplay flow
- the human role can draft, review, and submit a turn inside the action-entry workspace
- the round feed shows submission progress without leaking the current-turn action or commentary before reveal
- the shell advances through collecting, resolving, and revealed states without deadlocking
- after reveal, the resolved commentary and events appear in the feed and the next round becomes playable

## Notes

- The default local profile assumes the configured provider endpoints are available.
- Human role selection still follows the app's preferred role order when `human_players` is greater than zero.
- AI-only runs stay in the Bubble Tea shell so the match can be watched as it resolves.
