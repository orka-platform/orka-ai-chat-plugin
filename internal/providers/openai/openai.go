package openai

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	p "orka-ai-chat-plugin/internal/providers"

	oa "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared"
)

type Client struct {
	apiKey string
	client oa.Client
}

func New(apiKey string) *Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 60 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}
	httpClient := &http.Client{Transport: transport, Timeout: 120 * time.Second}
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.openai.com/v1"),
		option.WithHTTPClient(httpClient),
		option.WithMaxRetries(5),
	}
	return &Client{
		apiKey: apiKey,
		client: oa.NewClient(opts...),
	}
}

func (c *Client) Chat(ctx context.Context, req p.ChatRequest) (map[string]any, error) {
	// Convert to SDK params
	messages := make([]oa.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			messages = append(messages, oa.SystemMessage(m.Content))
		case "assistant":
			messages = append(messages, oa.AssistantMessage(m.Content))
		default:
			messages = append(messages, oa.UserMessage(m.Content))
		}
	}
	params := oa.ChatCompletionNewParams{
		Model:    shared.ChatModel(req.Model),
		Messages: messages,
	}
	if req.Temperature != 0 {
		params.Temperature = oa.Float(float64(req.Temperature))
	}
	if req.MaxTokens != 0 {
		params.MaxTokens = oa.Int(int64(req.MaxTokens))
	}
	if req.TopP != 0 {
		params.TopP = oa.Float(float64(req.TopP))
	}

	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai chat error: %w", err)
	}

	result := map[string]any{}
	result["id"] = resp.ID
	result["model"] = resp.Model
	if len(resp.Choices) > 0 {
		ch := resp.Choices[0]
		result["finishReason"] = ch.FinishReason
		result["content"] = ch.Message.Content
		result["role"] = ch.Message.Role
	}
	result["usage"] = map[string]any{
		"promptTokens":     resp.Usage.PromptTokens,
		"completionTokens": resp.Usage.CompletionTokens,
		"totalTokens":      resp.Usage.TotalTokens,
	}
	return result, nil
}

func (c *Client) Complete(ctx context.Context, req p.TextRequest) (map[string]any, error) {
	// Implement Complete via Chat with a single user message for simplicity and consistency
	return c.Chat(ctx, p.ChatRequest{
		Model:       req.Model,
		Messages:    []p.ChatMessage{{Role: "user", Content: req.Prompt}},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stop:        req.Stop,
	})
}

var _ p.Provider = (*Client)(nil)
