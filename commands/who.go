package commands

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/urfave/cli/v3"

	"seeed-cli/commands/configs"
	"seeed-cli/commands/funcs"
)

const whoBatchKB = 512

const whoBatchPrompt = `你是代码风格分析师。下面「代码批次」来自用户本地工程（路径与内容对应）。请**极粗略**估计本批次中：
- 「古法」：偏传统手写、老练、随意或个人痕迹；
- 「AI」：偏 AI 辅助生成感（过于工整、模板化、注释风格像说明文档、命名过于「教科书」等）。

注意：无法验证真实作者；注释很少、大段无注释也可**倾向**判为 AI 感（仍须诚实，不确定处宁可给中间值）。

只输出**一行**，不要其它文字、不要 markdown，格式严格为：
古法:数字 AI:数字
其中两个整数 0～100 且相加等于 100。`

const whoMergePrompt = `下面是同一仓库多个代码批次分别给出的「古法/AI」占比（每行格式 古法:xx AI:yy，xx+yy=100）。请综合为**整仓**一个估计（可加权，非精确科学）。

只输出**一行**，格式严格为：
古法:数字 AI:数字
两个整数 0～100 且相加等于 100。

各批结果：
`

var reWhoLine = regexp.MustCompile(`(?i)古法\D*(\d+)\D*AI\D*(\d+)`)

// parseWhoRatios 从 LLM 返回文本中提取 古法% 与 AI%。
func parseWhoRatios(text string) (trad, ai float64, ok bool) {
	t := strings.TrimSpace(text)
	if idx := strings.IndexAny(t, "\r\n"); idx >= 0 {
		t = strings.TrimSpace(t[:idx])
	}
	m := reWhoLine.FindStringSubmatch(t)
	if len(m) != 3 {
		m = reWhoLine.FindStringSubmatch(strings.ReplaceAll(text, "\n", " "))
	}
	if len(m) != 3 {
		return 0, 0, false
	}
	var a, b int
	_, e1 := fmt.Sscanf(m[1], "%d", &a)
	_, e2 := fmt.Sscanf(m[2], "%d", &b)
	if e1 != nil || e2 != nil {
		return 0, 0, false
	}
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	if a > 100 {
		a = 100
	}
	if b > 100 {
		b = 100
	}
	sum := a + b
	if sum == 0 {
		return 50, 50, true
	}
	return float64(a) * 100 / float64(sum), float64(b) * 100 / float64(sum), true
}

// HandleWho：全屏 rep UI，扫描目录后经 LLM 估计古法/AI 占比并展示比例条。
func HandleWho(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	total, err := funcs.CountFiles(pwd)
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI WHO", pwd, total, false, runWhoWork)
}

func runWhoWork(m *repModel) {
	m.SendLog(bootOKLine("kernel: code provenance scan"))
	m.SendLog(bootOKLine(fmt.Sprintf("rootfs: %s", m.pwd)))

	cfg, err := configs.LoadConfig()
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	model := cfg.Provider.BaiLian.Model

	var estimates []string
	codeContent := ""
	current := 0
	den := m.total
	if den < 1 {
		den = 1
	}

	flushBatch := func() {
		if strings.TrimSpace(codeContent) == "" {
			return
		}
		m.SendLog(bootWaitLine("llm: who batch…"))
		prompt := whoBatchPrompt + "\n\n--- 代码批次 ---\n" + codeContent
		out, lerr := m.RunLLMStream(prompt, model)
		if lerr != nil {
			m.SendLog(logWarn.Render(lerr.Error()))
		} else {
			line := strings.TrimSpace(out)
			if idx := strings.IndexAny(line, "\r\n"); idx >= 0 {
				line = strings.TrimSpace(line[:idx])
			}
			if _, _, ok := parseWhoRatios(line); ok {
				estimates = append(estimates, line)
				m.SendLog(bootOKLine("batch: " + line))
			} else {
				m.SendLog(logWarn.Render("parse: " + line))
			}
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
			if countKB > whoBatchKB || current == m.total {
				flushBatch()
			}
		}
		return nil
	})
	if walkErr != nil {
		m.SendLog(logWarn.Render(walkErr.Error()))
		return
	}

	if len(estimates) == 0 {
		m.SendLog(logWarn.Render("no valid 古法/AI estimates"))
		return
	}

	var finalLine string
	if len(estimates) == 1 {
		finalLine = estimates[0]
	} else {
		m.SendLog(bootWaitLine("llm: merging…"))
		merged, merr := m.RunLLMStream(whoMergePrompt+strings.Join(estimates, "\n"), model)
		if merr != nil {
			m.SendLog(logWarn.Render(merr.Error()))
			return
		}
		finalLine = strings.TrimSpace(merged)
		if idx := strings.IndexAny(finalLine, "\r\n"); idx >= 0 {
			finalLine = strings.TrimSpace(finalLine[:idx])
		}
	}

	trad, ai, ok := parseWhoRatios(finalLine)
	if !ok {
		m.SendLog(logWarn.Render("merge parse: " + finalLine))
		return
	}

	const barW = 80
	nTrad := int(math.Round(trad / 100 * float64(barW)))
	if nTrad < 0 {
		nTrad = 0
	}
	if nTrad > barW {
		nTrad = barW
	}
	tradSt := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	aiSt := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	ch := "█"
	left := strings.Repeat(ch, nTrad)
	right := strings.Repeat(ch, barW-nTrad)
	barLine := tradSt.Render(left) + aiSt.Render(right)

	m.SendLog(logDim.Render("\n（LLM 主观估计，请理性看待）\n"))
	m.SendLog(barLine)
	m.SendLog(barLine)
	m.SendLog(fmt.Sprintf("\n%s 古法 %.0f%%    %s AI %.0f%% \n\n",
		tradSt.Render("▓"), trad, aiSt.Render("█"), ai))
}
