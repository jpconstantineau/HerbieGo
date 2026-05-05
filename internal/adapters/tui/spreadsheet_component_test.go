package tui

import (
	"regexp"
	"strings"
	"testing"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestSpreadsheetComponentRendersSingleRowWithoutDuplicatingCellContent(t *testing.T) {
	component := spreadsheetComponent{
		Columns: []spreadsheetColumn{
			{Title: "Part", Width: 12},
			{Title: "Quantity", Width: 10},
		},
		Rows: [][]spreadsheetCell{{
			{Text: "Housing", Active: true},
			{Text: "4"},
		}},
	}

	view := ansiPattern.ReplaceAllString(component.Render(), "")
	if got := strings.Count(view, "Housing"); got != 1 {
		t.Fatalf("Housing count = %d, want 1\n%s", got, view)
	}
	if got := strings.Count(view, "4"); got != 1 {
		t.Fatalf("quantity count = %d, want 1\n%s", got, view)
	}
}

func TestSpreadsheetComponentShowsActivePlaceholderRowWhenEmpty(t *testing.T) {
	component := spreadsheetComponent{
		Columns: []spreadsheetColumn{
			{Title: "[Workstation]", Width: 18},
			{Title: "Product", Width: 18},
		},
		Rows: [][]spreadsheetCell{{
			{Placeholder: "Choose workstation", Active: true},
			{Placeholder: "Choose product"},
		}},
	}

	view := ansiPattern.ReplaceAllString(component.Render(), "")
	if !strings.Contains(view, "Choose workstation") {
		t.Fatalf("empty spreadsheet missing active placeholder row\n%s", view)
	}
}

func TestFitSpreadsheetTextOnlyAddsEllipsisWhenNeeded(t *testing.T) {
	if got := fitSpreadsheetText("Valve", 10); got != "Valve" {
		t.Fatalf("fitSpreadsheetText short text = %q, want Valve", got)
	}
	if got := fitSpreadsheetText("Fabrication", 8); got != "Fabri..." {
		t.Fatalf("fitSpreadsheetText long text = %q, want Fabri...", got)
	}
}
