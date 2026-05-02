package main

import ( 
    // "fmt"
    "log"
    "os"
    "context"

    "github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"

	"seeed-cli/commands"
)


// 命令集合，方便后面扩展
var _commands = []*cli.Command{ 
	{ 
		Name: "set-ak",
		Usage: "设置 LLM 的 api-key, seeed-cli set-ak xxx[您的api key]", 
		Action: commands.HandleSetAK, 
	},
	{ 
		Name: "get-ak",
		Usage: "查看已经设置的 LLM 的 api-key, seeed-cli get-ak", 
		Action: commands.HandleGetAK, 
	},
	{ 
		Name: "frame",
		Usage: "项目架构分析",
		Aliases:  []string{"f"},
		Action: commands.HandleFrame, 
	},
	{ 
		Name: "safe-scan",
		Usage: "项目代码安全扫描",
		Aliases:  []string{"ss"},
		Action: commands.HandleSafeScan, 
	},

	{ 
		Name: "clear",
		Usage: "清除所有配置", 
		Action: commands.HandleClear, 
	},
	
}

func main() {

	// 初始化项目配置
	err := configs.Init()
	if err != nil {
		panic(err)
	}


	cfg, err := configs.LoadConfig()
	if err != nil {
		panic(err)
	}
 
	// 装饰一下 Help 命令
	commands.CustomHelp()

	// 启动命令
	cmd := &cli.Command{
		Name: cfg.Name,
        Usage: cfg.Desc,
        Version: cfg.Version,
		Commands: _commands,
	}
  
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
