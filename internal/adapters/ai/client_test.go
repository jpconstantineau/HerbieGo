package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/ports"
)

func TestRoutingClientDispatchesByProvider(t *testing.T) {
	ollamaClient := &stubDecisionClient{
		result: ports.ProviderDecisionResult{RawResponse: `{"ok":true}`},
	}
	router := NewRoutingClient(map[string]ports.DecisionClient{
		"ollama": ollamaClient,
	})

	result, err := router.RequestDecision(context.Background(), ports.ProviderDecisionRequest{
		Provider: "OLLAMA",
		Model:    "gemma4:e4b",
	})
	if err != nil {
		t.Fatalf("RequestDecision() error = %v", err)
	}
	if result.RawResponse != `{"ok":true}` {
		t.Fatalf("RawResponse = %q, want JSON payload", result.RawResponse)
	}
	if len(ollamaClient.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(ollamaClient.requests))
	}
}

func TestRoutingClientRejectsUnsupportedProvider(t *testing.T) {
	router := NewRoutingClient(map[string]ports.DecisionClient{
		"ollama": &stubDecisionClient{},
	})

	_, err := router.RequestDecision(context.Background(), ports.ProviderDecisionRequest{Provider: "openrouter"})
	if err == nil {
		t.Fatal("RequestDecision() error = nil, want unsupported-provider error")
	}
	if !strings.Contains(err.Error(), `unsupported AI provider "openrouter"`) {
		t.Fatalf("RequestDecision() error = %v, want unsupported-provider message", err)
	}
}

type stubDecisionClient struct {
	requests []ports.ProviderDecisionRequest
	result   ports.ProviderDecisionResult
	err      error
}

func (s *stubDecisionClient) RequestDecision(_ context.Context, request ports.ProviderDecisionRequest) (ports.ProviderDecisionResult, error) {
	s.requests = append(s.requests, request)
	if s.err != nil {
		return ports.ProviderDecisionResult{}, s.err
	}
	return s.result, nil
}
