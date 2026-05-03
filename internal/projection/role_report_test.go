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
	var details []string
	for _, section := range report.Department.Sections {
		details = append(details, section.Summary...)
		details = append(details, section.Facts...)
		for _, warning := range section.Warnings {
			details = append(details, warning.Headline, warning.Detail)
		}
	}
	joined := strings.Join(details, "\n")

	if !strings.Contains(joined, "Projected position after all open commitments is cash 3 and debt 0.") {
		t.Fatalf("finance sections missing projected position: %q", joined)
	}
	if !strings.Contains(joined, "Next-round maturities net to 2 (5 receivable, 3 payable).") {
		t.Fatalf("finance sections missing maturity summary: %q", joined)
	}
}
