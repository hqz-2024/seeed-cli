package funcs

import (
	"fmt"
	"context"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"seeed-cli/commands/configs"
)

func FetchLLMStream(ctx context.Context, provider string, text string, llmModel string, onDelta func(string)) (string, error) {
	cfg, err := configs.LoadConfig()
	if err != nil {
		return "", err
	}

	if provider == "" {
		provider = cfg.DefaultProvider
		if provider == "" {
			provider = "BaiLian"
		}
	}

	ak, err := configs.GetAK(provider)
	if err != nil {
		return "", err
	}
	if ak == "" {
		return "", fmt.Errorf("\n\n[ERROR] api key is not set for %s, please set api key first.\n\n", provider)
	}

	if llmModel == "" {
		llmModel = configs.GetProviderModel(cfg, provider)
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
		Model: llmModel,
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
		return b.String(), err
	}
	return b.String(), nil
}

func FetchLLM(provider string, text string, llmModel string) string {
	out, err := FetchLLMStream(context.TODO(), provider, text, llmModel, nil)
	if err != nil {
		panic(err)
	}
	return out
}
