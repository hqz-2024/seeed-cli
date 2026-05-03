package funcs

import (
	"fmt"
	"context"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"seeed-cli/commands/configs"
)

func FetchLLMStream(ctx context.Context, text string, llmModel string, onDelta func(string)) (string, error) {
	cfg, err := configs.LoadConfig()
	if err != nil {
		return "", err
	}

	ak, err := configs.GetAK("BaiLian")
	if err != nil {
		return "", err
	}
	if ak == "" {
		return "", fmt.Errorf("\n\n[ERROR] api key is not set, please set api key first.\n\n")
	}

	if llmModel == "" {
		llmModel = cfg.Provider.BaiLian.Model
	}

	client := openai.NewClient(
		option.WithAPIKey(ak),
		option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
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

func FetchLLM(text string, llmModel string) string {
	out, err := FetchLLMStream(context.TODO(), text, llmModel, nil)
	if err != nil {
		panic(err)
	}
	return out
}
