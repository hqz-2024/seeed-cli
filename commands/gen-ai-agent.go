// gen-ai-agent：遍历当前项目源码，分批经 LLM 归纳后在执行目录生成 AI-AGENT.md。
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"
	"seeed-cli/commands/funcs"
)

const genAIAgentBatchPrompt = `你是项目规范分析助手。下面「代码批次」来自用户本地工程中的若干源文件（路径与内容一一对应）。

请只做**能从代码中直接观察到的归纳**（技术栈、目录/模块习惯、命名与注释风格、错误处理、日志、测试组织、配置方式等）。不要编造当前批次中未出现的内容；不确定的不要写。

输出 Markdown，建议用二级标题分段（如「技术栈与依赖」「目录与模块」「代码风格」「测试与质量」「配置与环境」「对 AI 协作的约束」等按本批实际能看出的来），条目用列表即可。不要输出与文档无关的寒暄。`

const genAIAgentMergePrompt = `你是技术文档编辑。下面「多批分析草稿」由同一项目的源代码分批自动归纳得到，可能有重复或矛盾（以后者/更具体者为准）。

请合并为**一份**面向后续 AI 编码助手的说明文档，保存为项目根目录的 AI-AGENT.md 风格内容：

- 主标题：# AI 协作说明
- 第二行可加一句小字说明：由工具根据仓库代码扫描归纳生成，可人工修订。
- 去重、理顺结构；冲突处选择更贴近代码事实的表述，无法判断的写入「待确认」而非臆造。
- 使用规范 Markdown（标题层级清晰，路径/命令/包名用行内代码格式标出）。
- 末尾可增加简短「禁止事项」：不要编造仓库中不存在的 API/路径/配置等。

以下为草稿全文：

`

const genAIAgentSynthMaxBytes = 120000

// HandleGenAIAgent：全屏 UI + 流式 LLM，扫描当前目录代码后写入 ./AI-AGENT.md。
func HandleGenAIAgent(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	total, err := funcs.CountFiles(pwd)
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI AI AGENT DOC", pwd, total, false, runGenAIAgentWork)
}

func runGenAIAgentWork(m *repModel) {
	m.SendLog(bootOKLine("kernel: ai-agent doc"))
	m.SendLog(bootOKLine(fmt.Sprintf("rootfs: %s", m.pwd)))

	cfg, err := configs.LoadConfig()
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	model := cfg.Provider.BaiLian.Model

	codeContent := ""
	var draft strings.Builder
	current := 0
	den := m.total
	if den < 1 {
		den = 1
	}

	flushBatch := func() {
		if strings.TrimSpace(codeContent) == "" {
			return
		}
		m.SendLog(bootWaitLine("llm: analyzing batch…"))
		prompt := genAIAgentBatchPrompt + "\n\n--- 代码批次 ---\n" + codeContent
		llmRes, err := m.RunLLMStream(prompt, model)
		if err != nil {
			m.SendLog(logWarn.Render(err.Error()))
		} else if strings.TrimSpace(llmRes) != "" {
			if draft.Len() > 0 {
				draft.WriteString("\n\n---\n\n")
			}
			draft.WriteString(strings.TrimSpace(llmRes))
		}
		codeContent = ""
	}

	walkErr := filepath.Walk(m.pwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && funcs.ShouldIgnore(info.Name()) {
			return filepath.SkipDir
		}
		if !info.IsDir() && funcs.IsCodeFile(path) && (info.Size() < funcs.MaxFileSize) {
			current++
			progress := float64(current) / float64(den) * 100
			m.SendLog(bootWaitLine(fmt.Sprintf("%.1f%% (%d/%d) %dkb %s", progress, current, m.total, info.Size()/1024, path)))

			content, rerr := os.ReadFile(path)
			if rerr != nil {
				m.SendLog(logDim.Render("[ ") + logWarn.Render("!!") + logDim.Render(" ] ") + logText.Render(path+": "+rerr.Error()))
				return nil
			}
			codeContent += "代码文件：" + path + "\n\n" + string(content)

			countKB := len(codeContent) / 1024
			if countKB > 512 || current == m.total {
				flushBatch()
			}
		}
		return nil
	})
	if walkErr != nil {
		m.SendLog(logWarn.Render(walkErr.Error()))
		return
	}

	combined := strings.TrimSpace(draft.String())
	var finalText string
	if combined == "" {
		finalText = "# AI 协作说明\n\n（未扫描到受支持的源代码文件；请确认在项目根目录执行，且存在常见源码扩展名。）\n"
		m.SendLog(logWarn.Render("no code files in batch output"))
	} else {
		m.SendLog(bootWaitLine("llm: merging document…"))
		synthIn := combined
		if len(synthIn) > genAIAgentSynthMaxBytes {
			synthIn = synthIn[:genAIAgentSynthMaxBytes] + "\n\n（前文已截断，请仅依据以上内容合并。）\n"
		}
		merged, merr := m.RunLLMStream(genAIAgentMergePrompt+synthIn, model)
		if merr != nil {
			m.SendLog(logWarn.Render(merr.Error()))
			finalText = "# AI 协作说明\n\n（合并步骤失败，以下为分批草稿拼接，请人工整理。）\n\n" + combined
		} else {
			finalText = strings.TrimSpace(merged)
			if finalText == "" {
				finalText = combined
			}
		}
	}

	outPath := filepath.Join(m.pwd, "AI-AGENT.md")
	if werr := os.WriteFile(outPath, []byte(finalText), 0644); werr != nil {
		m.SendLog(logWarn.Render(werr.Error()))
		return
	}
	m.lastSave = outPath
	m.SendLog(bootOKLine("saved: " + outPath + " @ " + time.Now().Format(time.RFC3339)))
}
