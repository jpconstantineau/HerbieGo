package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/projection"
	"golang.org/x/sync/errgroup"
)

// RoundCollector gathers one action submission per assigned role through the
// shared player contract.
type RoundCollector struct {
	Players map[domain.RoleID]ports.Player
	Now     func() time.Time
}

// Collect walks the assigned roles in stable match order so mixed human and AI
// controllers can submit through the same orchestration path.
func (c RoundCollector) Collect(ctx context.Context, state domain.MatchState, previous map[domain.RoleID]domain.ActionSubmission) ([]domain.ActionSubmission, error) {
	if state.MatchID == "" {
		return nil, fmt.Errorf("app: collect round %d: match id must not be empty", state.CurrentRound)
	}
	if state.CurrentRound <= 0 {
		return nil, fmt.Errorf("app: collect round %d: round must be positive", state.CurrentRound)
	}

	now := c.now()
	collected := make([]domain.ActionSubmission, len(state.Roles))
	group, groupCtx := errgroup.WithContext(ctx)

	for i, assignment := range state.Roles {
		i := i
		assignment := assignment
		player, ok := c.Players[assignment.RoleID]
		if !ok {
			return nil, fmt.Errorf("app: collect round %d: player missing for role %q", state.CurrentRound, assignment.RoleID)
		}

		request := ports.RoundRequest{
			Assignment:             assignment,
			RoleView:               projection.BuildRoundView(state, assignment.RoleID),
			RoleReport:             projection.BuildRoleRoundReport(state, assignment.RoleID),
			PreviousAcceptedAction: clonePrevious(previous[assignment.RoleID]),
		}

		group.Go(func() error {
			submission, err := player.SubmitRound(groupCtx, request)
			if err != nil {
				submission, err = player.RecoverFromNonResponse(groupCtx, request, err)
				if err != nil {
					return fmt.Errorf("app: collect round %d for %q: %w", state.CurrentRound, assignment.RoleID, err)
				}
			}

			collected[i] = normalizeSubmission(submission, request, now)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return collected, nil
}

func (c RoundCollector) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func clonePrevious(previous domain.ActionSubmission) *domain.ActionSubmission {
	if previous == (domain.ActionSubmission{}) {
		return nil
	}

	cloned := previous.Clone()
	return &cloned
}

func normalizeSubmission(submission domain.ActionSubmission, request ports.RoundRequest, now time.Time) domain.ActionSubmission {
	submission.MatchID = request.RoleView.MatchID
	submission.Round = request.RoleView.Round
	submission.RoleID = request.Assignment.RoleID
	submission.SubmittedAt = submittedAt(submission.SubmittedAt, now)
	submission.ActionID = actionID(submission.ActionID, request)

	commentary := submission.Commentary.Clone()
	commentary.CommentaryID = commentaryID(commentary.CommentaryID, request)
	commentary.MatchID = submission.MatchID
	commentary.Round = submission.Round
	commentary.RoleID = submission.RoleID
	commentary.ActorID = commentaryActor(commentary.ActorID, request)
	commentary.Visibility = commentaryVisibility(commentary.Visibility)
	submission.Commentary = commentary

	return submission
}

func submittedAt(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value.UTC()
}

func actionID(existing domain.ActionID, request ports.RoundRequest) domain.ActionID {
	if existing != "" {
		return existing
	}
	return domain.ActionID(fmt.Sprintf("%s-r%d", request.Assignment.RoleID, request.RoleView.Round))
}

func commentaryID(existing domain.CommentaryID, request ports.RoundRequest) domain.CommentaryID {
	if existing != "" {
		return existing
	}
	return domain.CommentaryID(fmt.Sprintf("%s-commentary-r%d", request.Assignment.RoleID, request.RoleView.Round))
}

func commentaryActor(existing domain.ActorID, request ports.RoundRequest) domain.ActorID {
	if existing != "" {
		return existing
	}
	if request.Assignment.PlayerID != "" {
		return domain.ActorID(request.Assignment.PlayerID)
	}
	return domain.ActorID(request.Assignment.RoleID)
}

func commentaryVisibility(existing domain.CommentaryVisibility) domain.CommentaryVisibility {
	if existing != "" {
		return existing
	}
	return domain.CommentaryPublic
}
