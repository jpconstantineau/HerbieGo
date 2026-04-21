package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/ports"
)

// RoutingClient dispatches provider-neutral decision requests to the concrete
// adapter registered for the requested provider name.
type RoutingClient struct {
	providers map[string]ports.DecisionClient
}

func NewRoutingClient(providers map[string]ports.DecisionClient) *RoutingClient {
	normalized := make(map[string]ports.DecisionClient, len(providers))
	for provider, client := range providers {
		name := normalizeProvider(provider)
		if name == "" || client == nil {
			continue
		}
		normalized[name] = client
	}
	return &RoutingClient{providers: normalized}
}

func (c *RoutingClient) SupportsProvider(provider string) bool {
	if c == nil {
		return false
	}
	_, ok := c.providers[normalizeProvider(provider)]
	return ok
}

func (c *RoutingClient) RequestDecision(ctx context.Context, request ports.ProviderDecisionRequest) (ports.ProviderDecisionResult, error) {
	if c == nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("ai routing client is not configured")
	}

	client, ok := c.providers[normalizeProvider(request.Provider)]
	if !ok {
		return ports.ProviderDecisionResult{}, fmt.Errorf("unsupported AI provider %q", request.Provider)
	}

	return client.RequestDecision(ctx, request)
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
