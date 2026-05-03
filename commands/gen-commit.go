// gen-commit：读取暂存区与 git 状态，经 LLM 生成带类型前缀与图标的单行提交说明。
package commands

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"
	"seeed-cli/commands/funcs"
)

const genCommitPrompt = `你是 Git 提交说明生成器。仅根据下面提供的「暂存区」信息。

格式（严格遵守）：
「单个 Emoji」「空格」「类型」「英文冒号」「空格」「中文简述」

类型只能是以下之一（首字母大写）：Feat / Fix / Docs / Style / Refactor / Chore / Perf / Test

Emoji 与类型固定搭配（从中选一）：
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
- 中文简述控制在 500 字以内，概括本次暂存修改的核心意图。
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

// HandleGenCommit：基于 git status 与 git diff --cached 生成提交说明并打印到 stdout。
func HandleGenCommit(ctx context.Context, cmd *cli.Command) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("未找到 git 命令: %w", err)
	}

	status, err := gitRun("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}

	diffFull, err := gitRun("diff", "--cached")
	if err != nil {
		return fmt.Errorf("git diff --cached: %w", err)
	}
	if diffFull == "" {
		return fmt.Errorf("暂存区为空，请先执行 git add")
	}

	diffStat, _ := gitRun("diff", "--cached", "--stat")
	if len(diffFull) > maxDiffBytes {
		diffFull = diffFull[:maxDiffBytes] + "\n\n_[diff truncated]_\n"
	}

	cfg, err := configs.LoadConfig()
	if err != nil {
		return err
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

	fmt.Println( sb.String())

	out, err := funcs.FetchLLMStream(ctx, sb.String(), model, nil)
	if err != nil {
		return err
	}
	line := strings.TrimSpace(out)
	if line == "" {
		return fmt.Errorf("LLM 未返回有效内容")
	}
	// 只取首行，去掉可能的代码围栏
	line = strings.TrimPrefix(line, "`")
	line = strings.TrimSuffix(line, "`")
	line = strings.TrimSpace(line)
	if idx := strings.IndexAny(line, "\r\n"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	fmt.Println(line)
	return nil
}
