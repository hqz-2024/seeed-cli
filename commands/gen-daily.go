// gen-daily：读取今日 git 提交记录，经 LLM 生成纯文本工作日报；全屏 UI 展示并写入 seeed-cli/daily-*.txt。
package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"
)

const genDailyPrompt = `你是工作日报助手。下面「今日 Git 提交记录」为当前仓库、按作者日期筛选出的今日全部提交（可能为空）。

请输出纯文本（禁止 Markdown：不要用 #、**、- 列表、代码围栏、链接语法），便于用户整段复制到飞书文档。
结构建议：
第一行写标题：工作日报 + 日期 
再空一行写「主要工作内容」…（按提交归纳，突出业务价值与技术点）
再空一行写「小结」…（一两句话）

若今日无任何提交，说明无提交并写一句明日可跟进占位即可。
不要编造不存在的提交；未在记录中出现的信息不要虚构。`

// HandleGenDaily：全屏 tea + 流式 LLM；纯文本展示（无 glamour）并落盘 ./seeed-cli/daily-*.txt。
func HandleGenDaily(ctx context.Context, cmd *cli.Command) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("未找到 git 命令: %w", err)
	}
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI DAILY REPORT", pwd, 1, true, runGenDailyWork)
}

// runGenDailyWork：在 rep UI 内拉取 git log、流式调用 LLM、写文件并刷新 lastSave。
func runGenDailyWork(m *repModel) {
	m.SendLog(bootOKLine("kernel: daily report"))
	m.SendLog(bootOKLine(fmt.Sprintf("rootfs: %s", m.pwd)))

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	since := dayStart.Format(time.RFC3339)
	until := dayEnd.Format(time.RFC3339)

	logOut, err := gitRun("log", "--since", since, "--until", until, "--no-merges",
		"--pretty=format:%h | %an | %ad | %s", "--date=iso-strict")
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}

	cfg, err := configs.LoadConfig()
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	model := cfg.Provider.BaiLian.Model

	var sb strings.Builder
	sb.WriteString(genDailyPrompt)
	sb.WriteString("\n\n--- 今日 Git 提交记录 ---\n")
	if strings.TrimSpace(logOut) == "" {
		sb.WriteString("(无提交)\n")
	} else {
		sb.WriteString(logOut)
		sb.WriteString("\n")
	}

	text, err := m.RunLLMStream(sb.String(), model)
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		m.SendLog(logWarn.Render("LLM 未返回有效内容"))
		return
	}

	ssPath := filepath.Join(m.pwd, "seeed-cli")
	if _, statErr := os.Stat(ssPath); os.IsNotExist(statErr) {
		if mkErr := os.MkdirAll(ssPath, 0755); mkErr != nil {
			m.SendLog(logWarn.Render(mkErr.Error()))
			return
		}
	}
	fileName := "daily-" + now.Format("2006-01-02_15_04_05") + ".txt"
	outPath := filepath.Join(ssPath, fileName)
	if werr := os.WriteFile(outPath, []byte(text), 0644); werr != nil {
		m.SendLog(logWarn.Render(werr.Error()))
		return
	}
	m.lastSave = outPath
	m.SendLog(bootOKLine("saved: " + outPath))
}
