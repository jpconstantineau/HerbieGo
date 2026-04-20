package ports

import (
	"context"
	"errors"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

var ErrNonResponsive = errors.New("player did not respond")

// Player presents one role-facing round contract regardless of whether the
// controller behind it is human, AI, or another future player type.
type Player interface {
	SubmitRound(ctx context.Context, request RoundRequest) (domain.ActionSubmission, error)
	RecoverFromNonResponse(ctx context.Context, request RoundRequest, cause error) (domain.ActionSubmission, error)
}

// RoundRequest is the full turn context delivered to a player for the active round.
type RoundRequest struct {
	Assignment             domain.RoleAssignment
	RoleView               domain.RoundView
	PreviousAcceptedAction *domain.ActionSubmission
}
