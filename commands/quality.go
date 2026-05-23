// quality：遍历代码文件分批调 LLM 做宽松代码质量点评，报告写入 seeed-cli/scan-quality-*.md。
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"
	"seeed-cli/commands/funcs"
)

// HandleQuality：CLI 入口。
func HandleQuality(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	total, err := funcs.CountFiles(pwd)
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI CODE QUALITY", pwd, total, false, runQualityWork)
}

// runQualityWork：按 512KB 批次调用 RunLLMStream，聚合 Markdown 落盘。
func runQualityWork(m *repModel) {
	codeContent := ""
	report := ""
	current := 0
	den := m.total
	if den < 1 {
		den = 1
	}

	m.SendLog(bootOKLine("kernel: quality reviewer online"))
	m.SendLog(bootOKLine(fmt.Sprintf("rootfs: %s", m.pwd)))

	cfg, cfgErr := configs.LoadConfig()
	if cfgErr != nil {
		m.SendLog(logWarn.Render(cfgErr.Error()))
		return
	}
	provider := configs.GetBestProvider(cfg)
	model := configs.GetProviderModel(cfg, provider)

	err := filepath.Walk(m.pwd, func(path string, info os.FileInfo, err error) error {
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
			strContent := string(content)
			codeContent += ("代码文件：" + path + "\n\n" + strContent)

			countSize := len(codeContent) / 1024
			if countSize > 512 || current == m.total {
				m.SendLog(bootWaitLine("llm: reviewing batch…"))
				prompt := `请对下面代码做「宽松」的代码质量点评，输出中文 Markdown 报告。要求：
- 语气友好，先肯定做得好的地方，再给改进建议；可适当加入一两句鼓励（不要空洞堆砌）。
- 不必像安全审计那样严苛；小问题可合并概括，重点写 2～5 条最有价值的建议即可。
- 若某文件信息不足，可简短说明。
- 结构建议：每个涉及文件一个小节，含：亮点、可改进点（可选：优先级低/中）、小结。
明确给出本批次你点评到的文件路径列表。`

				llmRes, _ := m.RunLLMStream(provider, prompt+"\n\n"+codeContent, model)
				report += llmRes
				codeContent = ""
			}

			if current == m.total {
				now := time.Now()
				fileName := "scan-quality-" + now.Format("2006-01-02_15_04_05") + ".md"
				dir, gerr := os.Getwd()
				if gerr != nil {
					return gerr
				}
				ssPath := filepath.Join(dir, "seeed-cli")
				if _, statErr := os.Stat(ssPath); os.IsNotExist(statErr) {
					os.MkdirAll(ssPath, 0755)
				}
				outPath := filepath.Join(ssPath, fileName)
				f, cerr := os.Create(outPath)
				if cerr != nil {
					return cerr
				}
				_, werr := f.WriteString(report)
				f.Close()
				if werr != nil {
					return werr
				}
				m.lastSave = outPath
				m.SendLog(bootOKLine("saved: " + outPath))
			}
		}
		return nil
	})
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
	}
}
