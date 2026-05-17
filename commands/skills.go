// skills-scan：全工程扫描 SKILL.md → LLM 综合分析（总览/分组/质量/缺口）→ 写入 seeed-cli/skills-*.md。
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/funcs"
)

const skillsPrompt = `你是 AI 编程工具配置审计助手。下面会给你两段输入：
（A）若干 SKILL.md 文件（每条标注 source、name、相对路径、frontmatter、正文片段）；
（B）项目技术栈摘要（清单文件 + 目录树片段）。

请输出一份 Markdown 报告，**严格按以下四章顺序与标题**：

## 1. 总览
- 表格列出：来源工具 | skill 名 | 应用场景（一句话） | 触发条件（一句话）

## 2. 按工具分组
对每个出现过的来源工具（Cursor/Claude/Windsurf/Augment/Generic）输出一个二级小节，
小节内逐个 skill 给出：
- **应用场景**：基于 description 与正文归纳，禁止编造；
- **触发条件**：基于 when / trigger / paths；缺失则写「未声明，建议补充」；
- **风险/重叠提示**：多个 skill 描述高度相似时标注「与 XXX 可能重叠」。

## 3. 质量评分
- 表格列出：skill 引用（source/name）| 描述清晰度 | 触发明确度 | 正文完备度 | 综合分 | 主要扣分项
- 评分维度均为 0–10 整数；综合分 = 三项均值四舍五入；
- 表格之后，挑选综合分 < 6 的 skill，逐条给出**修复建议**（每条 ≤ 2 句）。

## 4. 缺口分析
- 基于（B）推断项目类型（语言、框架、构建/部署方式）。
- 对照（A）已有 skill 类目，列出**项目应当具备但目前缺失**的 skill 类目；
- 表格列出：缺失类目 | 缺失原因（一句话）| 建议 skill 名 | 建议落地工具；
- 严禁臆测项目不需要的方向；技术栈摘要中无明显证据时写「无明显缺口」即可。

要求：
- 不修改原文、不新增不存在的 skill；
- 全文中文，标准 Markdown；
- 第 3、4 章的表格列与字段名必须**逐字一致**，便于后续解析。`

const skillsCorpusMaxBytes = 200000
const skillsFrameCorpusMaxBytes = 32000

// HandleSkills：CLI 入口，复用 runRepUI 全屏 TUI。
func HandleSkills(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return runRepUI(ctx, "SEEED-CLI SKILLS SCAN", pwd, 0, false, runSkillsWork)
}

// runSkillsWork：扫描 → 采集项目技术栈 → LLM 综合分析 → 落盘。
func runSkillsWork(m *repModel) {
	m.SendLog(bootOKLine("skills scanner online"))
	m.SendLog(bootOKLine(fmt.Sprintf("workspace: %s", m.pwd)))

	m.SendLog(bootWaitLine("scanning SKILL.md across project…"))
	scan, err := funcs.ScanSkills(m.pwd)
	if err != nil {
		m.SendLog(logWarn.Render(err.Error()))
		return
	}
	m.SendLog(bootOKLine(fmt.Sprintf("found %d skill(s) from %d source(s)", len(scan.Skills), len(scan.Detected))))
	for _, sk := range scan.Skills {
		m.SendLog(bootOKLine(fmt.Sprintf("  · [%s] %s  ←  %s", sk.Source, sk.Name, sk.RelPath)))
	}
	for _, p := range scan.SkippedNo {
		rel, _ := filepath.Rel(m.pwd, p)
		if rel == "" {
			rel = p
		}
		m.SendLog(logWarn.Render(fmt.Sprintf("  ! skipped (oversize/parse-fail): %s", filepath.ToSlash(rel))))
	}

	if len(scan.Skills) == 0 {
		m.SendLog(logWarn.Render("no skills detected"))
		return
	}

	m.SendLog(bootWaitLine("collecting project tech-stack corpus…"))
	frameCorpus := CollectFrameCorpus(m.pwd)
	if len(frameCorpus) > skillsFrameCorpusMaxBytes {
		frameCorpus = frameCorpus[:skillsFrameCorpusMaxBytes] + "\n\n_[frame corpus truncated]_\n"
	}
	corpus := buildSkillsCorpus(scan, frameCorpus)
	if len(corpus) > skillsCorpusMaxBytes {
		corpus = corpus[:skillsCorpusMaxBytes] + "\n\n_[corpus truncated]_\n"
	}
	m.SendLog(bootOKLine(fmt.Sprintf("corpus size: %d bytes", len(corpus))))

	m.SendLog(bootWaitLine("llm: analyzing skills…"))
	full, err := m.RunLLMStream(skillsPrompt+"\n\n---\n\n"+corpus, "")
	if err != nil || strings.TrimSpace(full) == "" {
		return
	}

	scores, gaps := parseAnalysisOutput(full)
	rep := &funcs.SkillAnalysisReport{Scan: scan, Scores: scores, Gaps: gaps, Raw: full}
	finalText := renderSkillsMarkdown(rep)

	now := time.Now()
	fileName := "skills-" + now.Format("2006-01-02_15_04_05") + ".md"
	ssPath := filepath.Join(m.pwd, "seeed-cli")
	if _, e := os.Stat(ssPath); os.IsNotExist(e) {
		_ = os.MkdirAll(ssPath, 0755)
	}
	outPath := filepath.Join(ssPath, fileName)
	if werr := os.WriteFile(outPath, []byte(finalText), 0644); werr != nil {
		m.SendLog(logWarn.Render(werr.Error()))
		return
	}
	m.lastSave = outPath
	m.SendLog(bootOKLine("saved: " + outPath))
}

// buildSkillsCorpus：拼接 skill 列表 + 项目技术栈摘要。
func buildSkillsCorpus(scan *funcs.SkillScanResult, frameCorpus string) string {
	var sb strings.Builder
	sb.WriteString("### (A) Discovered skills\n\n")
	for i, sk := range scan.Skills {
		sb.WriteString(fmt.Sprintf("#### [%d] %s/%s\n\n", i+1, sk.Source, sk.Name))
		sb.WriteString("- source: " + string(sk.Source) + "\n")
		sb.WriteString("- name: " + sk.Name + "\n")
		sb.WriteString("- path: " + sk.RelPath + "\n")
		if sk.Description != "" {
			sb.WriteString("- description: " + sk.Description + "\n")
		}
		if sk.Trigger != "" {
			sb.WriteString("- trigger: " + sk.Trigger + "\n")
		}
		if len(sk.Frontmatter) > 0 {
			sb.WriteString("- frontmatter:\n")
			for k, v := range sk.Frontmatter {
				sb.WriteString("    " + k + ": " + v + "\n")
			}
		}
		sb.WriteString("\n```markdown\n")
		sb.WriteString(sk.Body)
		sb.WriteString("\n```\n\n")
	}
	sb.WriteString("\n### (B) Project tech-stack snapshot\n\n")
	sb.WriteString(frameCorpus)
	return sb.String()
}


// parseAnalysisOutput：从 LLM Markdown 中尽力提取「质量评分」「缺口分析」两张表。
// 解析失败时返回空切片；Raw 字段始终承载完整 LLM 输出。
func parseAnalysisOutput(raw string) ([]funcs.SkillQualityScore, []funcs.SkillGapSuggestion) {
	scores := parseScoreTable(extractSection(raw, "质量评分"))
	gaps := parseGapTable(extractSection(raw, "缺口分析"))
	return scores, gaps
}

var sectionHeadRE = regexp.MustCompile(`(?m)^##\s+\d+\.\s+`)

// extractSection：按 "## N. <title>" 切片，返回该章节正文（不含下章）。
func extractSection(raw, title string) string {
	idxs := sectionHeadRE.FindAllStringIndex(raw, -1)
	if len(idxs) == 0 {
		return ""
	}
	for i, m := range idxs {
		head := raw[m[0]:]
		nl := strings.Index(head, "\n")
		if nl < 0 {
			continue
		}
		line := head[:nl]
		if !strings.Contains(line, title) {
			continue
		}
		body := head[nl+1:]
		if i+1 < len(idxs) {
			next := idxs[i+1][0] - m[0]
			if next > nl+1 && next-nl-1 <= len(body) {
				body = body[:next-nl-1]
			}
		}
		return body
	}
	return ""
}

var tableRowRE = regexp.MustCompile(`^\s*\|`)

// parseScoreTable：扫描章节内 Markdown 表格，提取每行 6 列评分。
func parseScoreTable(section string) []funcs.SkillQualityScore {
	var out []funcs.SkillQualityScore
	for _, line := range strings.Split(section, "\n") {
		if !tableRowRE.MatchString(line) {
			continue
		}
		cells := splitMDRow(line)
		if len(cells) < 6 {
			continue
		}
		if strings.Contains(cells[0], "skill") || strings.HasPrefix(cells[0], "---") || strings.HasPrefix(cells[0], ":-") {
			continue
		}
		desc, ok1 := parseIntCell(cells[1])
		trig, ok2 := parseIntCell(cells[2])
		body, ok3 := parseIntCell(cells[3])
		over, ok4 := parseIntCell(cells[4])
		if !(ok1 && ok2 && ok3 && ok4) {
			continue
		}
		issues := splitIssues(cells[5])
		out = append(out, funcs.SkillQualityScore{
			SkillRef: cells[0], Description: desc, Trigger: trig, Body: body, Overall: over, Issues: issues,
		})
	}
	return out
}

// parseGapTable：扫描缺口分析章节表格，提取每行 4 列。
func parseGapTable(section string) []funcs.SkillGapSuggestion {
	var out []funcs.SkillGapSuggestion
	for _, line := range strings.Split(section, "\n") {
		if !tableRowRE.MatchString(line) {
			continue
		}
		cells := splitMDRow(line)
		if len(cells) < 4 {
			continue
		}
		if strings.Contains(cells[0], "缺失类目") || strings.HasPrefix(cells[0], "---") || strings.HasPrefix(cells[0], ":-") {
			continue
		}
		out = append(out, funcs.SkillGapSuggestion{
			Category:   cells[0],
			Reason:     cells[1],
			SkillName:  cells[2],
			SuggestFor: parseSuggestFor(cells[3]),
		})
	}
	return out
}

func splitMDRow(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	parts := strings.Split(t, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseIntCell(s string) (int, bool) {
	s = strings.TrimSpace(s)
	digits := strings.Builder{}
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		} else if digits.Len() > 0 {
			break
		}
	}
	if digits.Len() == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(digits.String())
	if err != nil {
		return 0, false
	}
	return n, true
}

func splitIssues(s string) []string {
	if s == "" || s == "-" {
		return nil
	}
	for _, sep := range []string{"；", ";", "、", "，"} {
		if strings.Contains(s, sep) {
			parts := strings.Split(s, sep)
			var out []string
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					out = append(out, t)
				}
			}
			return out
		}
	}
	return []string{s}
}

func parseSuggestFor(s string) []funcs.SkillSource {
	low := strings.ToLower(s)
	var out []funcs.SkillSource
	for src := range funcs.SkillTargetDirMap {
		if strings.Contains(low, string(src)) {
			out = append(out, src)
		}
	}
	return out
}


// renderSkillsMarkdown：拼装最终报告（扫描元信息 + LLM 正文 + JSON 附录）。
func renderSkillsMarkdown(rep *funcs.SkillAnalysisReport) string {
	var sb strings.Builder
	sb.WriteString("# Skills 扫描报告\n\n")
	sb.WriteString("> 由 seeed-cli skills-scan 自动生成，可人工修订。\n\n")

	sb.WriteString("## 扫描元信息\n\n")
	sb.WriteString("| 字段 | 值 |\n|---|---|\n")
	sb.WriteString("| 项目根 | `" + filepath.ToSlash(rep.Scan.Root) + "` |\n")
	sb.WriteString(fmt.Sprintf("| 发现 SKILL.md | %d |\n", len(rep.Scan.Skills)))
	sb.WriteString(fmt.Sprintf("| 来源工具数 | %d |\n", len(rep.Scan.Detected)))
	sb.WriteString("| 扫描时间 | " + time.Now().Format("2006-01-02 15:04:05") + " |\n\n")

	sb.WriteString("## 已发现的 SKILL.md 文件\n\n")
	sb.WriteString("| # | 来源 | 名称 | 相对路径 |\n|---|---|---|---|\n")
	for i, sk := range rep.Scan.Skills {
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | `%s` |\n", i+1, sk.Source, sk.Name, sk.RelPath))
	}
	if len(rep.Scan.SkippedNo) > 0 {
		sb.WriteString("\n_跳过（超大或解析失败）_：\n")
		for _, p := range rep.Scan.SkippedNo {
			rel, _ := filepath.Rel(rep.Scan.Root, p)
			if rel == "" {
				rel = p
			}
			sb.WriteString("- `" + filepath.ToSlash(rel) + "`\n")
		}
	}
	sb.WriteString("\n---\n\n")
	sb.WriteString(strings.TrimSpace(rep.Raw))
	sb.WriteString("\n\n---\n\n")

	sb.WriteString("## 附录：结构化结果\n\n")
	if scoresJSON, err := json.MarshalIndent(rep.Scores, "", "  "); err == nil {
		sb.WriteString("### scores\n\n```json\n")
		sb.Write(scoresJSON)
		sb.WriteString("\n```\n\n")
	}
	if gapsJSON, err := json.MarshalIndent(rep.Gaps, "", "  "); err == nil {
		sb.WriteString("### gaps\n\n```json\n")
		sb.Write(gapsJSON)
		sb.WriteString("\n```\n")
	}
	return sb.String()
}
