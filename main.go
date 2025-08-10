package main

import (
	"context"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"strings"
	"time"

	"orka-ai-chat-plugin/internal/providers"
	"orka-ai-chat-plugin/internal/providers/openai"

	sdk "github.com/orka-platform/orka-plugin-sdk"
)

// LlmPlugin implements Orka's RPC service for LLM operations.
// Methods: Chat, Complete
type LlmPlugin struct{}

func init() {
	gob.Register(map[string]any{})
	gob.Register([]any{})
	gob.Register(map[string]string{})
	gob.Register([]string{})
}

func (p *LlmPlugin) CallMethod(req sdk.Request, res *sdk.Response) error {
	method := normalizeMethod(req.Method)
	switch method {
	case "Chat":
		out, err := handleChat(req.Args)
		if err != nil {
			*res = sdk.Response{Success: false, Error: err.Error()}
			return nil
		}
		*res = sdk.Response{Success: true, Data: out}
		return nil
	case "Complete":
		out, err := handleComplete(req.Args)
		if err != nil {
			*res = sdk.Response{Success: false, Error: err.Error()}
			return nil
		}
		*res = sdk.Response{Success: true, Data: out}
		return nil
	default:
		*res = sdk.Response{Success: false, Error: fmt.Sprintf("unknown method: %s", req.Method)}
		return nil
	}
}

func normalizeMethod(m string) string {
	if m == "" {
		return m
	}
	return strings.ToUpper(m[:1]) + m[1:]
}

func handleChat(args map[string]any) (map[string]any, error) {
	model := getString(args, "model", "")
	if model == "" {
		return nil, errors.New("missing required arg: model")
	}
	temperature := getFloat32(args, "temperature", 0)
	maxTokens := getInt(args, "maxTokens", 0)
	topP := getFloat32(args, "topP", 0)
	stop := getStringSlice(args, "stop")
	stream := getBool(args, "stream", false)

	messages := extractMessages(args)
	if len(messages) == 0 {
		// fallback: build from prompt if provided
		prompt := getString(args, "prompt", "")
		if prompt == "" {
			return nil, errors.New("either messages[] or prompt is required")
		}
		messages = []providers.ChatMessage{{Role: "user", Content: prompt}}
	}

	prov, err := makeProviderFromModel(model, args)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := prov.Chat(ctx, providers.ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		TopP:        topP,
		Stop:        stop,
		Stream:      stream,
	})
	if err != nil {
		return nil, err
	}
	return filterChatOutput(out), nil
}

func handleComplete(args map[string]any) (map[string]any, error) {
	model := getString(args, "model", "")
	if model == "" {
		return nil, errors.New("missing required arg: model")
	}
	prompt := getString(args, "prompt", "")
	if prompt == "" {
		return nil, errors.New("missing required arg: prompt")
	}
	temperature := getFloat32(args, "temperature", 0)
	maxTokens := getInt(args, "maxTokens", 0)
	topP := getFloat32(args, "topP", 0)
	stop := getStringSlice(args, "stop")

	prov, err := makeProviderFromModel(model, args)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := prov.Complete(ctx, providers.TextRequest{
		Model:       model,
		Prompt:      prompt,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		TopP:        topP,
		Stop:        stop,
	})
	if err != nil {
		return nil, err
	}
	return filterCompleteOutput(out), nil
}

func makeProvider(name string, args map[string]any) (providers.Provider, error) { // backwards compatibility
	return makeProviderFromModel(name, args)
}

func makeProviderFromModel(model string, args map[string]any) (providers.Provider, error) {
	lower := strings.ToLower(model)
	// Detect OpenAI models; extend with more providers later
	if strings.HasPrefix(lower, "gpt-") || strings.Contains(lower, "gpt") || strings.HasPrefix(lower, "o1") {
		apiKey := getString(args, "apiKey", "")
		if apiKey == "" {
			return nil, errors.New("missing OpenAI API key: pass apiKey argument")
		}
		return openai.New(apiKey), nil
	}
	return nil, fmt.Errorf("unsupported model: %s", model)
}

// Arg helpers
func getString(m map[string]any, key string, def string) string {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		}
	}
	// normalization: also check ID vs Id
	if strings.HasSuffix(key, "ID") {
		if v, ok := m[strings.TrimSuffix(key, "ID")+"Id"]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return def
}

func getInt(m map[string]any, key string, def int) int {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case int:
			return t
		case float64:
			return int(t)
		case string:
			if t == "" {
				break
			}
			if i, err := strconv.Atoi(t); err == nil {
				return i
			}
		}
	}
	return def
}

func getFloat32(m map[string]any, key string, def float32) float32 {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case float32:
			return t
		case float64:
			return float32(t)
		case string:
			if t == "" {
				break
			}
			if f, err := strconv.ParseFloat(t, 32); err == nil {
				return float32(f)
			}
		}
	}
	return def
}

func getBool(m map[string]any, key string, def bool) bool {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case bool:
			return t
		case string:
			if t == "" {
				break
			}
			if b, err := strconv.ParseBool(t); err == nil {
				return b
			}
		}
	}
	return def
}

func getStringSlice(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []any:
		out := make([]string, 0, len(arr))
		for _, it := range arr {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return arr
	case string:
		if strings.TrimSpace(arr) == "" {
			return nil
		}
		// comma-separated string
		parts := strings.Split(arr, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func extractMessages(args map[string]any) []providers.ChatMessage {
	v, ok := args["messages"]
	if !ok || v == nil {
		return nil
	}
	out := []providers.ChatMessage{}
	switch arr := v.(type) {
	case []any:
		for _, it := range arr {
			if m, ok := it.(map[string]any); ok {
				role := getString(m, "role", "")
				content := getString(m, "content", "")
				if role != "" && content != "" {
					out = append(out, providers.ChatMessage{Role: role, Content: content})
				}
			}
		}
	case []map[string]any:
		for _, m := range arr {
			role := getString(m, "role", "")
			content := getString(m, "content", "")
			if role != "" && content != "" {
				out = append(out, providers.ChatMessage{Role: role, Content: content})
			}
		}
	}
	return out
}

func filterChatOutput(out map[string]any) map[string]any {
	filtered := map[string]any{}
	if v, ok := out["content"]; ok {
		filtered["content"] = v
	}
	if v, ok := out["finishReason"]; ok {
		filtered["finishReason"] = v
	}
	if v, ok := out["usage"]; ok {
		filtered["usage"] = v
	}
	return filtered
}

func filterCompleteOutput(out map[string]any) map[string]any {
	filtered := map[string]any{}
	// Map content->text if needed
	if v, ok := out["text"]; ok {
		filtered["text"] = v
	} else if v, ok := out["content"]; ok {
		filtered["text"] = v
	}
	if v, ok := out["finishReason"]; ok {
		filtered["finishReason"] = v
	}
	if v, ok := out["usage"]; ok {
		filtered["usage"] = v
	}
	return filtered
}

func main() {
	port := flag.Int("port", 0, "TCP port for RPC server (required)")
	flag.Parse()
	if *port == 0 {
		log.Fatal("missing --port/-port")
	}

	if err := rpc.Register(&LlmPlugin{}); err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("llm plugin listening on %s", addr)
	rpc.Accept(ln)
}
