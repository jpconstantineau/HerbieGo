package llm

import (
	"context"
	"errors"
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
	if got := submission.Commentary.Body; got != "Safe no-op submitted after AI timeout." {
		t.Fatalf("submission.Commentary.Body = %q, want timeout fallback message", got)
	}
}

func TestRecoverFromNonResponseRejectsGenericProviderFailures(t *testing.T) {
	player := New(nil)
	request := ports.RoundRequest{
		Assignment: domain.RoleAssignment{RoleID: domain.RoleSalesManager},
	}

	_, err := player.RecoverFromNonResponse(context.Background(), request, ports.ErrProviderFailure)
	if err == nil {
		t.Fatal("RecoverFromNonResponse() error = nil, want wrapped provider failure")
	}
	if !errors.Is(err, ports.ErrProviderFailure) {
		t.Fatalf("RecoverFromNonResponse() error = %v, want ErrProviderFailure", err)
	}
}
