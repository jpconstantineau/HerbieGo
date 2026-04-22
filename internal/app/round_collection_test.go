package app_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jpconstantineau/herbiego/internal/adapters/player/human"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

func TestRoundCollectorCollectsMixedPlayersWithoutBranching(t *testing.T) {
	state := fixtureMatchState()
	now := time.Date(2026, time.April, 19, 16, 0, 0, 0, time.UTC)

	var collectedMu sync.Mutex
	collectedViews := map[domain.RoleID]domain.RoundView{}
	collectedReports := map[domain.RoleID]domain.RoleRoundReport{}
	collector := app.RoundCollector{
		Now: func() time.Time { return now },
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
				collectedMu.Lock()
				collectedViews[request.Assignment.RoleID] = request.RoleView.Clone()
				collectedReports[request.Assignment.RoleID] = request.RoleReport.Clone()
				collectedMu.Unlock()
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Procurement: &domain.ProcurementAction{
							Orders: []domain.PurchaseOrderIntent{
								{PartID: "housing", SupplierID: "forgeco", Quantity: 2},
							},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Keeping one round of buffer."},
				}, nil
			}),
			domain.RoleProductionManager: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
				collectedMu.Lock()
				collectedViews[request.Assignment.RoleID] = request.RoleView.Clone()
				collectedReports[request.Assignment.RoleID] = request.RoleReport.Clone()
				collectedMu.Unlock()
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Production: &domain.ProductionAction{
							Releases: []domain.ProductionRelease{
								{ProductID: "pump", Quantity: 1},
							},
							CapacityAllocation: []domain.CapacityAllocation{
								{WorkstationID: "fabrication", ProductID: "pump", Capacity: 1},
							},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Protecting assembly flow."},
				}, nil
			}),
			domain.RoleSalesManager: human.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
				collectedMu.Lock()
				collectedViews[request.Assignment.RoleID] = request.RoleView.Clone()
				collectedReports[request.Assignment.RoleID] = request.RoleReport.Clone()
				collectedMu.Unlock()
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Sales: &domain.SalesAction{
							ProductOffers: []domain.ProductOffer{
								{ProductID: "pump", UnitPrice: 14},
							},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Holding the premium price."},
				}, nil
			}),
			domain.RoleFinanceController: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
				collectedMu.Lock()
				collectedViews[request.Assignment.RoleID] = request.RoleView.Clone()
				collectedReports[request.Assignment.RoleID] = request.RoleReport.Clone()
				collectedMu.Unlock()
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Finance: &domain.FinanceAction{
							NextRoundTargets: domain.BudgetTargets{
								ProcurementBudget:     20,
								ProductionSpendBudget: 16,
								RevenueTarget:         30,
								CashFloorTarget:       8,
								DebtCeilingTarget:     18,
							},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Preserving cash while funding throughput."},
				}, nil
			}),
		},
	}

	actions, err := collector.Collect(context.Background(), state, nil)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if got := len(actions); got != len(state.Roles) {
		t.Fatalf("len(actions) = %d, want %d", got, len(state.Roles))
	}
	collectedMu.Lock()
	for _, assignment := range state.Roles {
		view, ok := collectedViews[assignment.RoleID]
		if !ok {
			t.Fatalf("role %q did not receive a round view", assignment.RoleID)
		}
		if view.ViewerRoleID != assignment.RoleID {
			t.Fatalf("view.ViewerRoleID for %q = %q, want %q", assignment.RoleID, view.ViewerRoleID, assignment.RoleID)
		}
		report, ok := collectedReports[assignment.RoleID]
		if !ok {
			t.Fatalf("role %q did not receive a role report", assignment.RoleID)
		}
		if report.Department.RoleID != assignment.RoleID {
			t.Fatalf("report.Department.RoleID for %q = %q, want %q", assignment.RoleID, report.Department.RoleID, assignment.RoleID)
		}
		if report.BonusReminder == "" {
			t.Fatalf("report.BonusReminder for %q is empty", assignment.RoleID)
		}
	}
	for _, action := range actions {
		if action.MatchID != state.MatchID {
			t.Fatalf("action.MatchID = %q, want %q", action.MatchID, state.MatchID)
		}
		if action.Round != state.CurrentRound {
			t.Fatalf("action.Round = %d, want %d", action.Round, state.CurrentRound)
		}
		if action.SubmittedAt != now {
			t.Fatalf("action.SubmittedAt = %s, want %s", action.SubmittedAt, now)
		}
		if action.Commentary.Visibility != domain.CommentaryPublic {
			t.Fatalf("action.Commentary.Visibility = %q, want %q", action.Commentary.Visibility, domain.CommentaryPublic)
		}
	}
	collectedMu.Unlock()
}

func TestRoundCollectorReusesPreviousActionWhenAIPlayerTimesOut(t *testing.T) {
	state := fixtureMatchState()
	now := time.Date(2026, time.April, 19, 17, 0, 0, 0, time.UTC)

	previous := domain.ActionSubmission{
		ActionID: "sales-r1",
		MatchID:  state.MatchID,
		Round:    1,
		RoleID:   domain.RoleSalesManager,
		Action: domain.RoleAction{
			Sales: &domain.SalesAction{
				ProductOffers: []domain.ProductOffer{
					{ProductID: "pump", UnitPrice: 13},
				},
			},
		},
		Commentary: domain.CommentaryRecord{
			Body: "Previous price held.",
		},
	}

	collector := app.RoundCollector{
		Now: func() time.Time { return now },
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return procurementSubmission(), nil
			}),
			domain.RoleProductionManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return productionSubmission(), nil
			}),
			domain.RoleSalesManager: llm.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{}, context.DeadlineExceeded
			}),
			domain.RoleFinanceController: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return financeSubmission(), nil
			}),
		},
	}

	actions, err := collector.Collect(context.Background(), state, map[domain.RoleID]domain.ActionSubmission{
		domain.RoleSalesManager: previous,
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	sales := findAction(actions, domain.RoleSalesManager)
	if sales == nil {
		t.Fatal("sales action missing from collected results")
	}
	if got := sales.Round; got != state.CurrentRound {
		t.Fatalf("sales.Round = %d, want %d", got, state.CurrentRound)
	}
	if got := sales.Action.Sales.ProductOffers[0].UnitPrice; got != 13 {
		t.Fatalf("sales.ProductOffers[0].UnitPrice = %d, want 13", got)
	}
	if got := sales.Commentary.Body; got != "Previous action reused after AI timeout." {
		t.Fatalf("sales.Commentary.Body = %q, want reuse message", got)
	}
}

func TestRoundCollectorUsesRoleSpecificAIFallbackPolicy(t *testing.T) {
	state := fixtureMatchState()

	collector := app.RoundCollector{
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return procurementSubmission(), nil
			}),
			domain.RoleProductionManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return productionSubmission(), nil
			}),
			domain.RoleSalesManager: llm.New(
				func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
					return domain.ActionSubmission{}, ports.ErrNonResponsive
				},
				llm.WithFallbackPolicy(func(request ports.RoundRequest, cause error) (domain.ActionSubmission, bool, error) {
					return domain.ActionSubmission{
						Action: domain.RoleAction{
							Sales: &domain.SalesAction{
								ProductOffers: []domain.ProductOffer{
									{ProductID: "pump", UnitPrice: 12},
								},
							},
						},
						Commentary: domain.CommentaryRecord{
							Body: "Role-specific fallback policy submitted a conservative price.",
						},
					}, true, nil
				}),
			),
			domain.RoleFinanceController: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return financeSubmission(), nil
			}),
		},
	}

	actions, err := collector.Collect(context.Background(), state, nil)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	sales := findAction(actions, domain.RoleSalesManager)
	if sales == nil {
		t.Fatal("sales action missing from collected results")
	}
	if got := sales.Action.Sales.ProductOffers[0].UnitPrice; got != 12 {
		t.Fatalf("sales.ProductOffers[0].UnitPrice = %d, want 12", got)
	}
	if got := sales.Commentary.Body; got != "Role-specific fallback policy submitted a conservative price." {
		t.Fatalf("sales.Commentary.Body = %q, want custom fallback commentary", got)
	}
}

func TestRoundCollectorReturnsErrorWhenHumanPlayerDoesNotRespond(t *testing.T) {
	state := fixtureMatchState()

	collector := app.RoundCollector{
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return procurementSubmission(), nil
			}),
			domain.RoleProductionManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return productionSubmission(), nil
			}),
			domain.RoleSalesManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{}, ports.ErrNonResponsive
			}),
			domain.RoleFinanceController: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return financeSubmission(), nil
			}),
		},
	}

	_, err := collector.Collect(context.Background(), state, nil)
	if err == nil {
		t.Fatal("Collect() error = nil, want human non-response error")
	}
	if !errors.Is(err, ports.ErrNonResponsive) {
		t.Fatalf("Collect() error = %v, want ErrNonResponsive", err)
	}
}

func TestRoundCollectorStartsAIPlayersBeforeEarlierHumanSubmits(t *testing.T) {
	state := fixtureMatchState()
	humanRelease := make(chan struct{})
	aiStarted := make(chan struct{}, 1)

	collector := app.RoundCollector{
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(ctx context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				select {
				case <-humanRelease:
					return procurementSubmission(), nil
				case <-ctx.Done():
					return domain.ActionSubmission{}, ctx.Err()
				}
			}),
			domain.RoleProductionManager: llm.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				select {
				case aiStarted <- struct{}{}:
				default:
				}
				return productionSubmission(), nil
			}),
			domain.RoleSalesManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Sales: &domain.SalesAction{
							ProductOffers: []domain.ProductOffer{{ProductID: "pump", UnitPrice: 14}},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Sales ready."},
				}, nil
			}),
			domain.RoleFinanceController: llm.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return financeSubmission(), nil
			}),
		},
	}

	result := make(chan error, 1)
	go func() {
		_, err := collector.Collect(context.Background(), state, nil)
		result <- err
	}()

	select {
	case <-aiStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("AI player was not started while the earlier human role was still waiting")
	}

	close(humanRelease)

	if err := <-result; err != nil {
		t.Fatalf("Collect() error = %v, want nil", err)
	}
}

func TestRoundCollectorPreservesCanonicalRoleOrder(t *testing.T) {
	state := fixtureMatchState()

	collector := app.RoundCollector{
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				time.Sleep(40 * time.Millisecond)
				return procurementSubmission(), nil
			}),
			domain.RoleProductionManager: llm.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return productionSubmission(), nil
			}),
			domain.RoleSalesManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				time.Sleep(10 * time.Millisecond)
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Sales: &domain.SalesAction{
							ProductOffers: []domain.ProductOffer{{ProductID: "pump", UnitPrice: 14}},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Sales ready."},
				}, nil
			}),
			domain.RoleFinanceController: llm.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return financeSubmission(), nil
			}),
		},
	}

	actions, err := collector.Collect(context.Background(), state, nil)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	got := make([]domain.RoleID, 0, len(actions))
	for _, action := range actions {
		got = append(got, action.RoleID)
	}

	want := make([]domain.RoleID, 0, len(state.Roles))
	for _, assignment := range state.Roles {
		want = append(want, assignment.RoleID)
	}

	if !slices.Equal(got, want) {
		t.Fatalf("collected role order = %v, want %v", got, want)
	}
}

func TestRoundCollectorReportsProviderWaitingProgress(t *testing.T) {
	state := fixtureMatchState()
	productionRelease := make(chan struct{})
	financeRelease := make(chan struct{})

	var (
		mu       sync.Mutex
		progress []domain.RoundFlowState
	)

	collector := app.RoundCollector{
		OnRoundFlow: func(flow domain.RoundFlowState) {
			mu.Lock()
			progress = append(progress, flow.Clone())
			mu.Unlock()
		},
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return procurementSubmission(), nil
			}),
			domain.RoleProductionManager: llm.New(func(ctx context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				select {
				case <-productionRelease:
					return productionSubmission(), nil
				case <-ctx.Done():
					return domain.ActionSubmission{}, ctx.Err()
				}
			}),
			domain.RoleSalesManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Sales: &domain.SalesAction{
							ProductOffers: []domain.ProductOffer{{ProductID: "pump", UnitPrice: 14}},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Sales ready."},
				}, nil
			}),
			domain.RoleFinanceController: llm.New(func(ctx context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				select {
				case <-financeRelease:
					return financeSubmission(), nil
				case <-ctx.Done():
					return domain.ActionSubmission{}, ctx.Err()
				}
			}),
		},
	}

	done := make(chan error, 1)
	go func() {
		_, err := collector.Collect(context.Background(), state, nil)
		done <- err
	}()

	requireProgressState := func(t *testing.T, predicate func(domain.RoundFlowState) bool, description string) {
		t.Helper()

		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			snapshots := slices.Clone(progress)
			mu.Unlock()

			for _, snapshot := range snapshots {
				if predicate(snapshot) {
					return
				}
			}

			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("did not observe progress state %s; saw %+v", description, progress)
	}

	requireProgressState(t, func(flow domain.RoundFlowState) bool {
		return slices.Contains(flow.ProviderWaitingRoles, domain.RoleProductionManager)
	}, "containing production_manager")
	requireProgressState(t, func(flow domain.RoundFlowState) bool {
		return slices.Contains(flow.ProviderWaitingRoles, domain.RoleProductionManager) &&
			slices.Contains(flow.ProviderWaitingRoles, domain.RoleFinanceController)
	}, "containing both production_manager and finance_controller")

	close(productionRelease)
	requireProgressState(t, func(flow domain.RoundFlowState) bool {
		return slices.Equal(flow.ProviderWaitingRoles, []domain.RoleID{domain.RoleFinanceController})
	}, "equal to [finance_controller]")

	close(financeRelease)
	if err := <-done; err != nil {
		t.Fatalf("Collect() error = %v, want nil", err)
	}

	requireProgressState(t, func(flow domain.RoundFlowState) bool {
		return len(flow.ProviderWaitingRoles) == 0
	}, "with no provider waits")
}

func fixtureMatchState() domain.MatchState {
	return domain.MatchState{
		MatchID:      "match-17",
		ScenarioID:   "starter",
		CurrentRound: 2,
		Roles: []domain.RoleAssignment{
			{RoleID: domain.RoleProcurementManager, PlayerID: "human-proc", IsHuman: true},
			{RoleID: domain.RoleProductionManager, PlayerID: "ai-prod", Provider: "ollama", ModelName: "llama3.2:3b"},
			{RoleID: domain.RoleSalesManager, PlayerID: "human-sales", IsHuman: true},
			{RoleID: domain.RoleFinanceController, PlayerID: "ai-fin", Provider: "openrouter", ModelName: "openai/gpt-5-mini"},
		},
		Plant: domain.PlantState{
			Cash:        24,
			DebtCeiling: 15,
			Workstations: []domain.WorkstationState{
				{WorkstationID: "fabrication", CapacityPerRound: 4},
				{WorkstationID: "assembly", CapacityPerRound: 3},
			},
		},
		Customers: []domain.CustomerState{
			{CustomerID: "northbuild", DisplayName: "NorthBuild", Sentiment: 6},
		},
		ActiveTargets: domain.BudgetTargets{
			EffectiveRound:        2,
			ProcurementBudget:     18,
			ProductionSpendBudget: 14,
			RevenueTarget:         28,
			CashFloorTarget:       8,
			DebtCeilingTarget:     15,
		},
		History: domain.RoundHistory{
			RecentRounds: []domain.RoundRecord{
				{
					Round: 1,
					Events: []domain.RoundEvent{
						{
							Type: domain.EventDemandRealized,
							Payload: map[string]any{
								"product_id":         "pump",
								"quantity":           2,
								"offered_unit_price": 14,
							},
						},
						{
							Type: domain.EventFinishedGoodsProduced,
							Payload: map[string]any{
								"product_id":     "pump",
								"quantity":       1,
								"inventory_cost": 5,
							},
						},
						{
							Type: domain.EventProductionReleased,
							Payload: map[string]any{
								"product_id":    "pump",
								"material_cost": 3,
							},
						},
						{
							Type: domain.EventShipmentCompleted,
							Payload: map[string]any{
								"product_id": "pump",
								"quantity":   1,
								"unit_price": 14,
							},
						},
					},
					Commentary: []domain.CommentaryRecord{
						{Body: "Round one commentary."},
					},
				},
			},
		},
	}
}

func procurementSubmission() domain.ActionSubmission {
	return domain.ActionSubmission{
		Action: domain.RoleAction{
			Procurement: &domain.ProcurementAction{
				Orders: []domain.PurchaseOrderIntent{
					{PartID: "housing", SupplierID: "forgeco", Quantity: 1},
				},
			},
		},
		Commentary: domain.CommentaryRecord{Body: "Procurement ready."},
	}
}

func productionSubmission() domain.ActionSubmission {
	return domain.ActionSubmission{
		Action: domain.RoleAction{
			Production: &domain.ProductionAction{
				Releases: []domain.ProductionRelease{
					{ProductID: "pump", Quantity: 1},
				},
				CapacityAllocation: []domain.CapacityAllocation{
					{WorkstationID: "fabrication", ProductID: "pump", Capacity: 1},
				},
			},
		},
		Commentary: domain.CommentaryRecord{Body: "Production ready."},
	}
}

func financeSubmission() domain.ActionSubmission {
	return domain.ActionSubmission{
		Action: domain.RoleAction{
			Finance: &domain.FinanceAction{
				NextRoundTargets: domain.BudgetTargets{
					ProcurementBudget:     18,
					ProductionSpendBudget: 14,
					RevenueTarget:         28,
					CashFloorTarget:       8,
					DebtCeilingTarget:     15,
				},
			},
		},
		Commentary: domain.CommentaryRecord{Body: "Finance ready."},
	}
}

func findAction(actions []domain.ActionSubmission, roleID domain.RoleID) *domain.ActionSubmission {
	for i := range actions {
		if actions[i].RoleID == roleID {
			return &actions[i]
		}
	}
	return nil
}
