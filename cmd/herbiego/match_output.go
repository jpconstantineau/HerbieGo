package main

import (
	"fmt"
	"io"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
)

func renderRoundOutcome(writer io.Writer, result engine.Result) {
	round := result.Round
	metrics := round.Metrics

	fmt.Fprintf(writer, "\n=== Round %d outcome ===\n", round.Round)
	fmt.Fprintf(writer, "Revenue %d | profit %d | cash change %+d | backlog units %d | output units %d\n",
		metrics.ThroughputRevenue,
		metrics.RoundProfit,
		metrics.NetCashChange,
		metrics.BacklogUnits,
		metrics.ProductionOutputUnits,
	)

	if len(round.Commentary) > 0 {
		fmt.Fprintln(writer, "Commentary:")
		for _, commentary := range round.Commentary {
			fmt.Fprintf(writer, "- %s: %s\n", displayRoleName(commentary.RoleID), commentary.Body)
		}
	}

	if len(round.Events) > 0 {
		fmt.Fprintln(writer, "Events:")
		for _, event := range headlineEvents(round.Events, 6) {
			fmt.Fprintf(writer, "- %s\n", event.Summary)
		}
	}
}

func headlineEvents(events []domain.RoundEvent, limit int) []domain.RoundEvent {
	if len(events) <= limit {
		return events
	}
	return events[:limit]
}
