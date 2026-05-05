package tui

import (
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/actionschema"
	"github.com/jpconstantineau/herbiego/internal/domain"
)

func TestActionFormModelSupportsCollectionNavigationAndChoiceCycling(t *testing.T) {
	model := newActionFormModel(actionschema.RoleSchema{
		RoleID: domain.RoleProcurementManager,
		Fields: []actionschema.FieldSpec{
			{
				ID:    "orders",
				Label: "Orders",
				Collection: &actionschema.CollectionSpec{
					Columns: []actionschema.ColumnSpec{
						{
							ID:       "part_id",
							Label:    "Part",
							Kind:     actionschema.ValueKindChoice,
							Required: true,
							Options: actionschema.OptionSource{Static: []actionschema.Option{
								{Value: "housing", Label: "Housing"},
								{Value: "seal_kit", Label: "Seal Kit"},
							}},
						},
						{
							ID:       "quantity",
							Label:    "Quantity",
							Kind:     actionschema.ValueKindInteger,
							Required: true,
						},
					},
				},
			},
			{ID: "commentary", Label: "Commentary", Kind: actionschema.ValueKindText},
		},
	})

	model.AddRow()
	if got := len(model.Values["orders"].Rows); got != 1 {
		t.Fatalf("len(rows) = %d, want 1", got)
	}

	if !model.CycleChoice(1) {
		t.Fatalf("CycleChoice(1) = false, want true")
	}
	if got := model.Values["orders"].Rows[0]["part_id"]; got != "housing" {
		t.Fatalf("part_id after cycle = %q, want housing", got)
	}

	model.MoveRight()
	if got := model.ColumnIndex; got != 1 {
		t.Fatalf("ColumnIndex = %d, want 1", got)
	}
	if !model.BeginEdit() {
		t.Fatalf("BeginEdit() = false, want true")
	}
	if !model.TypeRunes("4") {
		t.Fatalf("TypeRunes() = false, want true")
	}
	if !model.CommitEdit() {
		t.Fatalf("CommitEdit() = false, want true")
	}
	if got := model.Values["orders"].Rows[0]["quantity"]; got != "4" {
		t.Fatalf("quantity = %q, want 4", got)
	}

	model.MoveDown()
	if got := model.FieldIndex; got != 1 {
		t.Fatalf("FieldIndex = %d, want 1", got)
	}
}

func TestActionFormModelRendersEditCursorForScalarAndCollectionValues(t *testing.T) {
	model := newActionFormModel(actionschema.RoleSchema{
		RoleID: domain.RoleSalesManager,
		Fields: []actionschema.FieldSpec{
			{ID: "commentary", Label: "Commentary", Kind: actionschema.ValueKindText, Placeholder: "Explain"},
			{
				ID:    "offers",
				Label: "Offers",
				Collection: &actionschema.CollectionSpec{
					Columns: []actionschema.ColumnSpec{
						{ID: "unit_price", Label: "Unit price", Kind: actionschema.ValueKindInteger, Placeholder: "0"},
					},
				},
			},
		},
	})

	if !model.BeginEdit() {
		t.Fatalf("BeginEdit() = false, want true")
	}
	model.TypeRunes("Hold price")
	if got := model.displayScalar(model.Schema.Fields[0]); got != "Hold price|" {
		t.Fatalf("displayScalar() = %q, want cursor suffix", got)
	}
	model.CommitEdit()

	model.MoveDown()
	model.AddRow()
	if !model.BeginEdit() {
		t.Fatalf("BeginEdit() for collection = false, want true")
	}
	model.TypeRunes("12")
	if got := model.displayCell(model.Schema.Fields[1], 0, model.Schema.Fields[1].Collection.Columns[0]); got != "12|" {
		t.Fatalf("displayCell() = %q, want cursor suffix", got)
	}
}

func TestActionFormModelRendersAllRowsInThreeRowCollectionTable(t *testing.T) {
	model := newActionFormModel(actionschema.RoleSchema{
		RoleID: domain.RoleProductionManager,
		Fields: []actionschema.FieldSpec{
			{
				ID:    "releases",
				Label: "Releases",
				Collection: &actionschema.CollectionSpec{
					Columns: []actionschema.ColumnSpec{
						{
							ID:       "product_id",
							Label:    "Product",
							Kind:     actionschema.ValueKindChoice,
							Required: true,
							Options: actionschema.OptionSource{Static: []actionschema.Option{
								{Value: "pump", Label: "Pump"},
								{Value: "valve", Label: "Valve"},
							}},
						},
						{ID: "quantity", Label: "Quantity", Kind: actionschema.ValueKindInteger, Required: true, Placeholder: "0"},
					},
				},
			},
		},
	})

	model.AddRow()
	model.CycleChoice(1)
	model.MoveRight()
	model.BeginEdit()
	model.TypeRunes("34")
	model.CommitEdit()

	model.AddRow()
	model.CycleChoice(1)
	model.CycleChoice(1)
	model.MoveRight()
	model.BeginEdit()
	model.TypeRunes("23")
	model.CommitEdit()

	model.AddRow()
	model.CycleChoice(1)
	model.MoveRight()
	model.BeginEdit()
	model.TypeRunes("11")
	model.CommitEdit()

	view := model.renderedCollectionTable(model.Schema.Fields[0], 80, true)
	for _, want := range []string{"Pump", "34", "Valve", "23", "11"} {
		if !strings.Contains(view, want) {
			t.Fatalf("table view missing %q\n%s", want, view)
		}
	}
}
