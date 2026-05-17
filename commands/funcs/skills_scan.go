// skills_scan：自项目根全工程递归扫描 SKILL.md（大小写敏感），并按路径推断来源工具。
package funcs

import (
	"os"
	"path/filepath"
	"strings"
)

// SkillSource 标识 skill 来源工具。
type SkillSource string

const (
	SourceCursor   SkillSource = "cursor"
	SourceClaude   SkillSource = "claude"
	SourceWindsurf SkillSource = "windsurf"
	SourceAugment  SkillSource = "augment"
	SourceGeneric  SkillSource = "generic"
)

// SkillTargetDirMap：已知工具 → 目标写盘根目录（相对项目根）。
var SkillTargetDirMap = map[SkillSource]string{
	SourceCursor:   ".cursor/skills",
	SourceClaude:   ".claude/skills",
	SourceWindsurf: ".windsurf/skills",
	SourceAugment:  ".augment/skills",
}

// SourcePathMarkers：路径子串 → source。
var SourcePathMarkers = map[string]SkillSource{
	".cursor/skills/":   SourceCursor,
	".claude/skills/":   SourceClaude,
	".windsurf/skills/": SourceWindsurf,
	".augment/skills/":  SourceAugment,
}

// Skill 表示一个被发现并解析后的 skill。
type Skill struct {
	Source      SkillSource
	Name        string
	Path        string
	RelPath     string
	Description string
	Trigger     string
	Frontmatter map[string]string
	Body        string
}

// SkillScanResult 一次扫描的聚合结果。
type SkillScanResult struct {
	Root      string
	Skills    []Skill
	Detected  []SkillSource
	SkippedNo []string
}

// SkillQualityScore 单个 skill 的质量评分（0–10 整数）。
type SkillQualityScore struct {
	SkillRef    string
	Description int
	Trigger     int
	Body        int
	Overall     int
	Issues      []string
}

// SkillGapSuggestion 缺口分析中的一条建议。
type SkillGapSuggestion struct {
	Category   string
	Reason     string
	SkillName  string
	SuggestFor []SkillSource
}

// SkillAnalysisReport 一次 skills-scan 的完整产物。
type SkillAnalysisReport struct {
	Scan   *SkillScanResult
	Scores []SkillQualityScore
	Gaps   []SkillGapSuggestion
	Raw    string
}

// SkillSyncPlan 一次 skills-sync 的执行计划与结果。
type SkillSyncPlan struct {
	Sources    []Skill
	Target     SkillSource
	TargetRoot string
	Overwrite  bool
	Written    []string
	Skipped    []string
	Failed     []string
}

// ScanSkills：自项目根递归扫描整个工程，收集所有 SKILL.md（大小写敏感）。
func ScanSkills(projectRoot string) (*SkillScanResult, error) {
	res := &SkillScanResult{Root: projectRoot}
	seen := map[SkillSource]struct{}{}

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) != "SKILL.md" {
			return nil
		}
		if info.Size() > MaxFileSize {
			res.SkippedNo = append(res.SkippedNo, path)
			return nil
		}
		src := InferSourceFromPath(path)
		sk, perr := ParseSkillFile(path, projectRoot, src)
		if perr != nil || sk == nil {
			res.SkippedNo = append(res.SkippedNo, path)
			return nil
		}
		res.Skills = append(res.Skills, *sk)
		seen[src] = struct{}{}
		return nil
	})
	if err != nil {
		return res, err
	}

	for s := range seen {
		res.Detected = append(res.Detected, s)
	}
	return res, nil
}

// InferSourceFromPath：将路径分隔符统一为正斜杠后按 SourcePathMarkers 匹配子串。
func InferSourceFromPath(absPath string) SkillSource {
	p := filepath.ToSlash(absPath)
	for marker, src := range SourcePathMarkers {
		if strings.Contains(p, marker) {
			return src
		}
	}
	return SourceGeneric
}

// ParseSkillFile：解析单个 SKILL.md，正文超 8KB 截断以控总语料。
func ParseSkillFile(absPath, projectRoot string, source SkillSource) (*Skill, error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	fm, body := parseFrontmatter(string(raw))
	folder := filepath.Base(filepath.Dir(absPath))
	rel, _ := filepath.Rel(projectRoot, absPath)
	if rel == "" {
		rel = absPath
	}
	const maxBody = 8 * 1024
	if len(body) > maxBody {
		body = body[:maxBody] + "\n\n_[truncated]_\n"
	}
	return &Skill{
		Source:      source,
		Name:        resolveSkillName(folder, fm),
		Path:        absPath,
		RelPath:     filepath.ToSlash(rel),
		Description: fm["description"],
		Trigger:     resolveTrigger(fm),
		Frontmatter: fm,
		Body:        body,
	}, nil
}

// parseFrontmatter：识别首行 "---" 至下一行 "---" 之间的 key: value（轻量级，不引入 yaml 库）。
func parseFrontmatter(raw string) (map[string]string, string) {
	fm := map[string]string{}
	s := strings.TrimLeft(raw, "\ufeff")
	if !strings.HasPrefix(s, "---") {
		return fm, raw
	}
	lines := strings.Split(s, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return fm, raw
	}
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx < 0 {
		return fm, raw
	}
	for _, line := range lines[1:endIdx] {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		idx := strings.Index(trim, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(trim[:idx])
		val := strings.TrimSpace(trim[idx+1:])
		val = strings.Trim(val, "\"'")
		if key != "" {
			fm[strings.ToLower(key)] = val
		}
	}
	body := strings.Join(lines[endIdx+1:], "\n")
	return fm, strings.TrimLeft(body, "\n")
}

// resolveSkillName：优先 frontmatter.name；否则回退所在文件夹名。
func resolveSkillName(folderName string, fm map[string]string) string {
	if v, ok := fm["name"]; ok {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return folderName
}

// resolveTrigger：合并 when / trigger / paths 三个候选字段。
func resolveTrigger(fm map[string]string) string {
	var parts []string
	for _, k := range []string{"when", "trigger", "paths"} {
		if v, ok := fm[k]; ok {
			if t := strings.TrimSpace(v); t != "" {
				parts = append(parts, t)
			}
		}
	}
	return strings.Join(parts, " | ")
}
