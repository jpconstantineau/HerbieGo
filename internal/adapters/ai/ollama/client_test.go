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

func TestClientRequestsNonStreamingJSONResponse(t *testing.T) {
	var requestBody generateRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/generate" {
			t.Fatalf("request path = %q, want /api/generate", request.URL.Path)
		}
		if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"response":"{\"contract_version\":\"herbiego.ai.v1\"}"}`))
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
		Model:           "gemma4:e4b",
		SystemPrompt:    "system",
		UserPrompt:      "user",
		RequireJSONOnly: true,
	})
	if err != nil {
		t.Fatalf("RequestDecision() error = %v", err)
	}

	if requestBody.Model != "gemma4:e4b" {
		t.Fatalf("request model = %q, want gemma4:e4b", requestBody.Model)
	}
	if requestBody.System != "system" {
		t.Fatalf("request system = %q, want system", requestBody.System)
	}
	if requestBody.Prompt != "user" {
		t.Fatalf("request prompt = %q, want user", requestBody.Prompt)
	}
	if requestBody.Format != "json" {
		t.Fatalf("request format = %v, want json", requestBody.Format)
	}
	if requestBody.Stream {
		t.Fatal("request stream = true, want false")
	}
	if result.RawResponse != `{"contract_version":"herbiego.ai.v1"}` {
		t.Fatalf("RawResponse = %q, want returned response text", result.RawResponse)
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
		_, _ = writer.Write([]byte(`{"response":"{\"contract_version\":\"herbiego.ai.v1\"}"}`))
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

	if requestPath != "/api/generate" {
		t.Fatalf("request path = %q, want /api/generate", requestPath)
	}
}
