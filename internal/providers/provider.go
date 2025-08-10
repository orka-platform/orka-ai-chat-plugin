package providers

import "context"

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"maxTokens,omitempty"`
	TopP        float32       `json:"topP,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type TextRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Temperature float32  `json:"temperature,omitempty"`
	MaxTokens   int      `json:"maxTokens,omitempty"`
	TopP        float32  `json:"topP,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (map[string]any, error)
	Complete(ctx context.Context, req TextRequest) (map[string]any, error)
}
