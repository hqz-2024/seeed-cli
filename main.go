package main

import (
	// "fmt"
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"

	"seeed-cli/commands"
)

// 命令集合，方便后面扩展
var _commands = []*cli.Command{
	{
		Name:   "set-ak",
		Usage:  "设置 LLM 的 api-key, seeed-cli set-ak BaiLian xxx[您的api key]",
		Action: commands.HandleSetAK,
	},
	{
		Name:   "get-ak",
		Usage:  "查看已经设置的 LLM 的 api-key, seeed-cli get-ak BaiLian",
		Action: commands.HandleGetAK,
	},
	{
		Name:    "frame",
		Usage:   "项目架构分析",
		Aliases: []string{"f"},
		Action:  commands.HandleFrame,
	},
	{
		Name:    "safe-scan",
		Usage:   "项目代码安全扫描",
		Aliases: []string{"safe"},
		Action:  commands.HandleSafeScan,
	},
	{
		Name:    "quality",
		Usage:   "代码质量评测",
		Aliases: []string{"q"},
		Action:  commands.HandleQuality,
	},
	{
		Name:    "gen-commit",
		Usage:   "根据暂存区生成带类型与图标的 commit 说明。(请先进行 git add 后在执行 seed-cli gen-commit)",
		Aliases: []string{"commit"},
		Action:  commands.HandleGenCommit,
	},
	{
		Name:    "gen-daily",
		Usage:   "根据今日 git 提交记录生成日报",
		Aliases: []string{"daily"},
		Action:  commands.HandleGenDaily,
	},
	{
		Name:    "gen-ai-agent",
		Usage:   "扫描当前目录源码，由 LLM 归纳生成 AI-AGENT.md",
		Aliases: []string{"agent"},
		Action:  commands.HandleGenAIAgent,
	},
	{
		Name:    "gen-api-doc",
		Usage:   "扫描当前目录源码，由 LLM 归纳生成 API-DOC.md（含示例）",
		Aliases: []string{"api"},
		Action:  commands.HandleGenAPIDoc,
	},
	{
		Name:   "who",
		Usage:  "启发式粗估源码「古法/手写」与「偏 AI 风格」占比（控制台比例条）",
		Action: commands.HandleWho,
	},
	{
		Name:    "skills-scan",
		Usage:   "扫描并分析 AI 工具 skills：场景/触发条件 + 质量评分 + 缺口分析",
		Aliases: []string{"skills", "sk"},
		Action:  commands.HandleSkills,
	},
	{
		Name:    "skills-sync",
		Usage:   "跨工具同步 skills：交互选择源 skill 与目标工具，生成 SKILL.md 包",
		Aliases: []string{"sync"},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "target", Usage: "目标工具：cursor|claude|windsurf|augment"},
			&cli.StringFlag{Name: "skills", Usage: "逗号分隔的源 skill 名（缺省进入交互式多选）"},
			&cli.BoolFlag{Name: "overwrite", Usage: "目标文件已存在时是否覆盖"},
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "跳过确认提示"},
		},
		Action: commands.HandleSkillsSync,
	},
	{
		Name:    "skills-pull",
		Usage:   "联网拉取高星 skill（内嵌 Top-50 索引），下载后按目标工具的方言安装",
		Aliases: []string{"pull"},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "search", Usage: "在 id/name/description/tags 中模糊过滤"},
			&cli.StringFlag{Name: "ids", Usage: "逗号分隔的 skill id 或 name（缺省进入交互式多选）"},
			&cli.StringFlag{Name: "target", Usage: "目标工具：cursor|claude|windsurf|augment"},
			&cli.BoolFlag{Name: "list", Aliases: []string{"l"}, Usage: "只列出当前匹配的 skill 索引，不下载"},
			&cli.BoolFlag{Name: "overwrite", Usage: "目标文件已存在时是否覆盖"},
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "跳过确认提示"},
		},
		Action: commands.HandleSkillsPull,
	},

	{
		Name:   "clear",
		Usage:  "清除所有配置",
		Action: commands.HandleClear,
	},
	{
		Name:   "install-s",
		Usage:  "创建一个 seee-cli 的简短命令： s （功能与 seeed-cli 一致），如: s -h",
		Action: commands.HandleInstallShort,
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
		Name:     cfg.Name,
		Usage:    cfg.Desc,
		Version:  configs.Version,
		Commands: _commands,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
