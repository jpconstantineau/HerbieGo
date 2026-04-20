package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

type SubmitFunc func(context.Context, ports.RoundRequest) (domain.ActionSubmission, error)

// Player adapts an AI-backed controller to the shared player port.
type Player struct {
	submit SubmitFunc
}

func New(submit SubmitFunc) *Player {
	return &Player{submit: submit}
}

func (p *Player) SubmitRound(ctx context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
	if p == nil || p.submit == nil {
		return domain.ActionSubmission{}, fmt.Errorf("llm player %q: submit handler is not configured", request.Assignment.RoleID)
	}
	return p.submit(ctx, request)
}

func (p *Player) RecoverFromNonResponse(_ context.Context, request ports.RoundRequest, cause error) (domain.ActionSubmission, error) {
	if !nonResponsive(cause) {
		return domain.ActionSubmission{}, fmt.Errorf("llm player %q: submit round: %w", request.Assignment.RoleID, cause)
	}

	if request.PreviousAcceptedAction != nil {
		reused := request.PreviousAcceptedAction.Clone()
		reused.Commentary = fallbackCommentary(request, "Previous action reused after AI timeout.")
		return reused, nil
	}

	return domain.ActionSubmission{
		Action:     safeNoOpAction(request),
		Commentary: fallbackCommentary(request, "Safe no-op submitted after AI timeout."),
	}, nil
}

func safeNoOpAction(request ports.RoundRequest) domain.RoleAction {
	switch request.Assignment.RoleID {
	case domain.RoleProcurementManager:
		return domain.RoleAction{
			Procurement: &domain.ProcurementAction{
				Orders: []domain.PurchaseOrderIntent{},
			},
		}
	case domain.RoleProductionManager:
		return domain.RoleAction{
			Production: &domain.ProductionAction{
				Releases:           []domain.ProductionRelease{},
				CapacityAllocation: []domain.CapacityAllocation{},
			},
		}
	case domain.RoleSalesManager:
		return domain.RoleAction{
			Sales: &domain.SalesAction{
				ProductOffers: []domain.ProductOffer{},
			},
		}
	case domain.RoleFinanceController:
		return domain.RoleAction{
			Finance: &domain.FinanceAction{
				NextRoundTargets: request.RoleView.ActiveTargets,
			},
		}
	default:
		return domain.RoleAction{}
	}
}

func fallbackCommentary(request ports.RoundRequest, body string) domain.CommentaryRecord {
	return domain.CommentaryRecord{
		ActorID:    domain.ActorID(request.Assignment.PlayerID),
		Visibility: domain.CommentaryPublic,
		Body:       body,
	}
}

func nonResponsive(err error) bool {
	return errors.Is(err, ports.ErrNonResponsive) || errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "timeout")
}
