package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	openaiadapter "github.com/jpconstantineau/herbiego/internal/adapters/ai/openai"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

const (
	defaultBaseURL = "http://localhost:11434/"
	defaultAPIKey  = "ollama"
)

type Option func(*Client)

// Client executes provider-neutral decision requests against Ollama's
// OpenAI-compatible chat completions endpoint.
type Client struct {
	inner *openaiadapter.Client
}

func WithBaseURL(rawURL string) Option {
	return func(client *Client) {
		if client != nil && client.inner != nil {
			_ = applyOption(client.inner, openaiadapter.WithBaseURL(normalizeBaseURL(rawURL)))
		}
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if client != nil && client.inner != nil && httpClient != nil {
			_ = applyOption(client.inner, openaiadapter.WithHTTPClient(httpClient))
		}
	}
}

func WithAPIKey(apiKey string) Option {
	return func(client *Client) {
		if client != nil && client.inner != nil {
			trimmed := strings.TrimSpace(apiKey)
			if trimmed == "" {
				trimmed = defaultAPIKey
			}
			_ = applyOption(client.inner, openaiadapter.WithAPIKey(trimmed))
		}
	}
}

func New(options ...Option) (*Client, error) {
	inner, err := openaiadapter.New(
		openaiadapter.WithBaseURL(normalizeBaseURL(defaultBaseURL)),
		openaiadapter.WithAPIKey(defaultAPIKey),
	)
	if err != nil {
		return nil, err
	}

	client := &Client{inner: inner}
	for _, option := range options {
		if option != nil {
			option(client)
		}
	}
	if client.inner == nil {
		return nil, fmt.Errorf("ollama client is not configured")
	}
	return client, nil
}

func (c *Client) RequestDecision(ctx context.Context, request ports.ProviderDecisionRequest) (ports.ProviderDecisionResult, error) {
	if c == nil || c.inner == nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("ollama client is not configured")
	}
	return c.inner.RequestDecision(ctx, request)
}

func normalizeBaseURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return defaultBaseURL + "v1/"
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	cleanPath := strings.TrimSuffix(parsed.Path, "/")
	if !strings.HasSuffix(cleanPath, "/v1") {
		if cleanPath == "" {
			cleanPath = "/v1"
		} else {
			cleanPath += "/v1"
		}
	}
	parsed.Path = cleanPath + "/"
	return parsed.String()
}

func applyOption(client *openaiadapter.Client, option openaiadapter.Option) error {
	if client == nil || option == nil {
		return nil
	}
	option(client)
	return nil
}
