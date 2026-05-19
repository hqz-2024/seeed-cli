// skills_scan：限定扫描 .cursor/.claude/.windsurf/.augment 四个工具根目录下的 skills/、rules/、workflows/ 三类资源。
package funcs

import (
	"os"
	"path/filepath"
	"strings"
)

// SkillSource 标识资源来源工具。
type SkillSource string

const (
	SourceCursor   SkillSource = "cursor"
	SourceClaude   SkillSource = "claude"
	SourceWindsurf SkillSource = "windsurf"
	SourceAugment  SkillSource = "augment"
	SourceGeneric  SkillSource = "generic"
)

// AssetKind：资源类别（skill / rule / workflow）。
type AssetKind string

const (
	AssetKindSkill    AssetKind = "skill"
	AssetKindRule     AssetKind = "rule"
	AssetKindWorkflow AssetKind = "workflow"
)

// SourceRootDirs：来源工具 → 工具配置根目录（相对项目根）。
var SourceRootDirs = map[SkillSource]string{
	SourceCursor:   ".cursor",
	SourceClaude:   ".claude",
	SourceWindsurf: ".windsurf",
	SourceAugment:  ".augment",
}

// AssetKindDirs：资源类别 → 子目录名（相对工具根）。
var AssetKindDirs = map[AssetKind]string{
	AssetKindSkill:    "skills",
	AssetKindRule:     "rules",
	AssetKindWorkflow: "workflows",
}

// SkillTargetDirMap：已知工具 → skills 写盘根目录（相对项目根），供 skills-sync 使用。
var SkillTargetDirMap = map[SkillSource]string{
	SourceCursor:   ".cursor/skills",
	SourceClaude:   ".claude/skills",
	SourceWindsurf: ".windsurf/skills",
	SourceAugment:  ".augment/skills",
}

// SourcePathMarkers：路径子串 → source（仅作为兜底推断使用）。
var SourcePathMarkers = map[string]SkillSource{
	".cursor/":   SourceCursor,
	".claude/":   SourceClaude,
	".windsurf/": SourceWindsurf,
	".augment/":  SourceAugment,
}

// Skill 表示一个被发现并解析后的资源（skill / rule / workflow 通用）。
type Skill struct {
	Source      SkillSource
	Kind        AssetKind
	Name        string
	Path        string
	RelPath     string
	SubPath     string // 相对 {tool}/{kindDir}/ 的路径；skill 为子文件夹名，rule/workflow 为含扩展名的相对文件路径
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
	Sources     []Skill
	Target      SkillSource
	ProjectRoot string // 项目根绝对路径
	TargetRoot  string // 目标工具的 skills/ 写盘绝对路径（仅供预览与兼容旧报告）
	Overwrite   bool
	Written     []string
	Skipped     []string
	Failed      []string
}

// ScanSkills：仅扫描 .cursor/.claude/.windsurf/.augment 四个工具根目录下的 skills/、rules/、workflows/ 三类资源。
func ScanSkills(projectRoot string) (*SkillScanResult, error) {
	res := &SkillScanResult{Root: projectRoot}
	seen := map[SkillSource]struct{}{}

	for _, src := range []SkillSource{SourceCursor, SourceClaude, SourceWindsurf, SourceAugment} {
		toolRoot := filepath.Join(projectRoot, SourceRootDirs[src])
		if info, err := os.Stat(toolRoot); err != nil || !info.IsDir() {
			continue
		}
		collectSkillFolders(filepath.Join(toolRoot, AssetKindDirs[AssetKindSkill]), projectRoot, src, res, seen)
		collectMarkdownFiles(filepath.Join(toolRoot, AssetKindDirs[AssetKindRule]), projectRoot, src, AssetKindRule, res, seen)
		collectMarkdownFiles(filepath.Join(toolRoot, AssetKindDirs[AssetKindWorkflow]), projectRoot, src, AssetKindWorkflow, res, seen)
	}

	for s := range seen {
		res.Detected = append(res.Detected, s)
	}
	return res, nil
}

// collectSkillFolders：遍历 skills/ 下一层子文件夹，每个含 SKILL.md 的文件夹视为一个 skill 单元。
func collectSkillFolders(skillsDir, projectRoot string, source SkillSource, res *SkillScanResult, seen map[SkillSource]struct{}) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		info, statErr := os.Stat(skillMD)
		if statErr != nil || info.IsDir() {
			continue
		}
		if info.Size() > MaxFileSize {
			res.SkippedNo = append(res.SkippedNo, skillMD)
			continue
		}
		sk, perr := ParseAssetFile(skillMD, projectRoot, source, AssetKindSkill)
		if perr != nil || sk == nil {
			res.SkippedNo = append(res.SkippedNo, skillMD)
			continue
		}
		sk.SubPath = e.Name()
		res.Skills = append(res.Skills, *sk)
		seen[source] = struct{}{}
	}
}

// collectMarkdownFiles：递归遍历 rules/ 或 workflows/，将 .md / .mdc 文件收作单条资源。
func collectMarkdownFiles(root, projectRoot string, source SkillSource, kind AssetKind, res *SkillScanResult, seen map[SkillSource]struct{}) {
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return
	}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, werr error) error {
		if werr != nil || info == nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".mdc" {
			return nil
		}
		if info.Size() > MaxFileSize {
			res.SkippedNo = append(res.SkippedNo, path)
			return nil
		}
		sk, perr := ParseAssetFile(path, projectRoot, source, kind)
		if perr != nil || sk == nil {
			res.SkippedNo = append(res.SkippedNo, path)
			return nil
		}
		sub, _ := filepath.Rel(root, path)
		sk.SubPath = filepath.ToSlash(sub)
		res.Skills = append(res.Skills, *sk)
		seen[source] = struct{}{}
		return nil
	})
}

// FilterByKind：按资源类别过滤扫描结果中的 Skills 切片。
func (r *SkillScanResult) FilterByKind(kind AssetKind) []Skill {
	var out []Skill
	for _, s := range r.Skills {
		if s.Kind == kind {
			out = append(out, s)
		}
	}
	return out
}

// CountByKind：按资源类别统计扫描结果数量。
func (r *SkillScanResult) CountByKind(kind AssetKind) int {
	n := 0
	for _, s := range r.Skills {
		if s.Kind == kind {
			n++
		}
	}
	return n
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

// ParseAssetFile：解析单个资源文件，按 kind 决定 Name 取值规则。
func ParseAssetFile(absPath, projectRoot string, source SkillSource, kind AssetKind) (*Skill, error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	fm, body := parseFrontmatter(string(raw))
	rel, _ := filepath.Rel(projectRoot, absPath)
	if rel == "" {
		rel = absPath
	}
	const maxBody = 8 * 1024
	if len(body) > maxBody {
		body = body[:maxBody] + "\n\n_[truncated]_\n"
	}
	var defaultName string
	if kind == AssetKindSkill {
		defaultName = filepath.Base(filepath.Dir(absPath))
	} else {
		base := filepath.Base(absPath)
		defaultName = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return &Skill{
		Source:      source,
		Kind:        kind,
		Name:        resolveSkillName(defaultName, fm),
		Path:        absPath,
		RelPath:     filepath.ToSlash(rel),
		Description: fm["description"],
		Trigger:     resolveTrigger(fm),
		Frontmatter: fm,
		Body:        body,
	}, nil
}

// ParseSkillFile：旧调用入口，等价于 ParseAssetFile(..., AssetKindSkill)，保留以兼容。
func ParseSkillFile(absPath, projectRoot string, source SkillSource) (*Skill, error) {
	return ParseAssetFile(absPath, projectRoot, source, AssetKindSkill)
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
