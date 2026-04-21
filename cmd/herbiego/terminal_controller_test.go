package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestTerminalControllerProcurementSubmissionRequiresConfirmation(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"1",
		"housing",
		"3",
		"Building a small buffer.",
		"n",
		"1",
		"seal_kit",
		"2",
		"Ordering only what assembly can absorb.",
		"y",
	}, "\n") + "\n")
	output := &bytes.Buffer{}
	controller := newTerminalController(scenario.Starter(), input, output)

	submission, err := controller.submitRound(context.Background(), fixtureRequest(domain.RoleProcurementManager))
	if err != nil {
		t.Fatalf("submitRound() error = %v", err)
	}

	if got := len(submission.Action.Procurement.Orders); got != 1 {
		t.Fatalf("len(Orders) = %d, want 1", got)
	}
	order := submission.Action.Procurement.Orders[0]
	if order.PartID != "seal_kit" {
		t.Fatalf("order.PartID = %q, want seal_kit", order.PartID)
	}
	if order.SupplierID != "sealworks" {
		t.Fatalf("order.SupplierID = %q, want sealworks", order.SupplierID)
	}
	if order.Quantity != 2 {
		t.Fatalf("order.Quantity = %d, want 2", order.Quantity)
	}
	if got := submission.Commentary.Body; got != "Ordering only what assembly can absorb." {
		t.Fatalf("submission.Commentary.Body = %q, want final commentary", got)
	}
	if !strings.Contains(output.String(), "Submission discarded. Let's draft it again.") {
		t.Fatalf("output missing redraft message:\n%s", output.String())
	}
}

func TestTerminalControllerFinanceUsesCurrentTargetsAsDefaults(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"",
		"",
		"",
		"",
		"",
		"Keeping target posture stable while we learn the plant.",
		"",
	}, "\n") + "\n")
	output := &bytes.Buffer{}
	controller := newTerminalController(scenario.Starter(), input, output)

	request := fixtureRequest(domain.RoleFinanceController)
	request.RoleView.ActiveTargets = domain.BudgetTargets{
		EffectiveRound:        3,
		ProcurementBudget:     21,
		ProductionSpendBudget: 17,
		RevenueTarget:         33,
		CashFloorTarget:       9,
		DebtCeilingTarget:     14,
	}

	submission, err := controller.submitRound(context.Background(), request)
	if err != nil {
		t.Fatalf("submitRound() error = %v", err)
	}

	targets := submission.Action.Finance.NextRoundTargets
	if targets.EffectiveRound != 4 {
		t.Fatalf("targets.EffectiveRound = %d, want 4", targets.EffectiveRound)
	}
	if targets.ProcurementBudget != 21 || targets.ProductionSpendBudget != 17 || targets.RevenueTarget != 33 || targets.CashFloorTarget != 9 || targets.DebtCeilingTarget != 14 {
		t.Fatalf("targets = %+v, want defaults copied forward", targets)
	}
}

func TestTerminalControllerRejectsBlankCommentary(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"1",
		"pump",
		"14",
		"",
		"Holding price while inventory is still thin.",
		"y",
	}, "\n") + "\n")
	output := &bytes.Buffer{}
	controller := newTerminalController(scenario.Starter(), input, output)

	submission, err := controller.submitRound(context.Background(), fixtureRequest(domain.RoleSalesManager))
	if err != nil {
		t.Fatalf("submitRound() error = %v", err)
	}

	if got := submission.Commentary.Body; got != "Holding price while inventory is still thin." {
		t.Fatalf("submission.Commentary.Body = %q, want recovered commentary", got)
	}
	if !strings.Contains(output.String(), "Explain your reasoning for this round.") {
		t.Fatalf("output missing blank commentary prompt:\n%s", output.String())
	}
}

func fixtureRequest(roleID domain.RoleID) ports.RoundRequest {
	return ports.RoundRequest{
		Assignment: domain.RoleAssignment{
			RoleID:   roleID,
			PlayerID: "human-role",
			IsHuman:  true,
		},
		RoleView: domain.RoundView{
			MatchID:      "starter-match",
			Round:        1,
			ViewerRoleID: roleID,
			Plant: domain.PlantState{
				Cash: 24,
				Debt: 0,
				Backlog: []domain.BacklogEntry{
					{CustomerID: "northbuild", ProductID: "pump", Quantity: 2},
				},
			},
			Metrics: domain.PlantMetrics{
				ThroughputRevenue: 12,
				RoundProfit:       3,
			},
			ActiveTargets: domain.BudgetTargets{
				EffectiveRound:        1,
				ProcurementBudget:     18,
				ProductionSpendBudget: 14,
				RevenueTarget:         28,
				CashFloorTarget:       8,
				DebtCeilingTarget:     15,
			},
		},
		RoleReport: domain.RoleRoundReport{
			Department: domain.DepartmentPerformanceReport{
				RoleID:      roleID,
				DetailLines: []string{"Role-specific detail line."},
			},
			BonusReminder: "Bonus reminder.",
		},
	}
}
