package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/ports"
	instructorcore "github.com/jxnl/instructor-go/pkg/instructor/core"
	instructoropenai "github.com/jxnl/instructor-go/pkg/instructor/providers/openai"
	openaiSDK "github.com/sashabaranov/go-openai"
)

const defaultBaseURL = "https://api.openai.com/v1/"

type Option func(*Client)

// Client executes provider-neutral decision requests against an OpenAI-compatible
// chat completions endpoint using instructor-go for structured extraction.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	apiKey     string
	maxRetries int
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

func WithMaxRetries(maxRetries int) Option {
	return func(client *Client) {
		if client != nil {
			client.maxRetries = maxRetries
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
		maxRetries: instructorcore.DefaultMaxRetries,
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

	cfg := openaiSDK.DefaultConfig(c.apiKey)
	cfg.BaseURL = strings.TrimSuffix(c.baseURL.String(), "/")
	cfg.HTTPClient = c.httpClient

	instructorClient := instructoropenai.FromOpenAI(
		openaiSDK.NewClientWithConfig(cfg),
		instructorcore.WithMode(instructorcore.ModeJSON),
		instructorcore.WithMaxRetries(max(0, c.maxRetries)),
	)

	var envelope ports.AIDecisionEnvelope
	response, err := instructorClient.CreateChatCompletion(
		ctx,
		openaiSDK.ChatCompletionRequest{
			Model: request.Model,
			Messages: []openaiSDK.ChatCompletionMessage{
				{Role: openaiSDK.ChatMessageRoleSystem, Content: request.SystemPrompt},
				{Role: openaiSDK.ChatMessageRoleUser, Content: request.UserPrompt},
			},
		},
		&envelope,
	)
	if err != nil {
		return ports.ProviderDecisionResult{}, fmt.Errorf("call openai chat completions API: %w", err)
	}

	content, err := firstChoiceContent(response)
	if err != nil {
		return ports.ProviderDecisionResult{}, err
	}

	return ports.ProviderDecisionResult{
		RawResponse:        content,
		StructuredResponse: &envelope,
	}, nil
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

func firstChoiceContent(response openaiSDK.ChatCompletionResponse) (string, error) {
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("openai response did not include any choices")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("openai response choice content was empty")
	}
	return content, nil
}
