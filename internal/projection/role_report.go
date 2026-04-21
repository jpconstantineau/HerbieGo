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

	return domain.CompanywidePerformanceReport{
		NewSales:                 newSalesSummary(latestRound, ok),
		UnshippedSales:           backlogSummary(state.Plant.Backlog),
		SalesAtRisk:              backlogAtRiskSummary(state.Plant.Backlog),
		ProductsProducedLastWeek: producedSummary(latestRound, ok),
		CurrentInventoryLevels:   inventorySummary(state.Plant),
		Financials:               financialSummary(latestRound, ok),
	}
}

func buildDepartmentPerformanceReport(state domain.MatchState, viewerRoleID domain.RoleID) domain.DepartmentPerformanceReport {
	switch viewerRoleID {
	case domain.RoleProcurementManager:
		return domain.DepartmentPerformanceReport{
			RoleID: viewerRoleID,
			KeyMetrics: []domain.MetricValue{
				{MetricID: "ordered_parts", Value: int(sumInTransitSupply(state.Plant.InTransitSupply)), DisplayUnit: "units"},
				{MetricID: "parts_on_hand", Value: int(state.Metrics.PartsOnHandUnits), DisplayUnit: "units"},
			},
			DetailLines: []string{
				fmt.Sprintf("In-transit supply totals %d units across open purchase orders.", sumInTransitSupply(state.Plant.InTransitSupply)),
				fmt.Sprintf("On-hand parts inventory value is %d.", inventoryBucketTotal(state.Plant.PartsInventory)),
			},
			BonusSummary: bonusReminder(viewerRoleID),
		}
	case domain.RoleProductionManager:
		return domain.DepartmentPerformanceReport{
			RoleID: viewerRoleID,
			KeyMetrics: []domain.MetricValue{
				{MetricID: "wip_units", Value: int(sumWIPUnits(state.Plant.WIPInventory)), DisplayUnit: "units"},
				{MetricID: "output_units", Value: int(state.Metrics.ProductionOutputUnits), DisplayUnit: "units"},
			},
			DetailLines: []string{
				fmt.Sprintf("Current WIP totals %d units across active workstations.", sumWIPUnits(state.Plant.WIPInventory)),
				fmt.Sprintf("Finished goods on hand total %d units.", state.Metrics.FinishedGoodsUnits),
			},
			BonusSummary: bonusReminder(viewerRoleID),
		}
	case domain.RoleSalesManager:
		return domain.DepartmentPerformanceReport{
			RoleID: viewerRoleID,
			KeyMetrics: []domain.MetricValue{
				{MetricID: "sales_pipeline", Value: int(sumBacklogUnits(state.Plant.Backlog)), DisplayUnit: "units"},
				{MetricID: "throughput_revenue", Value: int(state.Metrics.ThroughputRevenue), DisplayUnit: "money"},
			},
			DetailLines: []string{
				fmt.Sprintf("Open backlog totals %d units awaiting shipment.", sumBacklogUnits(state.Plant.Backlog)),
				fmt.Sprintf("Latest realized throughput revenue is %d.", state.Metrics.ThroughputRevenue),
			},
			BonusSummary: bonusReminder(viewerRoleID),
		}
	case domain.RoleFinanceController:
		return domain.DepartmentPerformanceReport{
			RoleID: viewerRoleID,
			KeyMetrics: []domain.MetricValue{
				{MetricID: "margin", Value: int(state.Metrics.RoundProfit), DisplayUnit: "money"},
				{MetricID: "cash_position", Value: int(state.Plant.Cash), DisplayUnit: "money"},
			},
			DetailLines: []string{
				fmt.Sprintf("Current cash is %d against debt ceiling %d.", state.Plant.Cash, state.Plant.DebtCeiling),
				fmt.Sprintf("Round profit most recently closed at %d.", state.Metrics.RoundProfit),
			},
			BonusSummary: bonusReminder(viewerRoleID),
		}
	default:
		return domain.DepartmentPerformanceReport{
			RoleID:       viewerRoleID,
			BonusSummary: bonusReminder(viewerRoleID),
		}
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
