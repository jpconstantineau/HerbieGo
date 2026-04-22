package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

type submissionKey struct {
	roleID domain.RoleID
	round  domain.RoundNumber
}

type liveGameplayController struct {
	mu          sync.Mutex
	latest      domain.MatchState
	updates     chan domain.MatchState
	submissions chan domain.ActionSubmission
	pending     map[submissionKey][]domain.ActionSubmission
	closed      bool
}

func newLiveGameplayController(initial domain.MatchState) *liveGameplayController {
	return &liveGameplayController{
		latest:      initial.Clone(),
		updates:     make(chan domain.MatchState, 8),
		submissions: make(chan domain.ActionSubmission, len(initial.Roles)+1),
		pending:     make(map[submissionKey][]domain.ActionSubmission),
	}
}

func (c *liveGameplayController) Snapshot() domain.MatchState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.latest.Clone()
}

func (c *liveGameplayController) Updates() <-chan domain.MatchState {
	return c.updates
}

func (c *liveGameplayController) Publish(state domain.MatchState) {
	cloned := state.Clone()

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.latest = cloned.Clone()
	updates := c.updates
	c.mu.Unlock()

	updates <- cloned
}

func (c *liveGameplayController) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	updates := c.updates
	close(c.submissions)
	c.mu.Unlock()

	close(updates)
}

func (c *liveGameplayController) Submit(submission domain.ActionSubmission) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("the live match is no longer accepting submissions")
	}
	submissions := c.submissions
	c.mu.Unlock()

	select {
	case submissions <- submission.Clone():
		return nil
	default:
		return fmt.Errorf("submission queue is full; wait for the current round to catch up")
	}
}

func (c *liveGameplayController) SubmitRound(ctx context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
	key := submissionKey{
		roleID: request.Assignment.RoleID,
		round:  request.RoleView.Round,
	}
	if submission, ok := c.takePending(key); ok {
		return submission, nil
	}

	for {
		select {
		case <-ctx.Done():
			return domain.ActionSubmission{}, fmt.Errorf("tui player %q: %w", request.Assignment.RoleID, ctx.Err())
		case submission, ok := <-c.submissions:
			if !ok {
				return domain.ActionSubmission{}, fmt.Errorf("tui player %q: live submission channel closed", request.Assignment.RoleID)
			}

			submissionKey := submissionKey{roleID: submission.RoleID, round: submission.Round}
			if submissionKey == key {
				return submission.Clone(), nil
			}
			c.storePending(submissionKey, submission)
		}
	}
}

func (c *liveGameplayController) takePending(key submissionKey) (domain.ActionSubmission, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := c.pending[key]
	if len(items) == 0 {
		return domain.ActionSubmission{}, false
	}

	submission := items[0].Clone()
	if len(items) == 1 {
		delete(c.pending, key)
	} else {
		c.pending[key] = items[1:]
	}
	return submission, true
}

func (c *liveGameplayController) storePending(key submissionKey, submission domain.ActionSubmission) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending[key] = append(c.pending[key], submission.Clone())
}
