// safe-scan：遍历代码文件分批调 LLM 做安全审计，报告写入 seeed-cli/scan-safe-*.md。
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

// HandleSafeScan：CLI 入口。
func HandleSafeScan(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	total, err := funcs.CountFiles(pwd)
	if err != nil {
		return err
	}

	return runRepUI(ctx, "SEEED-CLI SECURITY SCAN", pwd, total, false, runSafeScanWork)
}

// runSafeScanWork：与旧版逻辑一致，按 512KB 批次调用 RunLLMStream，最后落盘聚合 Markdown。
func runSafeScanWork(m *repModel) {
	codeContent := ""
	safeContent := ""
	current := 0
	den := m.total
	if den < 1 {
		den = 1
	}

	m.SendLog(bootOKLine("kernel: security scanner online"))
	m.SendLog(bootOKLine(fmt.Sprintf("rootfs: %s", m.pwd)))

	cfg, cfgErr := configs.LoadConfig()
	if cfgErr != nil {
		m.SendLog(logWarn.Render(cfgErr.Error()))
		return
	}
	provider := cfg.DefaultProvider
	if provider == "" {
		provider = "BaiLian"
	}
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
				m.SendLog(bootWaitLine("llm: analyzing batch…"))
				prompt := "帮我检查下面代码中是否存在安全问题，明确告诉我存在安全的问题数量和具体的文件地址、代码、问题描述、修复方案。\n" + codeContent
				llmRes, _ := m.RunLLMStream(provider, prompt, model)
				safeContent += llmRes
				codeContent = ""
			}

			if current == m.total {
				now := time.Now()
				fileName := "scan-safe-" + now.Format("2006-01-02_15_04_05") + ".md"
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
				_, werr := f.WriteString(safeContent)
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
