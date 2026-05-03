package scenario

import (
	"fmt"
	"slices"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

const DefaultID = StarterID

var registry = map[domain.ScenarioID]Definition{}

// Register adds a scenario definition to the process registry.
func Register(definition Definition) error {
	if definition.ID == "" {
		return fmt.Errorf("scenario id must not be empty")
	}
	if _, exists := registry[definition.ID]; exists {
		return fmt.Errorf("scenario %q already registered", definition.ID)
	}
	registry[definition.ID] = definition
	return nil
}

// MustRegister adds a scenario definition to the process registry and panics on error.
func MustRegister(definition Definition) {
	if err := Register(definition); err != nil {
		panic(err)
	}
}

// Lookup resolves one scenario definition from the registry.
func Lookup(id domain.ScenarioID) (Definition, bool) {
	definition, ok := registry[id]
	return definition, ok
}

// Default returns the configured default scenario definition.
func Default() Definition {
	return MustLookup(DefaultID)
}

// MustLookup resolves one scenario definition from the registry and panics when missing.
func MustLookup(id domain.ScenarioID) Definition {
	definition, ok := Lookup(id)
	if !ok {
		panic(fmt.Sprintf("scenario %q is not registered", id))
	}
	return definition
}

// RegisteredIDs returns the known scenario ids in stable order.
func RegisteredIDs() []domain.ScenarioID {
	ids := make([]domain.ScenarioID, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
