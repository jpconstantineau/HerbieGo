package domain

import "slices"

// RoleRoundReport is the player-facing report delivered alongside the round
// view at the start of a round.
type RoleRoundReport struct {
	Companywide   CompanywidePerformanceReport
	Department    DepartmentPerformanceReport
	BonusReminder string
}

type CompanywidePerformanceReport struct {
	NewSales                 []ProductValueSummary
	UnshippedSales           []ProductValueSummary
	SalesAtRisk              []ProductValueSummary
	ProductsProducedLastWeek []ProductUnitSummary
	CurrentInventoryLevels   InventoryValueSummary
	Financials               []ProductFinancialSummary
}

type DepartmentPerformanceReport struct {
	RoleID       RoleID
	KeyMetrics   []MetricValue
	DetailLines  []string
	BonusSummary string
}

type ProductValueSummary struct {
	ProductID   ProductID
	Count       Units
	TotalValue  Money
	Description string
}

type ProductUnitSummary struct {
	ProductID   ProductID
	Count       Units
	Description string
}

type InventoryValueSummary struct {
	TotalValue Money
	Detail     []InventoryBucketValue
}

type InventoryBucketValue struct {
	Bucket     string
	TotalValue Money
}

type ProductFinancialSummary struct {
	ProductID          ProductID
	Revenue            Money
	ProductionCost     Money
	PartsCost          Money
	ContributionMargin Money
}

func (r RoleRoundReport) Clone() RoleRoundReport {
	return RoleRoundReport{
		Companywide:   r.Companywide.Clone(),
		Department:    r.Department.Clone(),
		BonusReminder: r.BonusReminder,
	}
}

func (r CompanywidePerformanceReport) Clone() CompanywidePerformanceReport {
	return CompanywidePerformanceReport{
		NewSales:                 slices.Clone(r.NewSales),
		UnshippedSales:           slices.Clone(r.UnshippedSales),
		SalesAtRisk:              slices.Clone(r.SalesAtRisk),
		ProductsProducedLastWeek: slices.Clone(r.ProductsProducedLastWeek),
		CurrentInventoryLevels:   r.CurrentInventoryLevels.Clone(),
		Financials:               slices.Clone(r.Financials),
	}
}

func (r DepartmentPerformanceReport) Clone() DepartmentPerformanceReport {
	return DepartmentPerformanceReport{
		RoleID:       r.RoleID,
		KeyMetrics:   slices.Clone(r.KeyMetrics),
		DetailLines:  slices.Clone(r.DetailLines),
		BonusSummary: r.BonusSummary,
	}
}

func (s InventoryValueSummary) Clone() InventoryValueSummary {
	return InventoryValueSummary{
		TotalValue: s.TotalValue,
		Detail:     slices.Clone(s.Detail),
	}
}
