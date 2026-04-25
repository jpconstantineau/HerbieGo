package projection

import (
	"fmt"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

// BuildRoleRoundReport projects the role-specific report and bonus reminder
// delivered with the canonical round view.
func BuildRoleRoundReport(state domain.MatchState, viewerRoleID domain.RoleID) domain.RoleRoundReport {
	return domain.RoleRoundReport{
		Companywide:   buildCompanywidePerformanceReport(state),
		Department:    buildDepartmentPerformanceReport(state, viewerRoleID),
		BonusReminder: bonusReminder(viewerRoleID),
	}
}

func buildCompanywidePerformanceReport(state domain.MatchState) domain.CompanywidePerformanceReport {
	latestRound, ok := latestRound(state)
	backlogUnits := sumBacklogUnits(state.Plant.Backlog)
	backlogAtRisk := sumBacklogUnitsAtRisk(state.Plant.Backlog)
	inventory := inventorySummary(state.Plant)
	financials := financialSummary(latestRound, ok)

	return domain.CompanywidePerformanceReport{
		Sections: []domain.RoleReportSection{
			{
				Kind:          domain.RoleReportSectionInventoryRisk,
				Title:         "Inventory risk",
				DecisionFocus: "Shows how much working capital is currently trapped in inventory.",
				Summary: []string{
					fmt.Sprintf("Inventory currently ties up %d across parts, WIP, and finished goods.", inventory.TotalValue),
				},
				Inventory: cloneInventorySummary(inventory),
				Facts: []string{
					fmt.Sprintf("Parts value is %d, WIP value is %d, and finished goods value is %d.", inventoryBucketTotal(state.Plant.PartsInventory), wipInventoryValue(state.Plant.WIPInventory), finishedInventoryValue(state.Plant.FinishedInventory)),
				},
				Warnings: inventoryRiskWarnings(state.Plant, inventory),
			},
			{
				Kind:          domain.RoleReportSectionBacklogHealth,
				Title:         "Backlog health and aging",
				DecisionFocus: "Highlights open demand that is still waiting to ship and where service risk is accumulating.",
				Summary: []string{
					fmt.Sprintf("Open backlog totals %d unit(s), with %d unit(s) already at expiry risk.", backlogUnits, backlogAtRisk),
				},
				ProductValues: backlogSummary(state.Plant.Backlog),
				Warnings:      backlogWarnings(state.Plant.Backlog),
			},
			{
				Kind:          domain.RoleReportSectionServiceRisk,
				Title:         "Service risk",
				DecisionFocus: "Shows whether shipments and backlog quality are putting customer service credibility at risk.",
				Summary: []string{
					fmt.Sprintf("On-time shipment rate closed at %d%% with %d lost sales unit(s).", state.Metrics.OnTimeShipmentRate, state.Metrics.LostSalesUnits),
				},
				Metrics: []domain.RoleReportMetric{
					reportMetric("on_time_shipment_rate", "On-time shipment rate", int(state.Metrics.OnTimeShipmentRate), "pct", "Higher values indicate the plant is keeping its service promises."),
					reportMetric("lost_sales_units", "Lost sales", int(state.Metrics.LostSalesUnits), "units", "Lost sales indicate demand the plant could not convert into revenue."),
				},
				Warnings: serviceWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionMarginVariance,
				Title:         "Margin and variance",
				DecisionFocus: "Summarizes whether revenue, production cost, and parts cost are producing healthy economics.",
				Summary: []string{
					fmt.Sprintf("Round profit closed at %d and product contribution summaries are available for %d product(s).", state.Metrics.RoundProfit, len(financials)),
				},
				Metrics: []domain.RoleReportMetric{
					reportMetric("round_profit", "Round profit", int(state.Metrics.RoundProfit), "money", "Profit signals whether the current operating mix is economically healthy."),
					reportMetric("throughput_revenue", "Throughput revenue", int(state.Metrics.ThroughputRevenue), "money", "Revenue shows how much shipped demand converted into cash-generating output."),
				},
				Financials: financials,
				Warnings:   marginWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionCapacityPressure,
				Title:         "Capacity pressure",
				DecisionFocus: "Shows whether the line is running into congestion, idle labor, or overtime dependence.",
				Summary: []string{
					fmt.Sprintf("The plant produced %d unit(s), lost %d unit(s) of capacity, and used %d overtime unit(s) last round.", state.Metrics.ProductionOutputUnits, state.Metrics.CapacityLossUnits, state.Metrics.OvertimeUnits),
				},
				Metrics: []domain.RoleReportMetric{
					reportMetric("production_output_units", "Output", int(state.Metrics.ProductionOutputUnits), "units", "Completed output is the plant's realized production flow."),
					reportMetric("capacity_loss_units", "Capacity loss", int(state.Metrics.CapacityLossUnits), "units", "Capacity loss indicates congestion or stress penalties."),
					reportMetric("overtime_units", "Overtime", int(state.Metrics.OvertimeUnits), "units", "Overtime can protect throughput but increases spend and stress."),
				},
				Facts: []string{
					stressSummaryLine(state.Plant.Workstations),
					laborBottleneckLine(state.Plant.Workstations),
				},
				Warnings: capacityWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionDemandHealth,
				Title:         "Customer sentiment and demand health",
				DecisionFocus: "Shows whether current demand is healthy, weak, or turning fragile because of service or price posture.",
				Summary: []string{
					fmt.Sprintf("New demand realized last round totaled %d product line(s) and customer sentiment averages %d.", len(newSalesSummary(latestRound, ok)), averageCustomerSentiment(state.Customers)),
				},
				ProductValues: newSalesSummary(latestRound, ok),
				Metrics: []domain.RoleReportMetric{
					reportMetric("customer_sentiment", "Average sentiment", averageCustomerSentiment(state.Customers), "score", "Sentiment helps signal whether future demand pressure is durable."),
				},
				Warnings: demandWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionCashDebtPressure,
				Title:         "Cash and debt pressure",
				DecisionFocus: "Shows whether the plant can fund operations comfortably or is drifting toward liquidity stress.",
				Summary: []string{
					fmt.Sprintf("Current cash is %d against debt %d and a debt ceiling of %d.", state.Plant.Cash, state.Plant.Debt, state.Plant.DebtCeiling),
				},
				Metrics: []domain.RoleReportMetric{
					reportMetric("cash_position", "Cash", int(state.Plant.Cash), "money", "Cash is the first buffer against operating volatility."),
					reportMetric("debt_position", "Debt", int(state.Plant.Debt), "money", "Debt rises when the plant spends beyond available cash."),
					reportMetric("net_cash_change", "Net cash change", int(state.Metrics.NetCashChange), "money", "Net cash change shows whether the week strengthened or weakened liquidity."),
				},
				Facts: []string{
					fmt.Sprintf("Open receivables total %d and open payables total %d.", sumCommitmentAmount(state.Plant.Receivables), sumCommitmentAmount(state.Plant.Payables)),
				},
				Warnings: cashDebtWarnings(state),
			},
		},
	}
}

func buildDepartmentPerformanceReport(state domain.MatchState, viewerRoleID domain.RoleID) domain.DepartmentPerformanceReport {
	switch viewerRoleID {
	case domain.RoleProcurementManager:
		return procurementDepartmentReport(state, viewerRoleID)
	case domain.RoleProductionManager:
		return productionDepartmentReport(state, viewerRoleID)
	case domain.RoleSalesManager:
		return salesDepartmentReport(state, viewerRoleID)
	case domain.RoleFinanceController:
		return financeDepartmentReport(state, viewerRoleID)
	default:
		return domain.DepartmentPerformanceReport{
			RoleID:       viewerRoleID,
			BonusSummary: bonusReminder(viewerRoleID),
			Sections: []domain.RoleReportSection{
				{
					Kind:          domain.RoleReportSectionExecutiveSummary,
					Title:         "Executive summary",
					DecisionFocus: "No role-specific report is configured yet.",
					Summary:       []string{"This role currently falls back to the shared companywide report only."},
				},
			},
		}
	}
}

func procurementDepartmentReport(state domain.MatchState, viewerRoleID domain.RoleID) domain.DepartmentPerformanceReport {
	inTransit := sumInTransitSupply(state.Plant.InTransitSupply)
	partsValue := inventoryBucketTotal(state.Plant.PartsInventory)
	return domain.DepartmentPerformanceReport{
		RoleID:        viewerRoleID,
		BonusSummary:  bonusReminder(viewerRoleID),
		FocusQuestion: "Which purchases best protect next-round production without creating avoidable cash and inventory drag?",
		Sections: []domain.RoleReportSection{
			{
				Kind:          domain.RoleReportSectionExecutiveSummary,
				Title:         "Executive summary",
				DecisionFocus: "Set the overall buying posture for this week.",
				Summary: []string{
					fmt.Sprintf("Procurement is balancing %d unit(s) already in transit against %d unit(s) on hand.", inTransit, state.Metrics.PartsOnHandUnits),
					fmt.Sprintf("Supplier reliability is %d%%, so buying posture should reflect both continuity and cash discipline.", state.Metrics.SupplierReliability),
				},
				Warnings: procurementWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionOperatingPicture,
				Title:         "Role-critical operating picture",
				DecisionFocus: "Shows current material coverage, receipts, and supplier posture.",
				Metrics: []domain.RoleReportMetric{
					reportMetric("ordered_parts", "In-transit supply", int(inTransit), "units", "Open purchase orders already provide some next-round protection."),
					reportMetric("parts_on_hand", "Parts on hand", int(state.Metrics.PartsOnHandUnits), "units", "On-hand parts are the immediate material buffer."),
					reportMetric("supplier_reliability", "Supplier reliability", int(state.Metrics.SupplierReliability), "pct", "Reliability signals how much schedule confidence procurement can assume."),
				},
				Facts: []string{
					fmt.Sprintf("On-hand parts inventory value is %d.", partsValue),
					supplierScorecardLine(state.Suppliers),
					nextSupplyArrivalLine(state.Plant.InTransitSupply, state.CurrentRound),
				},
			},
			{
				Kind:          domain.RoleReportSectionConstraintRiskView,
				Title:         "Constraint and risk view",
				DecisionFocus: "Highlights where shortages or overbuy risk are becoming dangerous.",
				Warnings:      procurementWarnings(state),
				Facts: []string{
					fmt.Sprintf("Open inbound supply totals %d unit(s) across current purchase orders.", inTransit),
				},
			},
			{
				Kind:          domain.RoleReportSectionTradeoffPressure,
				Title:         "Tradeoff and pressure view",
				DecisionFocus: "Shows the tension between shortage protection and cash discipline.",
				Tradeoffs: []domain.RoleReportTradeoff{
					{
						Focus:    "Continuity versus overbuy",
						Tension:  "Buying too little risks starving production, but buying too much traps cash in parts inventory.",
						Guidance: fmt.Sprintf("Use the %d unit(s) already in transit before layering on duplicate protection.", inTransit),
					},
					{
						Focus:    "Cheap supply versus reliable supply",
						Tension:  "A lower-cost source is not attractive if reliability makes next-round production fragile.",
						Guidance: supplierScorecardLine(state.Suppliers),
					},
				},
			},
			{
				Kind:          domain.RoleReportSectionDecisionPrompts,
				Title:         "Decision prompts",
				DecisionFocus: "End with the concrete buying questions procurement should answer now.",
				Prompts: []domain.RoleReportPrompt{
					{Question: "Which exposed part is actually worth buying now?", WhyItMatters: "Not every low stock position deserves immediate cash."},
					{Question: "Does existing in-transit supply already cover the next bottleneck risk?", WhyItMatters: "Avoiding duplicate buys preserves liquidity."},
					{Question: "Where is reliability weak enough that supply continuity deserves extra protection?", WhyItMatters: "A fragile supplier base can erase a seemingly efficient order plan."},
				},
			},
		},
	}
}

func productionDepartmentReport(state domain.MatchState, viewerRoleID domain.RoleID) domain.DepartmentPerformanceReport {
	return domain.DepartmentPerformanceReport{
		RoleID:        viewerRoleID,
		BonusSummary:  bonusReminder(viewerRoleID),
		FocusQuestion: "What is the highest-value feasible production plan given visible parts, capacity, WIP, and spend pressure?",
		Sections: []domain.RoleReportSection{
			{
				Kind:          domain.RoleReportSectionExecutiveSummary,
				Title:         "Executive summary",
				DecisionFocus: "Sets whether this week is about throughput, recovery, or congestion control.",
				Summary: []string{
					fmt.Sprintf("Production closed with %d unit(s) of output, %d unit(s) of WIP, and %d unit(s) of capacity loss.", state.Metrics.ProductionOutputUnits, sumWIPUnits(state.Plant.WIPInventory), state.Metrics.CapacityLossUnits),
					fmt.Sprintf("Overtime contributed %d unit(s) while idle labor left %d unit(s) unused.", state.Metrics.OvertimeUnits, state.Metrics.IdleLaborUnits),
				},
				Warnings: productionWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionOperatingPicture,
				Title:         "Role-critical operating picture",
				DecisionFocus: "Shows output flow, bottlenecks, and material readiness.",
				Metrics: []domain.RoleReportMetric{
					reportMetric("wip_units", "WIP", int(sumWIPUnits(state.Plant.WIPInventory)), "units", "WIP indicates how much work is already committed to the line."),
					reportMetric("output_units", "Output", int(state.Metrics.ProductionOutputUnits), "units", "Output shows how much useful work actually finished."),
					reportMetric("capacity_loss", "Capacity loss", int(state.Metrics.CapacityLossUnits), "units", "Capacity loss highlights congestion and stress."),
					reportMetric("idle_labor", "Idle labor", int(state.Metrics.IdleLaborUnits), "units", "Idle labor shows unused labor that could be redeployed."),
					reportMetric("overtime_units", "Overtime", int(state.Metrics.OvertimeUnits), "units", "Overtime may protect output but increases operating pressure."),
				},
				Facts: []string{
					fmt.Sprintf("Finished goods on hand total %d unit(s).", state.Metrics.FinishedGoodsUnits),
					stressSummaryLine(state.Plant.Workstations),
					laborBottleneckLine(state.Plant.Workstations),
				},
			},
			{
				Kind:          domain.RoleReportSectionConstraintRiskView,
				Title:         "Constraint and risk view",
				DecisionFocus: "Shows where line congestion or material readiness can break the plan.",
				Warnings:      productionWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionTradeoffPressure,
				Title:         "Tradeoff and pressure view",
				DecisionFocus: "Frames the throughput-versus-congestion decisions production must make.",
				Tradeoffs: []domain.RoleReportTradeoff{
					{
						Focus:    "Throughput versus congestion",
						Tension:  "More release can improve output only if bottlenecks are still absorbent.",
						Guidance: stressSummaryLine(state.Plant.Workstations),
					},
					{
						Focus:    "Overtime versus spend discipline",
						Tension:  "Selective overtime can rescue high-value output, but broad overtime adds cost without fixing bad flow.",
						Guidance: fmt.Sprintf("Last round used %d overtime unit(s) for %d in overtime cost.", state.Metrics.OvertimeUnits, state.Metrics.OvertimeCost),
					},
				},
			},
			{
				Kind:          domain.RoleReportSectionDecisionPrompts,
				Title:         "Decision prompts",
				DecisionFocus: "Ends with the production plan choices that matter this week.",
				Prompts: []domain.RoleReportPrompt{
					{Question: "What is the highest-value feasible mix this week?", WhyItMatters: "Not every release is equally valuable when capacity is constrained."},
					{Question: "Will more release protect throughput or just clog the bottleneck?", WhyItMatters: "Pushing work into a congested line can lower real output."},
					{Question: "Is selective overtime justified by the output it protects?", WhyItMatters: "Overtime should solve a real constraint, not mask a bad plan."},
				},
			},
		},
	}
}

func salesDepartmentReport(state domain.MatchState, viewerRoleID domain.RoleID) domain.DepartmentPerformanceReport {
	return domain.DepartmentPerformanceReport{
		RoleID:        viewerRoleID,
		BonusSummary:  bonusReminder(viewerRoleID),
		FocusQuestion: "How should pricing shape future demand without creating backlog, service, or margin damage the plant cannot support?",
		Sections: []domain.RoleReportSection{
			{
				Kind:          domain.RoleReportSectionExecutiveSummary,
				Title:         "Executive summary",
				DecisionFocus: "Sets whether sales should press growth, hold steady, or protect credibility.",
				Summary: []string{
					fmt.Sprintf("Sales is carrying %d unit(s) of backlog with %d%% on-time shipment performance.", sumBacklogUnits(state.Plant.Backlog), state.Metrics.OnTimeShipmentRate),
					fmt.Sprintf("Latest throughput revenue is %d and lost sales reached %d unit(s).", state.Metrics.ThroughputRevenue, state.Metrics.LostSalesUnits),
				},
				Warnings: salesWarnings(state),
			},
			{
				Kind:          domain.RoleReportSectionOperatingPicture,
				Title:         "Role-critical operating picture",
				DecisionFocus: "Shows backlog, shipped revenue, and customer demand pressure.",
				Metrics: []domain.RoleReportMetric{
					reportMetric("sales_pipeline", "Backlog", int(sumBacklogUnits(state.Plant.Backlog)), "units", "Backlog is the booked demand already competing for fulfillment."),
					reportMetric("throughput_revenue", "Throughput revenue", int(state.Metrics.ThroughputRevenue), "money", "Revenue shows the demand that actually converted into shipments."),
					reportMetric("on_time_shipment_rate", "On-time shipment rate", int(state.Metrics.OnTimeShipmentRate), "pct", "Service quality shapes future credibility."),
				},
				Facts: []string{
					fmt.Sprintf("Finished goods on hand total %d unit(s).", state.Metrics.FinishedGoodsUnits),
					fmt.Sprintf("Average visible customer sentiment is %d.", averageCustomerSentiment(state.Customers)),
				},
			},
			{
				Kind:          domain.RoleReportSectionConstraintRiskView,
				Title:         "Constraint and risk view",
				DecisionFocus: "Highlights where backlog or service misses make incremental demand dangerous.",
				Warnings:      salesWarnings(state),
				ProductValues: backlogAtRiskSummary(state.Plant.Backlog),
			},
			{
				Kind:          domain.RoleReportSectionTradeoffPressure,
				Title:         "Tradeoff and pressure view",
				DecisionFocus: "Frames the growth-versus-service tradeoff sales is managing.",
				Tradeoffs: []domain.RoleReportTradeoff{
					{
						Focus:    "Revenue growth versus service credibility",
						Tension:  "Pushing for more demand is only healthy if the plant can still fulfill it on time.",
						Guidance: fmt.Sprintf("Backlog at risk already totals %d unit(s).", sumBacklogUnitsAtRisk(state.Plant.Backlog)),
					},
					{
						Focus:    "Demand capture versus margin quality",
						Tension:  "A lower price may book more demand, but it can create weak economics if margin and service both deteriorate.",
						Guidance: fmt.Sprintf("Round profit is %d while throughput revenue is %d.", state.Metrics.RoundProfit, state.Metrics.ThroughputRevenue),
					},
				},
			},
			{
				Kind:          domain.RoleReportSectionDecisionPrompts,
				Title:         "Decision prompts",
				DecisionFocus: "Ends with the pricing questions sales should answer now.",
				Prompts: []domain.RoleReportPrompt{
					{Question: "Are we winning healthy demand or dangerous backlog?", WhyItMatters: "Demand quality matters as much as demand volume."},
					{Question: "Should pricing cool demand until service credibility recovers?", WhyItMatters: "Overpromising can damage future demand more than a cautious week."},
					{Question: "Which product can still support price without outrunning finished-goods availability?", WhyItMatters: "Pricing should respect the plant's visible ability to serve."},
				},
			},
		},
	}
}

func financeDepartmentReport(state domain.MatchState, viewerRoleID domain.RoleID) domain.DepartmentPerformanceReport {
	projectedCash, projectedDebt := projectedPositionAfterCommitments(state.Plant)
	return domain.DepartmentPerformanceReport{
		RoleID:        viewerRoleID,
		BonusSummary:  bonusReminder(viewerRoleID),
		FocusQuestion: "Which next-round targets protect liquidity and discipline without starving the plant of the support it still needs?",
		Sections: []domain.RoleReportSection{
			{
				Kind:          domain.RoleReportSectionExecutiveSummary,
				Title:         "Executive summary",
				DecisionFocus: "Sets whether finance should tighten, hold, or selectively support.",
				Summary: []string{
					fmt.Sprintf("Finance is carrying %d cash, %d debt, and %d round profit.", state.Plant.Cash, state.Plant.Debt, state.Metrics.RoundProfit),
					fmt.Sprintf("Projected position after all open commitments is cash %d and debt %d.", projectedCash, projectedDebt),
				},
				Warnings: financeWarnings(state, projectedCash, projectedDebt),
			},
			{
				Kind:          domain.RoleReportSectionOperatingPicture,
				Title:         "Role-critical operating picture",
				DecisionFocus: "Shows liquidity, cash commitments, and current economic quality.",
				Metrics: []domain.RoleReportMetric{
					reportMetric("margin", "Round profit", int(state.Metrics.RoundProfit), "money", "Profit is the cleanest weekly read on whether the plant is earning its activity."),
					reportMetric("cash_position", "Cash", int(state.Plant.Cash), "money", "Cash is the plant's immediate operating buffer."),
					reportMetric("cash_receipts", "Cash receipts", int(state.Metrics.CashReceipts), "money", "Receipts show recent cash conversion."),
					reportMetric("cash_disbursements", "Cash disbursements", int(state.Metrics.CashDisbursements), "money", "Disbursements show the weekly cash burn."),
					reportMetric("labor_cost", "Labor cost", int(state.Metrics.LaborCost), "money", "Labor cost is a major controllable operating spend."),
					reportMetric("overtime_cost", "Overtime cost", int(state.Metrics.OvertimeCost), "money", "Overtime cost should buy real throughput, not just activity."),
				},
				Facts: []string{
					fmt.Sprintf("Current cash is %d against debt ceiling %d.", state.Plant.Cash, state.Plant.DebtCeiling),
					fmt.Sprintf("Next-round maturities net to %d (%d receivable, %d payable).", nextRoundNetCash(state.Plant, state.CurrentRound), sumCommitmentsDue(state.Plant.Receivables, state.CurrentRound), sumCommitmentsDue(state.Plant.Payables, state.CurrentRound)),
					fmt.Sprintf("Open receivables total %d, with %d due next round.", sumCommitmentAmount(state.Plant.Receivables), sumCommitmentsDue(state.Plant.Receivables, state.CurrentRound)),
					fmt.Sprintf("Open payables total %d, with %d due next round.", sumCommitmentAmount(state.Plant.Payables), sumCommitmentsDue(state.Plant.Payables, state.CurrentRound)),
				},
			},
			{
				Kind:          domain.RoleReportSectionConstraintRiskView,
				Title:         "Constraint and risk view",
				DecisionFocus: "Highlights where liquidity or working-capital stress is becoming dangerous.",
				Warnings:      financeWarnings(state, projectedCash, projectedDebt),
			},
			{
				Kind:          domain.RoleReportSectionTradeoffPressure,
				Title:         "Tradeoff and pressure view",
				DecisionFocus: "Frames the support-versus-discipline tradeoffs finance must enforce.",
				Tradeoffs: []domain.RoleReportTradeoff{
					{
						Focus:    "Liquidity discipline versus throughput support",
						Tension:  "Tighter budgets protect cash, but over-tightening can starve the operations that still create value.",
						Guidance: fmt.Sprintf("Projected post-commitment position is cash %d and debt %d.", projectedCash, projectedDebt),
					},
					{
						Focus:    "Inventory support versus trapped working capital",
						Tension:  "Inventory can protect service, but excess stock traps cash that may be needed elsewhere.",
						Guidance: fmt.Sprintf("Inventory currently represents %d of plant value exposure.", inventorySummary(state.Plant).TotalValue),
					},
				},
			},
			{
				Kind:          domain.RoleReportSectionDecisionPrompts,
				Title:         "Decision prompts",
				DecisionFocus: "Ends with the questions finance should answer when setting next-round guardrails.",
				Prompts: []domain.RoleReportPrompt{
					{Question: "Which spend should tighten and which still deserves support?", WhyItMatters: "The plant rarely needs every budget tightened equally."},
					{Question: "How much liquidity buffer is actually required for the next round?", WhyItMatters: "Targets should reflect visible cash risk rather than generic caution."},
					{Question: "Where is cash being trapped in low-value inventory or overtime?", WhyItMatters: "Working-capital leakage reduces strategic flexibility quickly."},
				},
			},
		},
	}
}

func bonusReminder(roleID domain.RoleID) string {
	switch roleID {
	case domain.RoleProcurementManager:
		return "Bonus reminder: protect supply continuity and favorable part economics without starving plant cash."
	case domain.RoleProductionManager:
		return "Bonus reminder: keep throughput moving and WIP under control at the bottleneck."
	case domain.RoleSalesManager:
		return "Bonus reminder: sales compensation favors booked demand and shipped revenue."
	case domain.RoleFinanceController:
		return "Bonus reminder: maintain liquidity, margins, and target discipline."
	default:
		return "Bonus reminder: optimize for plant-wide performance."
	}
}

func latestRound(state domain.MatchState) (domain.RoundRecord, bool) {
	if len(state.History.RecentRounds) == 0 {
		return domain.RoundRecord{}, false
	}
	return state.History.RecentRounds[len(state.History.RecentRounds)-1].Clone(), true
}

func newSalesSummary(round domain.RoundRecord, ok bool) []domain.ProductValueSummary {
	if !ok {
		return nil
	}
	summary := map[domain.ProductID]domain.ProductValueSummary{}
	for _, event := range round.Events {
		if event.Type != domain.EventDemandRealized {
			continue
		}
		productID, _ := event.Payload["product_id"].(string)
		quantity, _ := event.Payload["quantity"].(int)
		unitPrice, _ := event.Payload["offered_unit_price"].(int)
		item := summary[domain.ProductID(productID)]
		item.ProductID = domain.ProductID(productID)
		item.Count += domain.Units(quantity)
		item.TotalValue += domain.Money(quantity * unitPrice)
		item.Description = "New demand realized last round"
		summary[item.ProductID] = item
	}
	return summaryValues(summary)
}

func backlogSummary(backlog []domain.BacklogEntry) []domain.ProductValueSummary {
	summary := map[domain.ProductID]domain.ProductValueSummary{}
	for _, entry := range backlog {
		item := summary[entry.ProductID]
		item.ProductID = entry.ProductID
		item.Count += entry.Quantity
		item.Description = "Unshipped backlog"
		summary[item.ProductID] = item
	}
	return summaryValues(summary)
}

func backlogAtRiskSummary(backlog []domain.BacklogEntry) []domain.ProductValueSummary {
	summary := map[domain.ProductID]domain.ProductValueSummary{}
	for _, entry := range backlog {
		if entry.AgeInRounds < 2 {
			continue
		}
		item := summary[entry.ProductID]
		item.ProductID = entry.ProductID
		item.Count += entry.Quantity
		item.Description = "Delayed backlog at expiry risk"
		summary[item.ProductID] = item
	}
	return summaryValues(summary)
}

func producedSummary(round domain.RoundRecord, ok bool) []domain.ProductUnitSummary {
	if !ok {
		return nil
	}
	summary := map[domain.ProductID]domain.ProductUnitSummary{}
	for _, event := range round.Events {
		if event.Type != domain.EventFinishedGoodsProduced {
			continue
		}
		productID, _ := event.Payload["product_id"].(string)
		quantity, _ := event.Payload["quantity"].(int)
		item := summary[domain.ProductID(productID)]
		item.ProductID = domain.ProductID(productID)
		item.Count += domain.Units(quantity)
		item.Description = "Finished last round"
		summary[item.ProductID] = item
	}
	return unitSummaryValues(summary)
}

func inventorySummary(plant domain.PlantState) domain.InventoryValueSummary {
	return domain.InventoryValueSummary{
		TotalValue: inventoryBucketTotal(plant.PartsInventory) + wipInventoryValue(plant.WIPInventory) + finishedInventoryValue(plant.FinishedInventory),
		Detail: []domain.InventoryBucketValue{
			{Bucket: "parts", TotalValue: inventoryBucketTotal(plant.PartsInventory)},
			{Bucket: "wip", TotalValue: wipInventoryValue(plant.WIPInventory)},
			{Bucket: "finished_goods", TotalValue: finishedInventoryValue(plant.FinishedInventory)},
		},
	}
}

func financialSummary(round domain.RoundRecord, ok bool) []domain.ProductFinancialSummary {
	if !ok {
		return nil
	}
	summary := map[domain.ProductID]domain.ProductFinancialSummary{}
	for _, event := range round.Events {
		productID, ok := event.Payload["product_id"].(string)
		if !ok || productID == "" {
			continue
		}
		item := summary[domain.ProductID(productID)]
		item.ProductID = domain.ProductID(productID)
		switch event.Type {
		case domain.EventShipmentCompleted:
			quantity, _ := event.Payload["quantity"].(int)
			unitPrice, _ := event.Payload["unit_price"].(int)
			item.Revenue += domain.Money(quantity * unitPrice)
		case domain.EventFinishedGoodsProduced:
			cost, _ := event.Payload["inventory_cost"].(int)
			item.ProductionCost += domain.Money(cost)
		case domain.EventProductionReleased:
			cost, _ := event.Payload["material_cost"].(int)
			item.PartsCost += domain.Money(cost)
		}
		item.ContributionMargin = item.Revenue - item.ProductionCost - item.PartsCost
		summary[item.ProductID] = item
	}
	return financialValues(summary)
}

func sumBacklogUnits(backlog []domain.BacklogEntry) domain.Units {
	total := domain.Units(0)
	for _, entry := range backlog {
		total += entry.Quantity
	}
	return total
}

func sumBacklogUnitsAtRisk(backlog []domain.BacklogEntry) domain.Units {
	total := domain.Units(0)
	for _, entry := range backlog {
		if entry.AgeInRounds < 2 {
			continue
		}
		total += entry.Quantity
	}
	return total
}

func sumInTransitSupply(lots []domain.SupplyLot) domain.Units {
	total := domain.Units(0)
	for _, lot := range lots {
		total += lot.Quantity
	}
	return total
}

func sumWIPUnits(items []domain.WIPInventory) domain.Units {
	total := domain.Units(0)
	for _, item := range items {
		total += item.Quantity
	}
	return total
}

func stressSummaryLine(items []domain.WorkstationState) string {
	if len(items) == 0 {
		return "No workstation stress profile is currently active."
	}

	worst := items[0]
	for _, item := range items[1:] {
		if item.StressCapacityLoss > worst.StressCapacityLoss {
			worst = item
		}
	}

	if worst.StressCapacityLoss <= 0 {
		return "No workstation lost effective capacity to congestion last round."
	}
	return fmt.Sprintf("%s lost %d unit(s) of effective capacity and closed at %d/%d.", worst.DisplayName, worst.StressCapacityLoss, worst.EffectiveCapacityPerRound, worst.CapacityPerRound)
}

func sumCommitmentAmount(items []domain.CashCommitment) domain.Money {
	total := domain.Money(0)
	for _, item := range items {
		total += item.Amount
	}
	return total
}

func sumCommitmentsDue(items []domain.CashCommitment, round domain.RoundNumber) domain.Money {
	total := domain.Money(0)
	for _, item := range items {
		if item.DueRound == round {
			total += item.Amount
		}
	}
	return total
}

func nextRoundNetCash(plant domain.PlantState, round domain.RoundNumber) domain.Money {
	return sumCommitmentsDue(plant.Receivables, round) - sumCommitmentsDue(plant.Payables, round)
}

func projectedPositionAfterCommitments(plant domain.PlantState) (domain.Money, domain.Money) {
	cash := plant.Cash
	debt := plant.Debt

	for _, item := range plant.Payables {
		cash, debt = applyProjectedCashDelta(cash, debt, -item.Amount)
	}
	for _, item := range plant.Receivables {
		cash, debt = applyProjectedCashDelta(cash, debt, item.Amount)
	}

	return cash, debt
}

func applyProjectedCashDelta(cash, debt, delta domain.Money) (domain.Money, domain.Money) {
	if delta == 0 {
		return cash, debt
	}
	if delta > 0 {
		paydown := minMoney(delta, debt)
		debt -= paydown
		cash += delta - paydown
		return cash, debt
	}

	spend := -delta
	if cash >= spend {
		return cash - spend, debt
	}

	return 0, debt + (spend - cash)
}

func nextSupplyArrivalLine(lots []domain.SupplyLot, currentRound domain.RoundNumber) string {
	if len(lots) == 0 {
		return "No inbound supply is currently scheduled."
	}

	next := lots[0]
	for _, lot := range lots[1:] {
		if lot.ArrivalRound < next.ArrivalRound {
			next = lot
		}
	}

	if next.ArrivalRound <= currentRound {
		return fmt.Sprintf("Next inbound lot from %s is due this round.", next.SupplierID)
	}
	if next.ArrivalRound > next.PromisedRound && next.PromisedRound <= currentRound {
		return fmt.Sprintf("Next inbound lot from %s is running %d round(s) late.", next.SupplierID, next.ArrivalRound-currentRound)
	}
	return fmt.Sprintf("Next inbound lot from %s is due in %d round(s).", next.SupplierID, next.ArrivalRound-currentRound)
}

func supplierScorecardLine(suppliers []domain.SupplierState) string {
	if len(suppliers) == 0 {
		return "No supplier scorecard is available."
	}

	best := suppliers[0]
	worst := suppliers[0]
	for _, supplier := range suppliers[1:] {
		if supplier.ReliabilityScore > best.ReliabilityScore {
			best = supplier
		}
		if supplier.ReliabilityScore < worst.ReliabilityScore {
			worst = supplier
		}
	}

	if best.SupplierID == worst.SupplierID {
		return fmt.Sprintf("%s is currently carrying a supplier score of %d.", best.SupplierID, best.ReliabilityScore)
	}
	return fmt.Sprintf("Supplier scorecard ranges from %s at %d to %s at %d.", worst.SupplierID, worst.ReliabilityScore, best.SupplierID, best.ReliabilityScore)
}

func laborBottleneckLine(items []domain.WorkstationState) string {
	if len(items) == 0 {
		return "No labor profile is currently configured."
	}

	mostConstrained := items[0]
	bestRemaining := laborRemaining(items[0])
	for _, item := range items[1:] {
		if remaining := laborRemaining(item); remaining < bestRemaining {
			bestRemaining = remaining
			mostConstrained = item
		}
	}

	return fmt.Sprintf("%s closed with %d labor unit(s) remaining and %d overtime unit(s) used.", mostConstrained.DisplayName, bestRemaining, mostConstrained.OvertimeUsed)
}

func laborRemaining(item domain.WorkstationState) domain.CapacityUnits {
	base := item.LaborCapacityPerRound
	if base <= 0 {
		base = item.CapacityPerRound
	}
	remaining := base - item.LaborUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

func inventoryBucketTotal(items []domain.PartInventory) domain.Money {
	total := domain.Money(0)
	for _, item := range items {
		total += item.UnitCost * domain.Money(item.OnHandQty)
	}
	return total
}

func wipInventoryValue(items []domain.WIPInventory) domain.Money {
	total := domain.Money(0)
	for _, item := range items {
		total += item.UnitCost * domain.Money(item.Quantity)
	}
	return total
}

func finishedInventoryValue(items []domain.FinishedInventory) domain.Money {
	total := domain.Money(0)
	for _, item := range items {
		total += item.UnitCost * domain.Money(item.OnHandQty)
	}
	return total
}

func summaryValues(items map[domain.ProductID]domain.ProductValueSummary) []domain.ProductValueSummary {
	values := make([]domain.ProductValueSummary, 0, len(items))
	for _, item := range items {
		values = append(values, item)
	}
	return values
}

func unitSummaryValues(items map[domain.ProductID]domain.ProductUnitSummary) []domain.ProductUnitSummary {
	values := make([]domain.ProductUnitSummary, 0, len(items))
	for _, item := range items {
		values = append(values, item)
	}
	return values
}

func financialValues(items map[domain.ProductID]domain.ProductFinancialSummary) []domain.ProductFinancialSummary {
	values := make([]domain.ProductFinancialSummary, 0, len(items))
	for _, item := range items {
		values = append(values, item)
	}
	return values
}

func minMoney(left, right domain.Money) domain.Money {
	if left < right {
		return left
	}
	return right
}

func reportMetric(metricID, label string, value int, unit, interpretation string) domain.RoleReportMetric {
	return domain.RoleReportMetric{
		Metric: domain.MetricValue{
			MetricID:    domain.MetricID(metricID),
			Value:       value,
			DisplayUnit: unit,
		},
		Label:          label,
		Interpretation: interpretation,
	}
}

func inventoryRiskWarnings(plant domain.PlantState, inventory domain.InventoryValueSummary) []domain.RoleReportWarning {
	warnings := make([]domain.RoleReportWarning, 0, 2)
	if inventory.TotalValue > plant.Cash && inventory.TotalValue > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "inventory_exceeds_cash",
			Headline: "Inventory value exceeds current cash.",
			Detail:   "Working capital is heavily tied up in stock relative to liquid cash on hand.",
		})
	}
	if len(plant.InspectionHolds) > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "inspection_hold_inventory",
			Headline: "Some inventory is stuck in inspection hold.",
			Detail:   fmt.Sprintf("%d hold lot(s) are not currently available to support demand.", len(plant.InspectionHolds)),
		})
	}
	return warnings
}

func backlogWarnings(backlog []domain.BacklogEntry) []domain.RoleReportWarning {
	atRisk := sumBacklogUnitsAtRisk(backlog)
	if atRisk == 0 {
		return nil
	}
	return []domain.RoleReportWarning{{
		Code:     "aging_backlog",
		Headline: "Aging backlog is accumulating.",
		Detail:   fmt.Sprintf("%d backlog unit(s) are at risk because they have aged two rounds or more.", atRisk),
	}}
}

func serviceWarnings(state domain.MatchState) []domain.RoleReportWarning {
	warnings := make([]domain.RoleReportWarning, 0, 2)
	if state.Metrics.OnTimeShipmentRate < 90 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "service_level_low",
			Headline: "On-time shipment performance is below target.",
			Detail:   "Customer service credibility is at risk if new demand continues to outpace fulfillment.",
		})
	}
	if state.Metrics.LostSalesUnits > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "lost_sales",
			Headline: "The plant is already losing demand.",
			Detail:   fmt.Sprintf("%d unit(s) of demand were lost last round.", state.Metrics.LostSalesUnits),
		})
	}
	return warnings
}

func marginWarnings(state domain.MatchState) []domain.RoleReportWarning {
	if state.Metrics.RoundProfit >= 0 {
		return nil
	}
	return []domain.RoleReportWarning{{
		Code:     "negative_profit",
		Headline: "The latest round closed at a loss.",
		Detail:   "Current pricing, spend, or production mix is not generating healthy economics.",
	}}
}

func capacityWarnings(state domain.MatchState) []domain.RoleReportWarning {
	warnings := make([]domain.RoleReportWarning, 0, 2)
	if state.Metrics.CapacityLossUnits > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "capacity_stress",
			Headline: "Capacity stress reduced effective output.",
			Detail:   fmt.Sprintf("%d unit(s) of capacity were lost to stress or congestion.", state.Metrics.CapacityLossUnits),
		})
	}
	if state.Metrics.OvertimeUnits > state.Metrics.IdleLaborUnits && state.Metrics.OvertimeUnits > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "overtime_dependency",
			Headline: "Output is leaning on overtime.",
			Detail:   "The plant is relying more on overtime than on unused labor slack, which can become expensive quickly.",
		})
	}
	return warnings
}

func demandWarnings(state domain.MatchState) []domain.RoleReportWarning {
	warnings := make([]domain.RoleReportWarning, 0, 2)
	if averageCustomerSentiment(state.Customers) <= 4 && len(state.Customers) > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "customer_sentiment_softening",
			Headline: "Customer sentiment is softening.",
			Detail:   "Future demand may weaken if service or pricing pressure continues.",
		})
	}
	if sumBacklogUnits(state.Plant.Backlog) > state.Metrics.FinishedGoodsUnits && len(state.Plant.Backlog) > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "demand_outpacing_inventory",
			Headline: "Booked demand is outrunning finished goods.",
			Detail:   "Current demand pressure may be turning into service risk rather than healthy growth.",
		})
	}
	return warnings
}

func cashDebtWarnings(state domain.MatchState) []domain.RoleReportWarning {
	projectedCash, projectedDebt := projectedPositionAfterCommitments(state.Plant)
	return financeWarnings(state, projectedCash, projectedDebt)
}

func procurementWarnings(state domain.MatchState) []domain.RoleReportWarning {
	warnings := make([]domain.RoleReportWarning, 0, 3)
	if state.Metrics.SupplierReliability < 90 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "supplier_reliability_low",
			Headline: "Supplier reliability is below comfort range.",
			Detail:   "Receipts may be less dependable, so coverage assumptions should stay conservative.",
		})
	}
	if sumInTransitSupply(state.Plant.InTransitSupply) == 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "no_inbound_supply",
			Headline: "No inbound supply is currently scheduled.",
			Detail:   "Production continuity may depend entirely on existing stock unless procurement acts.",
		})
	}
	if inventoryBucketTotal(state.Plant.PartsInventory) > state.Plant.Cash && state.Metrics.PartsOnHandUnits > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "parts_cash_drag",
			Headline: "Parts inventory is becoming a cash drag.",
			Detail:   "More buys may worsen liquidity unless they clearly protect near-term throughput.",
		})
	}
	return warnings
}

func productionWarnings(state domain.MatchState) []domain.RoleReportWarning {
	warnings := capacityWarnings(state)
	if sumWIPUnits(state.Plant.WIPInventory) > state.Metrics.ProductionOutputUnits && sumWIPUnits(state.Plant.WIPInventory) > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "wip_building",
			Headline: "WIP is building faster than finished output.",
			Detail:   "More release may create congestion instead of more useful throughput.",
		})
	}
	return warnings
}

func salesWarnings(state domain.MatchState) []domain.RoleReportWarning {
	warnings := serviceWarnings(state)
	if sumBacklogUnitsAtRisk(state.Plant.Backlog) > 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "backlog_expiry_risk",
			Headline: "Some backlog is old enough to threaten service credibility.",
			Detail:   "Aggressive pricing may be counterproductive until aging orders are stabilized.",
		})
	}
	return warnings
}

func financeWarnings(state domain.MatchState, projectedCash, projectedDebt domain.Money) []domain.RoleReportWarning {
	warnings := make([]domain.RoleReportWarning, 0, 3)
	if projectedDebt > state.Plant.Debt {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "projected_debt_increase",
			Headline: "Open commitments are likely to increase debt.",
			Detail:   fmt.Sprintf("Projected debt rises to %d after currently visible commitments.", projectedDebt),
		})
	}
	if projectedCash < state.ActiveTargets.CashFloorTarget {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "cash_floor_pressure",
			Headline: "Projected cash falls below the active cash floor target.",
			Detail:   fmt.Sprintf("Projected cash is %d against a target floor of %d.", projectedCash, state.ActiveTargets.CashFloorTarget),
		})
	}
	if nextRoundNetCash(state.Plant, state.CurrentRound) < 0 {
		warnings = append(warnings, domain.RoleReportWarning{
			Code:     "negative_next_round_maturities",
			Headline: "Next-round maturities are net cash-negative.",
			Detail:   "Near-term commitments are likely to pressure liquidity unless operations recover cash quickly.",
		})
	}
	return warnings
}

func averageCustomerSentiment(customers []domain.CustomerState) int {
	if len(customers) == 0 {
		return 0
	}
	total := 0
	for _, customer := range customers {
		total += customer.Sentiment
	}
	return total / len(customers)
}

func cloneInventorySummary(summary domain.InventoryValueSummary) *domain.InventoryValueSummary {
	cloned := summary.Clone()
	return &cloned
}
