// gen-api-doc：遍历当前项目源码，分批经 LLM 归纳后在执行目录生成 API-DOC.md（含请求/响应案例）。
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

const genAPIDocBatchPrompt = `你是后端接口文档助手。下面「代码批次」来自用户本地工程中的若干源文件（路径与内容一一对应）。

请只依据**本批代码中可直接读出的**路由/接口信息做归纳（如 HTTP 方法+路径、handler 名、gRPC service/method、GraphQL 类型与字段、中间件、鉴权注解等）。不要编造本批未出现的路径或字段；读不出的标「待确认」。

对每个可识别的接口，用 Markdown 输出，建议结构：
- 二级标题：模块或文件相对路径（择一，本批内一致即可）
- 每个接口用三级标题：「HTTP方法 + 路径」或 gRPC 的 FullMethod 名
- 正文列表：简要说明、路径参数/Query/Body 字段（来自 struct/json tag 等）、成功/错误响应形态（若有类型定义则概括）
- **必须**包含「示例」小节：给出 1 组可执行的示例（如 curl 或等价 HTTP 原文），以及**示例**请求/响应 JSON（字段名与类型尽量与代码一致；缺省值用合理占位，并注明「示例值」）

不要输出与接口文档无关的寒暄。`

const genAPIDocMergePrompt = `你是 API 文档编辑。下面「多批接口草稿」由同一项目源码分批自动归纳，可能有重复或矛盾（以后者/更具体者为准）。

请合并为**一份**后端接口说明文档：

- 主标题：# 接口文档
- 第二行小字：由工具根据仓库代码扫描归纳生成，可人工修订。
- 开头可增加简短「约定」：Base URL、鉴权方式等（仅当草稿中有依据时写出，否则省略或写待确认）。
- 按模块/服务分组；同一 path+method 只保留一条；冲突选更贴近代码的表述。
- 每个接口保留或补全「示例」（curl + JSON）；无法从代码推断的示例标为示例占位。
- 路径/方法/类型名用行内代码格式。

以下为草稿全文：

`

const genAPIDocSynthMaxBytes = 120000

// HandleGenAPIDoc：全屏 UI + 流式 LLM，扫描当前目录代码后写入 ./API-DOC.md。
func HandleGenAPIDoc(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	total, err := funcs.CountFiles(pwd)
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI API DOC", pwd, total, false, runGenAPIDocWork)
}

func runGenAPIDocWork(m *repModel) {
	m.SendLog(bootOKLine("kernel: api doc"))
	m.SendLog(bootOKLine(fmt.Sprintf("rootfs: %s", m.pwd)))

	cfg, err := configs.LoadConfig()
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	provider := configs.GetBestProvider(cfg)
	model := configs.GetProviderModel(cfg, provider)

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
		prompt := genAPIDocBatchPrompt + "\n\n--- 代码批次 ---\n" + codeContent
		llmRes, err := m.RunLLMStream(provider, prompt, model)
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
		finalText = "# 接口文档\n\n（未扫描到受支持的源代码文件；请在项目根目录执行，且存在常见源码扩展名。）\n"
		m.SendLog(logWarn.Render("no code files in batch output"))
	} else {
		m.SendLog(bootWaitLine("llm: merging document…"))
		synthIn := combined
		if len(synthIn) > genAPIDocSynthMaxBytes {
			synthIn = synthIn[:genAPIDocSynthMaxBytes] + "\n\n（前文已截断，请仅依据以上内容合并。）\n"
		}
		merged, merr := m.RunLLMStream(provider, genAPIDocMergePrompt+synthIn, model)
		if merr != nil {
			m.SendLog(logWarn.Render(merr.Error()))
			finalText = "# 接口文档\n\n（合并步骤失败，以下为分批草稿拼接，请人工整理。）\n\n" + combined
		} else {
			finalText = strings.TrimSpace(merged)
			if finalText == "" {
				finalText = combined
			}
		}
	}

	outPath := filepath.Join(m.pwd, "API-DOC.md")
	if werr := os.WriteFile(outPath, []byte(finalText), 0644); werr != nil {
		m.SendLog(logWarn.Render(werr.Error()))
		return
	}
	m.lastSave = outPath
	m.SendLog(bootOKLine("saved: " + outPath + " @ " + time.Now().Format(time.RFC3339)))
}
