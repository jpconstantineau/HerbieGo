package projection_test

import (
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/projection"
)

func TestFinanceReportIncludesProjectedCashView(t *testing.T) {
	state := domain.MatchState{
		CurrentRound: 4,
		Plant: domain.PlantState{
			Cash:        6,
			Debt:        2,
			DebtCeiling: 12,
			Receivables: []domain.CashCommitment{
				{CommitmentID: "ar-1", Amount: 5, DueRound: 4},
				{CommitmentID: "ar-2", Amount: 4, DueRound: 5},
			},
			Payables: []domain.CashCommitment{
				{CommitmentID: "ap-1", Amount: 3, DueRound: 4},
				{CommitmentID: "ap-2", Amount: 7, DueRound: 6},
			},
		},
		Metrics: domain.PlantMetrics{
			RoundProfit:       11,
			CashReceipts:      5,
			CashDisbursements: 3,
		},
	}

	report := projection.BuildRoleRoundReport(state, domain.RoleFinanceController)
	details := strings.Join(report.Department.DetailLines, "\n")

	if !strings.Contains(details, "Projected cash after all open commitments is 3 with projected debt 0.") {
		t.Fatalf("finance detail lines missing projected position: %q", details)
	}
	if !strings.Contains(details, "Next-round maturities net to 2 (5 receivable, 3 payable).") {
		t.Fatalf("finance detail lines missing maturity summary: %q", details)
	}
}
