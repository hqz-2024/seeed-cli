package commands

import (
    "fmt"  
	"os" 
    "context"

    "github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"
)


func HandleClear(ctx context.Context, cmd *cli.Command) error { 
    path := configs.GetConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err){
        // 本来就不存在，无需删除
        return nil
    }

    err := os.Remove(path)
    if err != nil {
        panic(err)
    }

    fmt.Printf("信息清除成功\n")
	return nil
}

 