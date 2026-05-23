package commands

import (
    "fmt"  
    "context"

    "github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"
)


func HandleSetAK(ctx context.Context, cmd *cli.Command) error { 
    provider := cmd.Args().Get(0)
    ak := cmd.Args().Get(1)
    
    if provider == "" || ak == "" {
        fmt.Printf("请输入您的 api-key, 使用方式： seeed-cli set-ak [BaiLian|DeepSeek|GPT] xxx(这是您的api-key)\n")
        return nil
    } 

    err := configs.SetAK(provider, ak) 
    if err != nil {  
		return err
	}

    fmt.Printf("设置 api key 成功 %q \n", ak)
	return nil
}


func HandleGetAK(ctx context.Context, cmd *cli.Command) error { 
    provider := cmd.Args().Get(0)
    if provider == "" {
        fmt.Printf("请输入您要查询的平台，如： seeed-cli get-ak [BaiLian|DeepSeek|GPT]\n")
        return nil
    }
        
    ak, err := GetAK(provider) 
    if err != nil { 
	    return err
    }
    fmt.Printf("api key:%s \n", ak)
	return nil
}

/**
* 通用的获取 api-key 的方式
*/
func GetAK(provider string) (string, error) {  
    ak, err := configs.GetAK(provider) 
    if err != nil {  
		fmt.Println("未查询到 api key")
		return "", nil
	} 
	return ak, nil
}