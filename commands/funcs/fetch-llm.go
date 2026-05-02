package funcs

import (
	"context" 
	"fmt"
	"time"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"seeed-cli/commands/configs"
)
 

func FetchLLM(text string, model string) string {
	cfg, err := configs.LoadConfig()
	if err != nil{  
		panic(err)
		return ""
	}

	ak, err := configs.GetAK("BaiLian")
	if err != nil{  
		panic(err)
		return ""
	}

	if model == "" {	 
		model = cfg.Provider.BaiLian.Model
	}

	start := time.Now()

	client := openai.NewClient( 
		option.WithAPIKey(ak),
		option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
	)
	chatCompletion, err := client.Chat.Completions.New(
		context.TODO(), openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(text),
			}, 
			Model: model,
			
		},
	)

	if err != nil {
		panic(err.Error())
	}

	fmt.Println("用户: ", text) 
	fmt.Println("AI: ", chatCompletion.Choices[0].Message.Content)  

	fmt.Println("LLM 耗时：", time.Since(start))
	return "1111"
}
