# HerbieGo

HerbieGo is a multiplayer computer board game inspired by the manufacturing plant in *The Goal*. Each player takes one of the key plant roles and tries to improve the overall profitability of the business. The tension comes from the fact that players may also be pushed toward local optimization, creating conflicting incentives that can hurt global performance even when an individual role appears to be succeeding.

The project is designed as both a strategy game and a simulation. Over many in-game years, the plant, the market, the available management ideas, and the viable strategies all evolve. New methods can emerge as the simulation progresses, allowing the game to reflect changing thinking in operations and management over time.

## Game Overview

### Premise

The game takes place inside a manufacturing plant modeled after the world described in *The Goal*. Players operate the plant together, but not always with perfectly aligned incentives. The shared objective is to maximize plant profitability, while the challenge is navigating the gap between local success and system-wide success.

### Players and Roles

HerbieGo supports a mix of human players and AI-agent players.

- There can be `0..N` human players.
- There can be `0..N` AI-agent players.
- `N` is the total number of roles required for a match.
- Any role may be played by either a human or an AI agent.
- Different AI-controlled roles may use different LLM providers at the same time.

Example departments and roles may include:

- Procurement
- Production
- Sales
- Additional plant functions introduced as the simulation expands

### Game Roles

The initial version of HerbieGo should focus on a small set of roles that create strong tension between local optimization and plant-wide performance. Each role has:

- A public responsibility visible to all players
- A private bias or hidden objective that shapes decision-making
- A system-prompt-ready description that can be given to either a human player as briefing text or an AI player as role instructions

These hidden objectives are not necessarily "evil" goals. They represent the kinds of local incentives that make real organizations drift away from system-wide optimization.

For the MVP, the canonical playable roles are:

- Procurement Manager
- Production Manager
- Sales Manager
- Finance Controller

The plant itself resolves turns as a system actor. The broader project vision may later add roles such as Plant Manager, but that role is not part of the MVP action roster.

#### Corporate Objectives

All roles operate inside the same plant and contribute to the same corporate objectives:

- Increase long-term plant profitability
- Improve throughput at the system level
- Reduce unnecessary inventory
- Control operating expense without starving the plant
- Maintain reliable customer delivery
- Protect cash flow and the long-term viability of the business

These are the shared goals the team should be trying to optimize together. The game becomes interesting when role-specific incentives pull players away from these corporate objectives.

#### Production Manager

Public responsibility:

- Maximize production output
- Keep machines and labor utilized
- Manage work-in-progress through the shop floor
- Meet production commitments

Hidden objective:

- Keep resources busy and local output high, even when producing more creates excess inventory or worsens bottlenecks elsewhere

Role prompt intent:

The Production Manager should push hard for efficiency, utilization, and output. This role is especially vulnerable to confusing busy work with productive flow, making it a key source of local optimization pressure.

#### Procurement Manager

Public responsibility:

- Secure materials required for operations
- Control input cost
- Protect the plant from shortages
- Build reliable supplier coverage

Hidden objective:

- Earn advantage from bulk buying and unit-cost reductions, even when that increases inventory, ties up cash, or commits the plant to the wrong materials at the wrong time

Role prompt intent:

The Procurement Manager should prioritize supply continuity and favorable purchasing terms. This role tends to see larger buys and cheaper unit prices as success, even when the system-level impact is harmful.

#### Sales Manager

Public responsibility:

- Grow revenue
- Capture demand
- Maintain customer relationships
- Push the plant toward market opportunity

Hidden objective:

- Maximize booked sales and customer promises, even when accepted orders strain capacity, create firefighting, or reduce delivery reliability

Role prompt intent:

The Sales Manager should think aggressively about revenue and opportunity capture. This role is rewarded by demand generation and order volume, which can create conflict with operational realities.

#### Finance Controller

Public responsibility:

- Monitor cash, cost, and financial performance
- Highlight waste and overspending
- Protect the business from financially dangerous decisions
- Provide visibility into profit drivers

Hidden objective:

- Prefer measurable short-term cost reductions and budget discipline, even when those cuts damage throughput, resilience, or future profit

Role prompt intent:

The Finance Controller should think in terms of cost discipline, liquidity, and reported financial health. This role creates healthy pressure on spending, but may overweight local cost savings compared with total system performance.

#### Role Design Notes

This initial role set creates a strong core tension:

- Production wants resources busy
- Procurement wants cheaper inputs and larger buys
- Sales wants more orders and stronger promises
- Finance wants tighter cost control

Future expansions may add roles such as Plant Manager, maintenance, engineering, distribution, human resources, or scenario-specific executives. The same design pattern should continue: public responsibilities aligned with the plant, and hidden incentives that tempt players toward local optimization.

Post-MVP expansions should also deepen shop-floor control. Likely additions include workstation scheduling, batch-size decisions, tooling/setup changes that consume production time, maintenance planning, machine failures, repair duration, and explicit bottleneck-management concepts such as Theory of Constraints, drum-buffer-rope, and planner/scheduler/expeditor responsibilities.

### Core Objective

The primary victory condition is maximizing the plant's long-term profitability. Players are expected to make decisions that affect:

- Throughput
- Inventory
- Operating expense
- Cash position
- Service levels
- Plant resilience over time

The game should make it possible for a player to improve their own local metrics while damaging total system performance. That tradeoff is a core part of the intended gameplay.

### What Makes The Game Interesting

- It is cooperative in outcome but competitive in incentives.
- It simulates the consequences of management decisions over long periods of time.
- Strategy changes as new operating methods and management ideas become available.
- Human and AI players can participate in the same match.
- Each player has incomplete information about the current turn until all actions are collected and resolved.

## Player-Facing Rules And Turn Logic

### Round Structure

Each round represents a step in the life of the plant. A round follows this sequence:

1. Broadcast the current game state to all players.
2. Collect actions from all players.
3. Resolve all actions together.
4. Update the world with resulting system changes and market events.
5. Prepare the next round and broadcast the updated state.

### Hidden Simultaneous Play

Players do not see the current turn decisions of the other players while they are choosing their own actions. All current-turn decisions remain hidden until every action is collected, resolved, and applied to the shared state.

This supports board-game-style simultaneous turns and encourages prediction, negotiation, bluffing, and conflicting local optimization.

### Information Presented To Players

At the start of each round, every player should receive:

- The current visible game state
- A brief history of the last 10 plays
- The comments or explanations attached to those recent plays
- The current metrics relevant to their role and the plant as a whole

Each player then submits:

- Their action for the turn
- A short explanation, rationale, or chat-style note describing their thinking

That commentary is part of the social layer of the game and helps it feel like a multiplayer board game rather than a silent optimizer.

### Turn Resolution Examples

Typical turn resolution may include effects such as:

- Spending cash
- Purchasing raw materials
- Scheduling production
- Moving inventory
- Accepting or rejecting sales opportunities
- Triggering bottlenecks
- Changing delivery performance
- Causing downstream financial or operational consequences

After player actions are resolved, the world update phase may introduce:

- Market demand changes
- Supply disruptions
- Price shifts
- Equipment issues
- Policy or management trend changes
- Other random or scenario-driven events

### Time Horizon

The simulation is intended to span decades of in-game time. Over that period:

- The operating environment changes
- Strategy options evolve
- New management methods become available
- Roles and incentives may shift
- The plant history becomes part of the strategic context

## User Interface Vision

The main interface is a TUI built for immersive play inside the Go ecosystem.

### Primary TUI Layout

- Left pane: department cards such as Procurement, Production, and Sales
- Center pane: event log showing game activity and player commentary
- Bottom pane: command bar for human player orders
- Right pane: live plant statistics such as cash, inventory, and other key metrics

### Social And Explanation Layer

The interface should include a log or chat view where each player explains their reasoning. This is important for:

- Making AI and human decisions legible
- Supporting negotiation and persuasion between roles
- Preserving recent strategic history
- Strengthening the multiplayer board game feel

### Secondary Interface

A secondary troubleshooting or operator interface may be provided for:

- Inspecting game state transitions
- Debugging turn resolution
- Reviewing LLM prompts and responses
- Troubleshooting provider integrations

## Technical Requirements

### Implementation Constraints

- The game must be written in Go.
- It should target the latest Go release practical for the project.
- Compiled binaries must not depend on `cgo`.
- If persistent storage is needed, SQLite is an acceptable choice.

### Architecture

The system should use a decoupled architecture with clear separation between:

- Game state
- Turn orchestration
- Player interfaces
- AI-provider integrations
- Persistence
- Presentation

The core game loop should:

1. Take player inputs for the current round.
2. Update the game state deterministically from those inputs.
3. Publish updated state to the UI layer.
4. Prepare and distribute the next round view for all players.

The UI should be treated as a subscription to state, regardless of visual style. The gameplay engine should not depend on direct UI behavior.

### Suggested High-Level Components

- `Game Engine`: owns rules, phases, turn resolution, and world updates
- `State Store`: owns canonical game state and history
- `Player Gateway`: handles human and AI player input uniformly
- `LLM Connectors`: provider adapters for OpenRouter and Ollama
- `Prompt/Decision Layer`: builds per-role context windows and parses AI decisions
- `UI Layer`: TUI for play and secondary tools for debugging
- `Persistence Layer`: saves games, event history, prompts, and outcomes when needed

### Player Input Model

Human and AI players should be treated through a common player interface where possible. A player implementation should be able to:

- Receive the round state
- Receive the recent 10-turn history and commentary
- Decide on an action
- Provide an explanation or chat message with that action

This abstraction allows:

- Human-only games
- AI-only games
- Mixed human/AI games
- Different AI backends for different roles in the same match

### LLM Provider Support

The game should support both OpenRouter and Ollama as LLM backends.

- Either provider may be used independently
- Both providers may be active in the same match
- Each AI-controlled role may be assigned its own model and provider

The provider layer should be isolated so the rest of the game only depends on a stable decision-making interface.

### TUI Technology

Bubble Tea is a strong default choice for the primary interface because it keeps the project fully within the Go ecosystem and fits well with a state-driven architecture.

## Early Design Principles

- Favor global plant performance over local scorekeeping alone
- Make local optimization a meaningful source of tension
- Keep the engine deterministic where possible and isolate randomness
- Preserve explainability for both human and AI decisions
- Treat turns as simultaneous and hidden until resolution
- Keep UI concerns separate from game logic
- Make AI participation a first-class feature, not an afterthought

## Initial Scope Suggestions

An early playable version could focus on:

- A small set of departments
- A compact economic model
- One plant scenario
- Human and AI mixed play
- A simple turn log with player commentary
- OpenRouter and Ollama integration behind one common interface
- A Bubble Tea TUI with the core four-pane layout

This would create a foundation that can later expand into longer time horizons, richer roles, and more sophisticated management mechanics.

## Project Intent

HerbieGo aims to be both a game and a systems-thinking simulation: a place where players can experience the tension between local decisions and global outcomes inside a plant that evolves over time. The project should be designed so that the gameplay remains understandable to humans, playable from the terminal, and open to experimentation with both human and AI decision-makers.

## Design Docs

- [MVP Game Design](docs/mvp-game-design.md)
- [Canonical Domain Model](docs/domain-model.md)
- [Architecture Decision Records](docs/adr/README.md)

## Contributor Quality Checks

HerbieGo targets Go `1.26.2`, and the contributor quality workflow is intentionally small and repeatable.

Run the full local check suite with:

```bash
go run ./cmd/quality
```

That single command runs:

```bash
gofmt -w <repo .go files>
go test ./...
go tool staticcheck ./...
```

You can also run an individual step when iterating locally:

```bash
go run ./cmd/quality fmt
go run ./cmd/quality test
go run ./cmd/quality lint
```

`staticcheck` is tracked in `go.mod` using Go's `tool` directive, so contributors and CI use the same linter version through the standard Go toolchain.

For CI or other automation, use the non-mutating verification command:

```bash
go run ./cmd/quality verify
```
