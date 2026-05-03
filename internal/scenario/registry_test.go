package scenario

import "testing"

func TestDefaultScenarioIsRegistered(t *testing.T) {
	definition, ok := Lookup(DefaultID)
	if !ok {
		t.Fatalf("Lookup(%q) = false, want registered default scenario", DefaultID)
	}
	if got := definition.ID; got != StarterID {
		t.Fatalf("definition.ID = %q, want %q", got, StarterID)
	}
}

func TestRegisteredIDsIncludesStarter(t *testing.T) {
	ids := RegisteredIDs()
	if len(ids) == 0 {
		t.Fatal("RegisteredIDs() = empty, want at least starter")
	}
	if ids[0] != StarterID {
		t.Fatalf("RegisteredIDs()[0] = %q, want %q", ids[0], StarterID)
	}
}
