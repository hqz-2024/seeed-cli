// frame：采集仓库清单与目录树，单次 LLM 生成架构报告（含 Mermaid），写入 seeed-cli/*.md。
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

const frameLLMPrompt = `你是一名资深软件架构师。下面提供某个代码仓库中自动采集的片段（清单文件、README、目录树、关键入口源码等）。请严格用 **Markdown** 输出一份《项目框架分析报告》，语言用中文，结构必须包含以下章节（标题级别自拟，但顺序与内容要覆盖全）：

## 1. 技术栈分析
- 根据 go.mod / package.json / Cargo.toml / pyproject.toml 等推断主语言、框架、构建与运行方式。
- 若信息不足，写明「推断依据」与「不确定项」。

## 2. 架构与数据流
- 描述分层、模块边界、请求/任务如何贯穿系统（从入口到存储或外部服务）。
- 必须包含 **至少一张 Mermaid 架构图**（例如 flowchart TB 或 C4 风格的 flowchart），节点名用英文或拼音缩写均可，但要能对应到真实目录或包名。

## 3. 关键模块与事件分析
- 挑选 2～4 个你认为最关键的业务或技术模块（例如 CLI 命令、配置加载、LLM 调用链等）。
- 每个模块给出 **一张 Mermaid 图**（优先 sequenceDiagram 描述交互时序；若更适合则用 flowchart LR）。
- 每张图前用一小段话说明场景与参与者。

## 4. 快速上手路线
- 给新人一条 **可执行的一天学习路径**（按天或按小时粒度均可）：从 clone 仓库开始，到能改一个最小功能、跑通主流程。
- 列出必读文件/目录、建议的调试断点或日志位置、常见坑。

写作要求：
- 全文使用标准 Markdown；代码块标明语言。
- Mermaid 须放在 fenced code block 中，首行语言标签写 mermaid。
- Mermaid 须可被 GitHub / Mermaid Live Editor 解析：第一行用 flowchart TD / LR、sequenceDiagram 等合法类型；节点 ID 仅用字母数字下划线且不以数字开头；含空格或特殊字符的展示文字一律用英文双引号包起来；subgraph 须正确嵌套与闭合；避免用 end、graph、style 等保留字作节点 id；少用 HTML 标签。
- 不要编造仓库中不存在的文件路径；若未在上下文中出现则写「未在采集上下文中出现」。
- 总长度建议控制在 2000 字以内；若上下文过长可优先保证第 2、4 章质量。`

// HandleFrame：CLI 入口，复用 runRepUI 全屏交互。
func HandleFrame(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI FRAME / ARCH", pwd, 0, false, runFrameWork)
}

// runFrameWork：采集语料、流式生成报告；写入磁盘的为 LLM 原始 Markdown（保留 mermaid 围栏供 IDE 渲染）。
func runFrameWork(m *repModel) {
	m.SendLog(bootOKLine("framework scan online"))
	m.SendLog(bootOKLine(fmt.Sprintf("workspace: %s", m.pwd)))

	m.SendLog(bootWaitLine("collecting corpus…"))
	corpus := collectFrameCorpus(m.pwd)
	if len(corpus) > 200000 {
		corpus = corpus[:200000] + "\n\n_[corpus truncated at 200KB]_\n"
	}
	m.SendLog(bootOKLine(fmt.Sprintf("corpus size: %d bytes", len(corpus))))

	m.SendLog(bootWaitLine("llm: generating architecture report…"))
	full, err := m.RunLLMStream(frameLLMPrompt+"\n\n---\n\n"+corpus, "qwen3.6-flash")
	if err != nil || full == "" {
		return
	}

	now := time.Now()
	fileName := "frame-" + now.Format("2006-01-02_15_04_05") + ".md"
	dir, err := os.Getwd()
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	ssPath := filepath.Join(dir, "seeed-cli")
	if _, e := os.Stat(ssPath); os.IsNotExist(e) {
		_ = os.MkdirAll(ssPath, 0755)
	}
	outPath := filepath.Join(ssPath, fileName)
	if err := os.WriteFile(outPath, []byte(full), 0644); err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	m.lastSave = outPath
	m.SendLog(bootOKLine("saved: " + outPath))
}

// readFileCap：读取文本文件，超长截断以免撑爆上下文。
func readFileCap(path string, maxBytes int) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(b) <= maxBytes {
		return string(b)
	}
	return string(b[:maxBytes]) + "\n\n_[truncated]_\n"
}

// collectFrameCorpus：拼接清单文件、目录树与若干 main.go 片段，供 LLM 推断架构。
func collectFrameCorpus(root string) string {
	var sb strings.Builder
	candidates := []string{
		"go.mod", "go.work", "go.sum",
		"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
		"Cargo.toml", "Cargo.lock",
		"pyproject.toml", "requirements.txt", "Pipfile",
		"Makefile", "makefile", "Dockerfile", "docker-compose.yml", "compose.yaml",
		"README.md", "README.MD", "readme.md", "Readme.md",
	}
	for _, name := range candidates {
		p := filepath.Join(root, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			sb.WriteString("### `")
			sb.WriteString(name)
			sb.WriteString("`\n\n")
			sb.WriteString("```\n")
			sb.WriteString(readFileCap(p, 64000))
			sb.WriteString("\n```\n\n")
		}
	}

	sb.WriteString("### Directory tree (depth ≤ 5)\n\n")
	sb.WriteString("```\n")
	sb.WriteString(buildDirTree(root, 5, 1000))
	sb.WriteString("\n```\n\n")

	sb.WriteString("### Entry / main samples\n\n")
	for _, rel := range []string{"main.go", filepath.Join("cmd", "main.go")} {
		p := filepath.Join(root, rel)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			sb.WriteString("#### `")
			sb.WriteString(rel)
			sb.WriteString("`\n\n```go\n")
			sb.WriteString(readFileCap(p, 48000))
			sb.WriteString("\n```\n\n")
		}
	}
	cmdRoot := filepath.Join(root, "cmd")
	if st, err := os.Stat(cmdRoot); err == nil && st.IsDir() {
		_ = filepath.Walk(cmdRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if path != cmdRoot && shouldIgnoreFrameDir(info.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Base(path) != "main.go" {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if rel == "main.go" {
				return nil
			}
			sb.WriteString("#### `")
			sb.WriteString(filepath.ToSlash(rel))
			sb.WriteString("`\n\n```go\n")
			sb.WriteString(readFileCap(path, 32000))
			sb.WriteString("\n```\n\n")
			return nil
		})
	}

	return sb.String()
}

// shouldIgnoreFrameDir：与 frameIgnore 表配合 filepath.SkipDir。
func shouldIgnoreFrameDir(name string) bool {
	_, ok := frameIgnore[name]
	return ok
}

// frameIgnore：生成目录树时跳过的目录名（含 seeed-cli 避免把本工具输出扫进语料）。
var frameIgnore = map[string]struct{}{
	".git": {}, "node_modules": {}, "vendor": {}, "dist": {}, "build": {},
	".idea": {}, ".vscode": {}, "seeed-cli": {},
}

// buildDirTree：自根目录向下列出树形文本，限制深度与总行数。
func buildDirTree(root string, maxDepth int, maxLines int) string {
	var lines []string
	count := 0
	var walk func(string, string, int)
	walk = func(absDir, indent string, remainingDepth int) {
		if count >= maxLines {
			return
		}
		ents, err := os.ReadDir(absDir)
		if err != nil {
			return
		}
		var names []string
		for _, e := range ents {
			if e.IsDir() && shouldIgnoreFrameDir(e.Name()) {
				continue
			}
			names = append(names, e.Name())
		}
		sort.Strings(names)
		for i, name := range names {
			if count >= maxLines {
				return
			}
			full := filepath.Join(absDir, name)
			fi, err := os.Stat(full)
			if err != nil {
				continue
			}
			branch := "├── "
			if i == len(names)-1 {
				branch = "└── "
			}
			suf := ""
			if fi.IsDir() {
				suf = "/"
			}
			lines = append(lines, indent+branch+name+suf)
			count++
			if fi.IsDir() && remainingDepth > 0 {
				next := indent
				if i == len(names)-1 {
					next += "    "
				} else {
					next += "│   "
				}
				walk(full, next, remainingDepth-1)
			}
		}
	}
	lines = append(lines, ".")
	count++
	walk(root, "", maxDepth)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}
