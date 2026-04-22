package domain

import (
	"testing"
	"time"
)

func TestRoundRecordCanonicalTimelinePreservesPhaseChronology(t *testing.T) {
	round := RoundRecord{
		Round: 4,
		Actions: []ActionSubmission{
			{
				ActionID:    "sales-4",
				RoleID:      RoleSalesManager,
				SubmittedAt: time.Date(2026, time.April, 21, 10, 0, 5, 0, time.UTC),
				Commentary:  CommentaryRecord{CommentaryID: "sales-commentary", RoleID: RoleSalesManager, Body: "Demand still looks strong."},
			},
			{
				ActionID:    "proc-4",
				RoleID:      RoleProcurementManager,
				SubmittedAt: time.Date(2026, time.April, 21, 10, 0, 1, 0, time.UTC),
				Commentary:  CommentaryRecord{CommentaryID: "proc-commentary", RoleID: RoleProcurementManager, Body: "Restocking the housings."},
			},
		},
		Commentary: []CommentaryRecord{
			{CommentaryID: "sales-commentary", RoleID: RoleSalesManager, Body: "Demand still looks strong."},
			{CommentaryID: "proc-commentary", RoleID: RoleProcurementManager, Body: "Restocking the housings."},
		},
		Events: []RoundEvent{
			{EventID: "event-1", Type: EventSupplyArrived, Summary: "Received 4 housings."},
			{EventID: "event-2", Type: EventMetricSnapshot, Summary: "Recorded round metrics."},
		},
	}

	timeline := round.CanonicalTimeline()
	if len(timeline) != 4 {
		t.Fatalf("len(timeline) = %d, want 4", len(timeline))
	}
	if got := timeline[0].Phase; got != RoundTimelinePhaseIntake {
		t.Fatalf("timeline[0].Phase = %q, want %q", got, RoundTimelinePhaseIntake)
	}
	if got := timeline[0].Commentary.RoleID; got != RoleProcurementManager {
		t.Fatalf("timeline[0].Commentary.RoleID = %q, want %q", got, RoleProcurementManager)
	}
	if got := timeline[1].Commentary.RoleID; got != RoleSalesManager {
		t.Fatalf("timeline[1].Commentary.RoleID = %q, want %q", got, RoleSalesManager)
	}
	if got := timeline[2].Phase; got != RoundTimelinePhaseSimulation {
		t.Fatalf("timeline[2].Phase = %q, want %q", got, RoundTimelinePhaseSimulation)
	}
	if got := timeline[2].Sequence; got != 1 {
		t.Fatalf("timeline[2].Sequence = %d, want 1", got)
	}
	if got := timeline[3].Phase; got != RoundTimelinePhaseSummary {
		t.Fatalf("timeline[3].Phase = %q, want %q", got, RoundTimelinePhaseSummary)
	}
}

func TestRoundRecordCloneDeepCopiesTimeline(t *testing.T) {
	round := RoundRecord{
		Round: 2,
		Timeline: []RoundTimelineEntry{
			{
				Phase:      RoundTimelinePhaseSummary,
				Sequence:   1,
				Kind:       RoundTimelineKindCommentary,
				Commentary: &CommentaryRecord{CommentaryID: "commentary-1", Body: "Original"},
			},
		},
	}

	cloned := round.Clone()
	cloned.Timeline[0].Commentary.Body = "Changed"

	if got := round.Timeline[0].Commentary.Body; got != "Original" {
		t.Fatalf("round timeline commentary body = %q, want %q", got, "Original")
	}
}
