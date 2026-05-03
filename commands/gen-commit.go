// gen-commit：读取暂存区与 git 状态，经 LLM 生成带类型前缀与图标的单行提交说明，写入 seeed-cli/gen-commit-*.md。
package commands

import (
	"bytes"
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

const genCommitPrompt = `你是 Git 提交说明生成器。仅根据下面提供的「暂存区」信息。

图标格式（严格遵守）：
✨ Feat
🐛 Fix
📝 Docs
💄 Style
♻️ Refactor
🔧 Chore
⚡ Perf
✅ Test

示例：
1. ✨ Feat: 新增用户注册功能
2.🐛 Fix: 修复用户登录bug
3.📝 Docs: 更新用户手册
4.💄 Style: 调整代码风格
5.♻️ Refactor: 重构用户中心模块
6.🔧 Chore: 更新依赖库
7.⚡ Perf: 优化性能
8.✅ Test: 添加单元测试

规则： 
- 每一项描述不要超过 100 字，简单概括即可。
- 每一项都需要独立一行。
- 不要 markdown、不要英文句子作主描述。
- 若信息不足，根据已有 diff 做最合理推断。`

const maxDiffBytes = 1024 * 1024

func gitRun(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// HandleGenCommit：全屏 UI + 流式 LLM；全文写入 seeed-cli/gen-commit-*.md，退出后 stdout 仅打印首行。
func HandleGenCommit(ctx context.Context, cmd *cli.Command) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("未找到 git 命令: %w", err)
	}
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI GEN COMMIT", pwd, 0, true, runGenCommitWork)
}

// runGenCommitWork：收集暂存区、流式调用 LLM；全文落盘，首行写入 printOnQuit。
func runGenCommitWork(m *repModel) {
	m.SendLog(bootOKLine("kernel: commit message synthesizer"))
	m.SendLog(bootOKLine("rootfs: " + m.pwd))

	m.SendLog(bootWaitLine("git: status --porcelain…"))
	status, err := gitRun("status", "--porcelain")
	if err != nil {
		m.SendLog(logDim.Render("[ ") + logWarn.Render("!!") + logDim.Render(" ] ") + logText.Render("git status: "+err.Error()))
		return
	}
	n := 0
	for _, ln := range strings.Split(status, "\n") {
		if strings.TrimSpace(ln) != "" {
			n++
		}
	}
	m.SendLog(bootOKLine(fmt.Sprintf("status: %d porcelain line(s)", n)))

	m.SendLog(bootWaitLine("git: diff --cached…"))
	diffFull, err := gitRun("diff", "--cached")
	if err != nil {
		m.SendLog(logDim.Render("[ ") + logWarn.Render("!!") + logDim.Render(" ] ") + logText.Render("git diff --cached: "+err.Error()))
		return
	}
	if diffFull == "" {
		m.SendLog(logWarn.Render("暂存区为空，请先执行 git add"))
		return
	}

	m.SendLog(bootWaitLine("git: diff --cached --stat…"))
	diffStat, _ := gitRun("diff", "--cached", "--stat")
	if len(diffFull) > maxDiffBytes {
		diffFull = diffFull[:maxDiffBytes] + "\n\n_[diff truncated]_\n"
		m.SendLog(bootOKLine(fmt.Sprintf("diff: truncated at %d bytes", maxDiffBytes)))
	} else {
		m.SendLog(bootOKLine(fmt.Sprintf("diff: %d bytes cached", len(diffFull))))
	}

	cfg, err := configs.LoadConfig()
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	model := cfg.Provider.BaiLian.Model

	var sb strings.Builder
	sb.WriteString(genCommitPrompt)
	sb.WriteString("\n\n--- git status --porcelain ---\n")
	sb.WriteString(status)
	sb.WriteString("\n\n--- git diff --cached --stat ---\n")
	sb.WriteString(diffStat)
	sb.WriteString("\n\n--- git diff --cached ---\n")
	sb.WriteString(diffFull)

	m.SendLog(bootWaitLine("llm: generating commit line…"))
	out, err := m.RunLLMStream(sb.String(), model)
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	full := strings.TrimSpace(out)
	if full == "" {
		m.SendLog(logWarn.Render("LLM 未返回有效内容"))
		return
	}

	now := time.Now()
	fileName := "gen-commit-" + now.Format("2006-01-02_15_04_05") + ".md"
	ssPath := filepath.Join(m.pwd, "seeed-cli")
	if _, statErr := os.Stat(ssPath); os.IsNotExist(statErr) {
		_ = os.MkdirAll(ssPath, 0755)
	}
	outPath := filepath.Join(ssPath, fileName)
	if werr := os.WriteFile(outPath, []byte(full+"\n"), 0644); werr != nil {
		m.SendLog(logDim.Render("[ ") + logWarn.Render("!!") + logDim.Render(" ] ") + logText.Render("写入文件: "+werr.Error()))
	} else {
		m.lastSave = outPath
		m.SendLog(bootOKLine("saved: " + outPath))
	}

	// 管道 / git commit -m 仍用首行
	first := full
	if idx := strings.IndexAny(first, "\r\n"); idx >= 0 {
		first = strings.TrimSpace(full[:idx])
	}
	first = strings.TrimPrefix(first, "`")
	first = strings.TrimSuffix(first, "`")
	first = strings.TrimSpace(first)
	m.printOnQuit = first
	// m.SendLog(bootOKLine("commit: " + first))
}
