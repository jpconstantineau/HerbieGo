package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/ports"
)

const defaultBaseURL = "https://api.openai.com/v1/"

type Option func(*Client)

// Client executes provider-neutral decision requests against an
// OpenAI-compatible `/chat/completions` endpoint.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	apiKey     string
}

func WithBaseURL(rawURL string) Option {
	return func(client *Client) {
		if client != nil {
			client.baseURL, _ = parseBaseURL(rawURL)
		}
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if client != nil && httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithAPIKey(apiKey string) Option {
	return func(client *Client) {
		if client != nil {
			client.apiKey = strings.TrimSpace(apiKey)
		}
	}
}

func New(options ...Option) (*Client, error) {
	baseURL, err := parseBaseURL(defaultBaseURL)
	if err != nil {
		return nil, err
	}

	client := &Client{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
	for _, option := range options {
		if option != nil {
			option(client)
		}
	}
	if client.baseURL == nil {
		return nil, fmt.Errorf("openai client base URL is not configured")
	}
	return client, nil
}

func (c *Client) RequestDecision(ctx context.Context, request ports.ProviderDecisionRequest) (ports.ProviderDecisionResult, error) {
	if c == nil || c.baseURL == nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("openai client is not configured")
	}

	body := chatCompletionsRequest{
		Model: request.Model,
		Messages: []chatMessage{
			{Role: "system", Content: request.SystemPrompt},
			{Role: "user", Content: request.UserPrompt},
		},
	}
	if request.RequireJSONOnly {
		body.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("marshal openai request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpointURL("/chat/completions"), bytes.NewReader(payload))
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("build openai request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("call openai chat completions API: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("read openai response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return ports.ProviderDecisionResult{}, fmt.Errorf("openai chat completions API returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}

	var parsed chatCompletionsResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("decode openai response: %w", err)
	}

	content, err := parsed.firstChoiceContent()
	if err != nil {
		return ports.ProviderDecisionResult{}, err
	}

	return ports.ProviderDecisionResult{RawResponse: content}, nil
}

func (c *Client) endpointURL(path string) string {
	return c.baseURL.ResolveReference(&url.URL{Path: path}).String()
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, fmt.Errorf("openai base URL must not be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse openai base URL %q: %w", rawURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("openai base URL %q must include scheme and host", rawURL)
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}

	return parsed, nil
}

type chatCompletionsRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionsResponse struct {
	Choices []chatCompletionChoice `json:"choices"`
}

type chatCompletionChoice struct {
	Message chatCompletionMessage `json:"message"`
}

type chatCompletionMessage struct {
	Content json.RawMessage `json:"content"`
}

func (r chatCompletionsResponse) firstChoiceContent() (string, error) {
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("openai response did not include any choices")
	}

	content, err := decodeContent(r.Choices[0].Message.Content)
	if err != nil {
		return "", fmt.Errorf("decode openai response content: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("openai response choice content was empty")
	}
	return content, nil
}

func decodeContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("content was missing")
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}

	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, part := range parts {
		if strings.TrimSpace(part.Text) == "" {
			continue
		}
		builder.WriteString(part.Text)
	}
	return builder.String(), nil
}

type contentPart struct {
	Text string `json:"text"`
}
