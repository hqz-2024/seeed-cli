package funcs

import (
	"fmt"
	"context"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"seeed-cli/commands/configs"
)

func FetchLLMStream(ctx context.Context, provider string, text string, llmModel string, onDelta func(string), onProvider func(provider, model string)) (string, error) {
	cfg, err := configs.LoadConfig()
	if err != nil {
		return "", err
	}

	var providers []string
	if provider != "" {
		providers = append(providers, provider)
	}
	providers = append(providers, configs.GetAvailableProviders(cfg, provider)...)
	if len(providers) == 0 {
		return "", fmt.Errorf("\n\n[ERROR] no api key configured, please set api key first.\n\n")
	}

	var lastErr error
	for i, p := range providers {
		model := llmModel
		if i > 0 {
			model = ""
		}
		if model == "" {
			model = configs.GetProviderModel(cfg, p)
		}
		if onProvider != nil {
			onProvider(p, model)
		}
		result, err := tryProvider(ctx, cfg, p, text, model, onDelta)
		if err == nil && result != "" {
			return result, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("all configured providers failed")
}

func tryProvider(ctx context.Context, cfg *configs.Config, provider string, text string, llmModel string, onDelta func(string)) (string, error) {
	ak, err := configs.GetAK(provider)
	if err != nil {
		return "", err
	}
	if ak == "" {
		return "", fmt.Errorf("api key not set for %s", provider)
	}

	model := llmModel
	if model == "" {
		model = configs.GetProviderModel(cfg, provider)
	}

	baseURL := configs.GetProviderBaseURL(provider)
	if baseURL == "" {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	client := openai.NewClient(
		option.WithAPIKey(ak),
		option.WithBaseURL(baseURL),
	)
	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(text),
		},
		Model: model,
	})
	defer stream.Close()

	var b strings.Builder
	for stream.Next() {
		ch := stream.Current()
		for _, c := range ch.Choices {
			if c.Delta.Content == "" {
				continue
			}
			b.WriteString(c.Delta.Content)
			if onDelta != nil {
				onDelta(c.Delta.Content)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return b.String(), fmt.Errorf("[%s] %w", provider, err)
	}
	return b.String(), nil
}

func FetchLLM(provider string, text string, llmModel string) string {
	out, err := FetchLLMStream(context.TODO(), provider, text, llmModel, nil, nil)
	if err != nil {
		panic(err)
	}
	return out
}
