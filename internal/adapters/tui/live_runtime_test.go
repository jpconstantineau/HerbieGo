package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/human"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestModelSupportsMultiRoundLiveHumanPlusAIPlay(t *testing.T) {
	t.Parallel()

	starter := scenario.Starter()
	initial := starter.InitialState("starter-match", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", Provider: "ollama", ModelName: "gemma"},
	})

	source := newLiveTestSource(initial)
	model := NewModelWithSubmit(starter, source, source.Submit)
	now := time.Date(2026, time.April, 21, 21, 0, 0, 0, time.UTC)

	runner := app.MatchRunner{
		Collector: app.RoundCollector{
			Now:     func() time.Time { return now },
			Players: scriptedLivePlayers(source),
		},
		Resolver: engine.NewResolver(starter.ResolverOptions()),
		Random:   seeded.New(1),
		OnState:  source.Publish,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		defer source.Close()
		_, _, err := runner.Play(ctx, initial, 2)
		result <- err
	}()

	loaded, _ := model.Update(stateLoadedMsg{state: source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 160
	shell.height = 80

	shell, _, collectingView := expectPublishedView(t, source, shell,
		"Action entry for Procurement Manager",
	)

	shell = submitProcurementTurn(t, shell, "housing=1", "Round 1 keeps housing tight while assembly clears backlog.")
	if shell.workspace != workspaceRoundFeed {
		t.Fatalf("workspace = %v, want %v", shell.workspace, workspaceRoundFeed)
	}

	lockedView := shell.View()
	for _, want := range []string{
		"Mode: round feed",
		"Submissions received: 1/4",
		"Waiting on: Production Manager, Sales Manager, Finance",
		"Controller",
		"Current-turn actions remain hidden until every role is collected",
		"resolves.",
	} {
		if !strings.Contains(lockedView, want) {
			t.Fatalf("locked collecting view missing %q\n%s", want, lockedView)
		}
	}
	for _, unwanted := range []string{
		"Order 1 of housing from forgeco",
		"Round 1 keeps housing tight while assembly clears backlog.",
	} {
		if strings.Contains(lockedView, unwanted) {
			t.Fatalf("locked collecting view leaked %q\n%s", unwanted, lockedView)
		}
	}

	shell, _, _ = expectPublishedView(t, source, shell,
		"Round 1 for Procurement Manager",
		"Current phase: resolving simultaneous turn",
		"The round is resolving.",
	)
	shell, _, revealedView := expectPublishedView(t, source, shell,
		"Round 2 for Procurement Manager",
		"Current phase: revealed round results",
		"[R1]",
		"Procurement Manager: Round 1 keeps housing tight while assembly",
		"clears backlog.",
	)
	shell, _, _ = expectPublishedView(t, source, shell,
		"Round 2 for Procurement Manager",
		"Current phase: hidden simultaneous turn collection",
		"[R1]",
	)

	nextModel, _ := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	shell = nextModel.(Model)
	roundTwoView := shell.View()
	if !strings.Contains(roundTwoView, "Action entry for Procurement Manager") {
		t.Fatalf("round two action entry not available\n%s", roundTwoView)
	}
	for _, want := range []string{
		"Previous accepted commentary",
		"Round 1 keeps housing tight while assembly clears backlog.",
	} {
		if !strings.Contains(roundTwoView, want) {
			t.Fatalf("round two action entry missing prior-round carryover %q\n%s", want, roundTwoView)
		}
	}
	if strings.Contains(roundTwoView, "Submission locked for this round.") {
		t.Fatalf("round two action entry retained locked-round state\n%s", roundTwoView)
	}

	shell = submitProcurementTurn(t, shell, "housing=2", "Round 2 buys ahead of the now-visible pump pull.")
	shell, _, _ = expectPublishedView(t, source, shell,
		"Round 2 for Procurement Manager",
		"Current phase: resolving simultaneous turn",
	)
	shell, _, finalView := expectPublishedView(t, source, shell,
		"Round 3 for Procurement Manager",
		"Current phase: revealed round results",
		"[R2]",
		"Procurement Manager: Round 2 buys ahead of the now-visible",
		"pump pull.",
	)

	if err := <-result; err != nil {
		t.Fatalf("runner.Play() error = %v", err)
	}
	if !strings.Contains(collectingView, "Action entry for Procurement Manager") {
		t.Fatalf("collecting view missing action-entry workspace\n%s", collectingView)
	}
	if !strings.Contains(revealedView, "[R1]") {
		t.Fatalf("revealed view missing round one history\n%s", revealedView)
	}
	if !strings.Contains(finalView, "[R2]") {
		t.Fatalf("final view missing round two history\n%s", finalView)
	}
}

func TestLiveMatchPublishesProviderWaitingSpinnerState(t *testing.T) {
	t.Parallel()

	starter := scenario.Starter()
	initial := starter.InitialState("starter-match", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", Provider: "ollama", ModelName: "gemma"},
	})

	source := newLiveTestSource(initial)
	model := NewModelWithSubmit(starter, source, source.Submit)
	productionRelease := make(chan struct{})
	now := time.Date(2026, time.April, 21, 21, 0, 0, 0, time.UTC)

	runner := app.MatchRunner{
		Collector: app.RoundCollector{
			Now: func() time.Time { return now },
			Players: map[domain.RoleID]ports.Player{
				domain.RoleProcurementManager: human.New(source.SubmitRound),
				domain.RoleProductionManager: llm.New(func(ctx context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
					select {
					case <-productionRelease:
						return domain.ActionSubmission{
							Action: domain.RoleAction{
								Production: &domain.ProductionAction{
									CapacityAllocation: []domain.CapacityAllocation{
										{WorkstationID: "assembly", ProductID: "pump", Capacity: 2},
									},
								},
							},
							Commentary: domain.CommentaryRecord{
								Body: fmt.Sprintf("Round %d production protects the assembly bottleneck.", request.RoleView.Round),
							},
						}, nil
					case <-ctx.Done():
						return domain.ActionSubmission{}, ctx.Err()
					}
				}),
				domain.RoleSalesManager: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
					return domain.ActionSubmission{
						Action: domain.RoleAction{
							Sales: &domain.SalesAction{
								ProductOffers: []domain.ProductOffer{
									{ProductID: "pump", UnitPrice: 14},
									{ProductID: "valve", UnitPrice: 9},
								},
							},
						},
						Commentary: domain.CommentaryRecord{
							Body: fmt.Sprintf("Round %d sales keeps starter pricing stable.", request.RoleView.Round),
						},
					}, nil
				}),
				domain.RoleFinanceController: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
					targets := request.RoleView.ActiveTargets
					targets.EffectiveRound = request.RoleView.Round + 1
					return domain.ActionSubmission{
						Action: domain.RoleAction{
							Finance: &domain.FinanceAction{NextRoundTargets: targets},
						},
						Commentary: domain.CommentaryRecord{
							Body: fmt.Sprintf("Round %d finance holds targets steady for comparison.", request.RoleView.Round),
						},
					}, nil
				}),
			},
		},
		Resolver: engine.NewResolver(starter.ResolverOptions()),
		Random:   seeded.New(1),
		OnState:  source.Publish,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		defer source.Close()
		_, _, err := runner.Play(ctx, initial, 1)
		result <- err
	}()

	loaded, _ := model.Update(stateLoadedMsg{state: source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 160
	shell.height = 80

	shell, _, _ = expectPublishedView(t, source, shell, "Action entry for Procurement Manager")

	shell = submitProcurementTurn(t, shell, "housing=1", "Holding material spend while AI players finish.")
	shell, _, spinnerView := expectPublishedView(t, source, shell,
		"Mode: round feed",
		"Production Manager "+providerSpinnerFrames[0]+" [AI]",
	)
	if strings.Contains(spinnerView, "Procurement Manager "+providerSpinnerFrames[0]+" [Human]") {
		t.Fatalf("spinner leaked onto human role\n%s", spinnerView)
	}

	close(productionRelease)
	if err := <-result; err != nil {
		t.Fatalf("runner.Play() error = %v", err)
	}
}

type liveSubmissionKey struct {
	roleID domain.RoleID
	round  domain.RoundNumber
}

type liveTestSource struct {
	mu          sync.Mutex
	latest      domain.MatchState
	updates     chan domain.MatchState
	submissions chan domain.ActionSubmission
	pending     map[liveSubmissionKey][]domain.ActionSubmission
	closed      bool
}

func newLiveTestSource(initial domain.MatchState) *liveTestSource {
	return &liveTestSource{
		latest:      initial.Clone(),
		updates:     make(chan domain.MatchState, 8),
		submissions: make(chan domain.ActionSubmission, len(initial.Roles)+1),
		pending:     make(map[liveSubmissionKey][]domain.ActionSubmission),
	}
}

func (s *liveTestSource) Snapshot() domain.MatchState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest.Clone()
}

func (s *liveTestSource) Updates() <-chan domain.MatchState {
	return s.updates
}

func (s *liveTestSource) Publish(state domain.MatchState) {
	cloned := state.Clone()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.latest = cloned.Clone()
	updates := s.updates
	s.mu.Unlock()

	updates <- cloned
}

func (s *liveTestSource) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	updates := s.updates
	close(s.submissions)
	s.mu.Unlock()

	close(updates)
}

func (s *liveTestSource) Submit(submission domain.ActionSubmission) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("the live match is no longer accepting submissions")
	}
	submissions := s.submissions
	s.mu.Unlock()

	select {
	case submissions <- submission.Clone():
		return nil
	default:
		return fmt.Errorf("submission queue is full; wait for the current round to catch up")
	}
}

func (s *liveTestSource) SubmitRound(ctx context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
	key := liveSubmissionKey{
		roleID: request.Assignment.RoleID,
		round:  request.RoleView.Round,
	}
	if submission, ok := s.takePending(key); ok {
		return submission, nil
	}

	for {
		select {
		case <-ctx.Done():
			return domain.ActionSubmission{}, fmt.Errorf("tui player %q: %w", request.Assignment.RoleID, ctx.Err())
		case submission, ok := <-s.submissions:
			if !ok {
				return domain.ActionSubmission{}, fmt.Errorf("tui player %q: live submission channel closed", request.Assignment.RoleID)
			}

			submissionKey := liveSubmissionKey{roleID: submission.RoleID, round: submission.Round}
			if submissionKey == key {
				return submission.Clone(), nil
			}
			s.storePending(submissionKey, submission)
		}
	}
}

func (s *liveTestSource) takePending(key liveSubmissionKey) (domain.ActionSubmission, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.pending[key]
	if len(items) == 0 {
		return domain.ActionSubmission{}, false
	}

	submission := items[0].Clone()
	if len(items) == 1 {
		delete(s.pending, key)
	} else {
		s.pending[key] = items[1:]
	}
	return submission, true
}

func (s *liveTestSource) storePending(key liveSubmissionKey, submission domain.ActionSubmission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[key] = append(s.pending[key], submission.Clone())
}

func scriptedLivePlayers(source *liveTestSource) map[domain.RoleID]ports.Player {
	return map[domain.RoleID]ports.Player{
		domain.RoleProcurementManager: human.New(source.SubmitRound),
		domain.RoleProductionManager: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
			return domain.ActionSubmission{
				Action: domain.RoleAction{
					Production: &domain.ProductionAction{
						CapacityAllocation: []domain.CapacityAllocation{
							{WorkstationID: "assembly", ProductID: "pump", Capacity: 2},
						},
					},
				},
				Commentary: domain.CommentaryRecord{
					Body: fmt.Sprintf("Round %d production protects the assembly bottleneck.", request.RoleView.Round),
				},
			}, nil
		}),
		domain.RoleSalesManager: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
			return domain.ActionSubmission{
				Action: domain.RoleAction{
					Sales: &domain.SalesAction{
						ProductOffers: []domain.ProductOffer{
							{ProductID: "pump", UnitPrice: 14},
							{ProductID: "valve", UnitPrice: 9},
						},
					},
				},
				Commentary: domain.CommentaryRecord{
					Body: fmt.Sprintf("Round %d sales keeps starter pricing stable.", request.RoleView.Round),
				},
			}, nil
		}),
		domain.RoleFinanceController: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
			targets := request.RoleView.ActiveTargets
			targets.EffectiveRound = request.RoleView.Round + 1
			return domain.ActionSubmission{
				Action: domain.RoleAction{
					Finance: &domain.FinanceAction{NextRoundTargets: targets},
				},
				Commentary: domain.CommentaryRecord{
					Body: fmt.Sprintf("Round %d finance holds targets steady for comparison.", request.RoleView.Round),
				},
			}, nil
		}),
	}
}

func expectStateUpdate(t *testing.T, model Model, cmd tea.Cmd, wants ...string) (Model, tea.Cmd, string) {
	t.Helper()

	if cmd == nil {
		t.Fatal("expected live update subscription command")
	}

	msg := cmd()
	nextModel, nextCmd := model.Update(msg)
	shell := nextModel.(Model)
	shell.width = 160
	shell.height = 80
	view := shell.View()

	for _, want := range wants {
		if !strings.Contains(view, want) {
			t.Fatalf("live update missing %q\n%s", want, view)
		}
	}

	return shell, nextCmd, view
}

func expectEventuallyStateUpdate(t *testing.T, model Model, cmd tea.Cmd, wants ...string) (Model, tea.Cmd, string) {
	t.Helper()

	currentModel := model
	currentCmd := cmd
	var lastView string

	for range 12 {
		if currentCmd == nil {
			t.Fatalf("expected live update subscription command while waiting for %q", wants)
		}

		msg := currentCmd()
		nextModel, nextCmd := currentModel.Update(msg)
		shell := nextModel.(Model)
		shell.width = 160
		shell.height = 80
		view := shell.View()
		lastView = view

		matched := true
		for _, want := range wants {
			if !strings.Contains(view, want) {
				matched = false
				break
			}
		}
		if matched {
			return shell, nextCmd, view
		}

		currentModel = shell
		currentCmd = nextCmd
	}

	t.Fatalf("live update never matched %q\n%s", wants, lastView)
	return Model{}, nil, ""
}

func expectPublishedView(t *testing.T, source *liveTestSource, model Model, wants ...string) (Model, tea.Cmd, string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	var lastView string

	for time.Now().Before(deadline) {
		select {
		case state, ok := <-source.Updates():
			if !ok {
				t.Fatalf("live update stream closed while waiting for %q", wants)
			}

			nextModel, nextCmd := model.Update(stateLoadedMsg{state: state})
			shell := nextModel.(Model)
			shell.width = 160
			shell.height = 80
			view := shell.View()
			lastView = view

			matched := true
			for _, want := range wants {
				if !strings.Contains(view, want) {
					matched = false
					break
				}
			}
			if matched {
				return shell, nextCmd, view
			}

			model = shell
		case <-time.After(20 * time.Millisecond):
		}
	}

	t.Fatalf("published view never matched %q\n%s", wants, lastView)
	return Model{}, nil, ""
}

func submitProcurementTurn(t *testing.T, model Model, orders string, commentary string) Model {
	t.Helper()

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'1'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune(orders)},
		{Type: tea.KeyEnter},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune(commentary)},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
	} {
		nextModel, _ := model.Update(key)
		model = nextModel.(Model)
	}
	model.width = 160
	model.height = 80

	return model
}
