package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/ports"
)

func TestClientRequestsJSONChatCompletion(t *testing.T) {
	var requestBody chatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat/completions" {
			t.Fatalf("request path = %q, want /chat/completions", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header = %q, want Bearer test-key", got)
		}
		if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{\"contract_version\":\"herbiego.ai.v1\"}"}}]}`))
	}))
	defer server.Close()

	client, err := New(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithAPIKey("test-key"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.RequestDecision(context.Background(), ports.ProviderDecisionRequest{
		Model:           "openai/gpt-5-mini",
		SystemPrompt:    "system",
		UserPrompt:      "user",
		RequireJSONOnly: true,
	})
	if err != nil {
		t.Fatalf("RequestDecision() error = %v", err)
	}

	if requestBody.Model != "openai/gpt-5-mini" {
		t.Fatalf("request model = %q, want openai/gpt-5-mini", requestBody.Model)
	}
	if len(requestBody.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(requestBody.Messages))
	}
	if requestBody.Messages[0].Role != "system" || requestBody.Messages[0].Content != "system" {
		t.Fatalf("system message = %#v, want system prompt", requestBody.Messages[0])
	}
	if requestBody.Messages[1].Role != "user" || requestBody.Messages[1].Content != "user" {
		t.Fatalf("user message = %#v, want user prompt", requestBody.Messages[1])
	}
	if requestBody.ResponseFormat == nil || requestBody.ResponseFormat.Type != "json_object" {
		t.Fatalf("response format = %#v, want json_object", requestBody.ResponseFormat)
	}
	if result.RawResponse != `{"contract_version":"herbiego.ai.v1"}` {
		t.Fatalf("RawResponse = %q, want response body text", result.RawResponse)
	}
}

func TestClientJoinsMultipartContentResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":[{"type":"output_text","text":"{\"contract_version\":"},{"type":"output_text","text":"\"herbiego.ai.v1\"}"}]}}]}`))
	}))
	defer server.Close()

	client, err := New(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.RequestDecision(context.Background(), ports.ProviderDecisionRequest{Model: "gpt-5-mini"})
	if err != nil {
		t.Fatalf("RequestDecision() error = %v", err)
	}
	if result.RawResponse != `{"contract_version":"herbiego.ai.v1"}` {
		t.Fatalf("RawResponse = %q, want concatenated multipart content", result.RawResponse)
	}
}

func TestClientReturnsHTTPFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, `{"error":"bad api key"}`, http.StatusUnauthorized)
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
	if !strings.Contains(err.Error(), "401 Unauthorized") {
		t.Fatalf("RequestDecision() error = %v, want HTTP status", err)
	}
	if !strings.Contains(err.Error(), "bad api key") {
		t.Fatalf("RequestDecision() error = %v, want response body", err)
	}
}
