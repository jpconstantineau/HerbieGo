package llm

import (
	"context"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

func TestRecoverFromNonResponseTreatsProviderTimeoutAsRecoverable(t *testing.T) {
	player := New(nil)
	request := ports.RoundRequest{
		Assignment: domain.RoleAssignment{
			RoleID:   domain.RoleSalesManager,
			PlayerID: "sales-player",
		},
		RoleView: domain.RoundView{
			ActiveTargets: domain.BudgetTargets{EffectiveRound: 3},
		},
	}

	submission, err := player.RecoverFromNonResponse(context.Background(), request, ports.ErrProviderTimeout)
	if err != nil {
		t.Fatalf("RecoverFromNonResponse() error = %v", err)
	}
	if submission.Action.Sales == nil {
		t.Fatalf("RecoverFromNonResponse() = %#v, want sales no-op fallback", submission.Action)
	}
	if got := submission.Commentary.Body; got != "Safe no-op submitted after AI transport failure." {
		t.Fatalf("submission.Commentary.Body = %q, want transport fallback message", got)
	}
}

func TestRecoverFromNonResponseTreatsGenericProviderFailuresAsRecoverable(t *testing.T) {
	player := New(nil)
	request := ports.RoundRequest{
		Assignment: domain.RoleAssignment{RoleID: domain.RoleSalesManager},
		RoleView: domain.RoundView{
			ActiveTargets: domain.BudgetTargets{EffectiveRound: 3},
		},
	}

	submission, err := player.RecoverFromNonResponse(context.Background(), request, ports.ErrProviderFailure)
	if err != nil {
		t.Fatalf("RecoverFromNonResponse() error = %v", err)
	}
	if submission.Action.Sales == nil {
		t.Fatalf("RecoverFromNonResponse() = %#v, want sales no-op fallback", submission.Action)
	}
	if got := submission.Commentary.Body; got != "Safe no-op submitted after AI transport failure." {
		t.Fatalf("submission.Commentary.Body = %q, want transport fallback message", got)
	}
}
