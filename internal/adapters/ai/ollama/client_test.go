package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/ports"
)

func TestClientRequestsOpenAICompatibleStructuredResponse(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" {
			t.Fatalf("request path = %q, want /v1/chat/completions", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer ollama" {
			t.Fatalf("authorization header = %q, want Bearer ollama", got)
		}
		if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{\"contract_version\":\"herbiego.ai.v1\",\"match_id\":\"match-17\",\"round\":2,\"role_id\":\"production_manager\",\"action\":{\"production\":{\"releases\":[{\"product_id\":\"pump\",\"quantity\":1}],\"capacity_allocation\":[{\"workstation_id\":\"fabrication\",\"product_id\":\"pump\",\"capacity\":1}]}},\"commentary\":{\"public_summary\":\"Release only what fabrication can move.\",\"focus_tags\":[\"throughput\"]}}"}}]}`))
	}))
	defer server.Close()

	client, err := New(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.RequestDecision(context.Background(), ports.ProviderDecisionRequest{
		Model:        "gemma4:e4b",
		SystemPrompt: "system",
		UserPrompt:   "user",
	})
	if err != nil {
		t.Fatalf("RequestDecision() error = %v", err)
	}

	if got := requestBody["model"]; got != "gemma4:e4b" {
		t.Fatalf("request model = %#v, want gemma4:e4b", got)
	}
	assertRequestContainsPrompt(t, requestBody, "system")
	assertRequestContainsPrompt(t, requestBody, "user")
	assertResponseFormatJSON(t, requestBody)
	if result.StructuredResponse == nil {
		t.Fatal("StructuredResponse = nil, want parsed instructor result")
	}
	if got := result.StructuredResponse.Commentary.PublicSummary; got != "Release only what fabrication can move." {
		t.Fatalf("PublicSummary = %q, want parsed commentary", got)
	}
}

func TestClientReturnsHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, `{"error":"model not found"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := New(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.RequestDecision(context.Background(), ports.ProviderDecisionRequest{Model: "missing-model"})
	if err == nil {
		t.Fatal("RequestDecision() error = nil, want HTTP failure")
	}
	if !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("RequestDecision() error = %v, want HTTP status", err)
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("RequestDecision() error = %v, want response body", err)
	}
}

func TestClientPreservesConfiguredAPIPrefix(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestPath = request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{\"contract_version\":\"herbiego.ai.v1\",\"match_id\":\"match-17\",\"round\":2,\"role_id\":\"sales_manager\",\"action\":{\"sales\":{\"product_offers\":[{\"product_id\":\"pump\",\"unit_price\":16}]}},\"commentary\":{\"public_summary\":\"Holding price to protect throughput.\",\"focus_tags\":[\"throughput\"]}}"}}]}`))
	}))
	defer server.Close()

	client, err := New(
		WithBaseURL(server.URL+"/api/"),
		WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.RequestDecision(context.Background(), ports.ProviderDecisionRequest{Model: "gemma4:e4b"})
	if err != nil {
		t.Fatalf("RequestDecision() error = %v", err)
	}

	if requestPath != "/api/v1/chat/completions" {
		t.Fatalf("request path = %q, want /api/v1/chat/completions", requestPath)
	}
}

func assertRequestContainsPrompt(t *testing.T, requestBody map[string]any, prompt string) {
	t.Helper()

	messages, ok := requestBody["messages"].([]any)
	if !ok {
		t.Fatalf("messages = %#v, want array", requestBody["messages"])
	}
	for _, raw := range messages {
		message, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		content, _ := message["content"].(string)
		if strings.Contains(content, prompt) {
			return
		}
	}
	t.Fatalf("messages = %#v, want prompt %q", requestBody["messages"], prompt)
}

func assertResponseFormatJSON(t *testing.T, requestBody map[string]any) {
	t.Helper()

	format, ok := requestBody["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("response_format = %#v, want object", requestBody["response_format"])
	}
	if got := format["type"]; got != "json_object" {
		t.Fatalf("response_format.type = %#v, want json_object", got)
	}
}
