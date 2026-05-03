package domain

import "slices"

// RoleRoundReport is the player-facing report delivered alongside the round
// view at the start of a round.
type RoleRoundReport struct {
	Companywide   CompanywidePerformanceReport
	Department    DepartmentPerformanceReport
	BonusReminder string
}

type RoleReportSectionKind string

const (
	RoleReportSectionExecutiveSummary   RoleReportSectionKind = "executive_summary"
	RoleReportSectionOperatingPicture   RoleReportSectionKind = "operating_picture"
	RoleReportSectionConstraintRiskView RoleReportSectionKind = "constraint_risk_view"
	RoleReportSectionTradeoffPressure   RoleReportSectionKind = "tradeoff_pressure_view"
	RoleReportSectionDecisionPrompts    RoleReportSectionKind = "decision_prompts"
	RoleReportSectionInventoryRisk      RoleReportSectionKind = "inventory_risk"
	RoleReportSectionBacklogHealth      RoleReportSectionKind = "backlog_health"
	RoleReportSectionServiceRisk        RoleReportSectionKind = "service_risk"
	RoleReportSectionMarginVariance     RoleReportSectionKind = "margin_variance"
	RoleReportSectionCapacityPressure   RoleReportSectionKind = "capacity_pressure"
	RoleReportSectionDemandHealth       RoleReportSectionKind = "demand_health"
	RoleReportSectionCashDebtPressure   RoleReportSectionKind = "cash_debt_pressure"
	RoleReportSectionOperationalNotes   RoleReportSectionKind = "operational_notes"
)

type CompanywidePerformanceReport struct {
	Sections []RoleReportSection
}

type DepartmentPerformanceReport struct {
	RoleID        RoleID
	Sections      []RoleReportSection
	BonusSummary  string
	FocusQuestion string
}

type RoleReportSection struct {
	Kind          RoleReportSectionKind
	Title         string
	DecisionFocus string
	Summary       []string
	Metrics       []RoleReportMetric
	Facts         []string
	ProductValues []ProductValueSummary
	ProductUnits  []ProductUnitSummary
	Inventory     *InventoryValueSummary
	Financials    []ProductFinancialSummary
	Warnings      []RoleReportWarning
	Tradeoffs     []RoleReportTradeoff
	Prompts       []RoleReportPrompt
}

type RoleReportMetric struct {
	Metric         MetricValue
	Label          string
	Interpretation string
}

type RoleReportWarning struct {
	Code     string
	Headline string
	Detail   string
}

type RoleReportTradeoff struct {
	Focus    string
	Tension  string
	Guidance string
}

type RoleReportPrompt struct {
	Question     string
	WhyItMatters string
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
		Sections: cloneSlice(r.Sections, RoleReportSection.Clone),
	}
}

func (r DepartmentPerformanceReport) Clone() DepartmentPerformanceReport {
	return DepartmentPerformanceReport{
		RoleID:        r.RoleID,
		Sections:      cloneSlice(r.Sections, RoleReportSection.Clone),
		BonusSummary:  r.BonusSummary,
		FocusQuestion: r.FocusQuestion,
	}
}

func (s InventoryValueSummary) Clone() InventoryValueSummary {
	return InventoryValueSummary{
		TotalValue: s.TotalValue,
		Detail:     slices.Clone(s.Detail),
	}
}

func (s RoleReportSection) Clone() RoleReportSection {
	return RoleReportSection{
		Kind:          s.Kind,
		Title:         s.Title,
		DecisionFocus: s.DecisionFocus,
		Summary:       slices.Clone(s.Summary),
		Metrics:       cloneSlice(s.Metrics, RoleReportMetric.Clone),
		Facts:         slices.Clone(s.Facts),
		ProductValues: slices.Clone(s.ProductValues),
		ProductUnits:  slices.Clone(s.ProductUnits),
		Inventory:     clonePtr(s.Inventory, InventoryValueSummary.Clone),
		Financials:    slices.Clone(s.Financials),
		Warnings:      cloneSlice(s.Warnings, RoleReportWarning.Clone),
		Tradeoffs:     cloneSlice(s.Tradeoffs, RoleReportTradeoff.Clone),
		Prompts:       cloneSlice(s.Prompts, RoleReportPrompt.Clone),
	}
}

func (m RoleReportMetric) Clone() RoleReportMetric {
	return RoleReportMetric{
		Metric:         m.Metric,
		Label:          m.Label,
		Interpretation: m.Interpretation,
	}
}

func (w RoleReportWarning) Clone() RoleReportWarning {
	return w
}

func (t RoleReportTradeoff) Clone() RoleReportTradeoff {
	return t
}

func (p RoleReportPrompt) Clone() RoleReportPrompt {
	return p
}
