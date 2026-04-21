package human

import (
	"context"
	"fmt"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

type SubmitFunc func(context.Context, ports.RoundRequest) (domain.ActionSubmission, error)

// Player adapts an interactive human controller to the shared player port.
type Player struct {
	submit SubmitFunc
}

func New(submit SubmitFunc) *Player {
	return &Player{submit: submit}
}

func (p *Player) SubmitRound(ctx context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
	if p == nil || p.submit == nil {
		return domain.ActionSubmission{}, fmt.Errorf("human player %q: submit handler is not configured", request.Assignment.RoleID)
	}
	return p.submit(ctx, request)
}

func (p *Player) RecoverFromNonResponse(_ context.Context, request ports.RoundRequest, cause error) (domain.ActionSubmission, error) {
	return domain.ActionSubmission{}, fmt.Errorf("human player %q requires an explicit submission: %w", request.Assignment.RoleID, cause)
}
