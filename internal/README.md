# Internal Package Layout

The `internal/` tree mirrors the package boundaries defined in [ADR 0001](../docs/adr/0001-initial-architecture.md).

- `app`: application use cases and orchestration
- `domain`: canonical game vocabulary and state records
- `scenario`: immutable scenario definitions and catalogs
- `engine`: deterministic game rules and round resolution
- `ports`: capability-oriented interfaces consumed by core packages
- `projection`: read models derived from canonical state
- `adapters`: concrete infrastructure implementations

These packages start intentionally small so the first implementation work can grow into stable boundaries rather than accidental ones.
