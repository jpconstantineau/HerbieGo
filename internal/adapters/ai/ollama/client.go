package ollama

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

const defaultBaseURL = "http://localhost:11434/"

type Option func(*Client)

// Client executes provider-neutral decision requests against the Ollama
// `/api/generate` endpoint.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
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
		return nil, fmt.Errorf("ollama client base URL is not configured")
	}
	return client, nil
}

func (c *Client) RequestDecision(ctx context.Context, request ports.ProviderDecisionRequest) (ports.ProviderDecisionResult, error) {
	if c == nil || c.baseURL == nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("ollama client is not configured")
	}

	body := generateRequest{
		Model:  strings.TrimSpace(request.Model),
		Prompt: request.UserPrompt,
		System: request.SystemPrompt,
		Stream: false,
	}
	if request.RequireJSONOnly {
		body.Format = "json"
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("marshal ollama request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpointURL("/api/generate"), bytes.NewReader(payload))
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("build ollama request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("call ollama generate API: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("read ollama response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return ports.ProviderDecisionResult{}, fmt.Errorf("ollama generate API returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}

	var parsed generateResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("decode ollama response: %w", err)
	}

	return ports.ProviderDecisionResult{
		RawResponse: parsed.Response,
	}, nil
}

func (c *Client) endpointURL(path string) string {
	endpoint := *c.baseURL
	basePath := strings.TrimSuffix(endpoint.Path, "/")
	cleanPath := strings.TrimLeft(path, "/")
	if trimmedBase := strings.TrimLeft(basePath, "/"); trimmedBase != "" {
		prefix := trimmedBase + "/"
		cleanPath = strings.TrimPrefix(cleanPath, prefix)
	}
	switch {
	case basePath == "":
		endpoint.Path = "/" + cleanPath
	case cleanPath == "":
		endpoint.Path = basePath
	default:
		endpoint.Path = basePath + "/" + cleanPath
	}
	return endpoint.String()
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, fmt.Errorf("ollama base URL must not be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse ollama base URL %q: %w", rawURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("ollama base URL %q must include scheme and host", rawURL)
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}

	return parsed, nil
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system,omitempty"`
	Format any    `json:"format,omitempty"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}
