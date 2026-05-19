// skills-sync：交互或非交互地把源 skill 同步到目标 vibe coding 工具，生成 SKILL.md 包并落盘报告。
package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/funcs"
)

// HandleSkillsSync：CLI 入口；flag 缺省则进入交互式选择。
func HandleSkillsSync(ctx context.Context, cmd *cli.Command) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	scan, err := funcs.ScanSkills(pwd)
	if err != nil {
		return err
	}
	if len(scan.Skills) == 0 {
		return errors.New("no skills/rules/workflows detected under .cursor/.claude/.windsurf/.augment")
	}

	targetStr := strings.TrimSpace(cmd.String("target"))
	skillsCSV := strings.TrimSpace(cmd.String("skills"))
	overwrite := cmd.Bool("overwrite")
	yes := cmd.Bool("yes")

	var sources []funcs.Skill
	if skillsCSV != "" {
		sources, err = filterSkillsByNames(scan.Skills, splitCSV(skillsCSV))
		if err != nil {
			return err
		}
	} else {
		sources, err = SelectSourceSkillsInteractive(scan.Skills)
		if err != nil {
			return err
		}
	}

	var target funcs.SkillSource
	if targetStr != "" {
		t := funcs.SkillSource(strings.ToLower(targetStr))
		if _, ok := funcs.SkillTargetDirMap[t]; !ok {
			return fmt.Errorf("invalid --target: %s (allowed: cursor|claude|windsurf|augment)", targetStr)
		}
		target = t
	} else {
		target, err = SelectTargetInteractive()
		if err != nil {
			return err
		}
	}

	plan, err := BuildSyncPlan(pwd, sources, target, overwrite)
	if err != nil {
		return err
	}

	printSyncPreview(plan)
	if !yes {
		if !confirmStdin("Proceed?") {
			return errors.New("cancelled by user")
		}
	}

	if err := ExecuteSyncPlan(plan); err != nil {
		return err
	}
	printSyncResult(plan)

	return saveSyncReport(pwd, plan)
}

// splitCSV：逗号 / 中文逗号皆识别，去空白。
func splitCSV(s string) []string {
	s = strings.ReplaceAll(s, "，", ",")
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// filterSkillsByNames：按 token 匹配资源；支持的 token 形式（任一命中即可）：
//   - "<name>"                     仅按名称匹配（含同名歧义时全部纳入）
//   - "<kind>/<name>"              kind ∈ skill|rule|workflow
//   - "<source>/<name>"            source ∈ cursor|claude|windsurf|augment
//   - "<source>/<kind>/<name>"     最精确，避免歧义
func filterSkillsByNames(assets []funcs.Skill, tokens []string) ([]funcs.Skill, error) {
	if len(tokens) == 0 {
		return nil, errors.New("--skills is empty")
	}
	var out []funcs.Skill
	matched := map[string]bool{}
	for _, tk := range tokens {
		hit := false
		for _, a := range assets {
			if assetMatchesToken(a, tk) {
				out = append(out, a)
				hit = true
			}
		}
		if hit {
			matched[tk] = true
		}
	}
	var missing []string
	for _, tk := range tokens {
		if !matched[tk] {
			missing = append(missing, tk)
		}
	}
	if len(missing) > 0 {
		return out, fmt.Errorf("assets not found: %s", strings.Join(missing, ", "))
	}
	return out, nil
}

// assetMatchesToken：判断单个资源是否匹配某个 token（按 `/` 拆分后从右到左匹配）。
func assetMatchesToken(a funcs.Skill, token string) bool {
	parts := strings.Split(token, "/")
	if len(parts) == 0 {
		return false
	}
	if a.Name != parts[len(parts)-1] {
		return false
	}
	for _, p := range parts[:len(parts)-1] {
		switch p {
		case string(a.Kind), string(a.Source):
		default:
			return false
		}
	}
	return true
}

// BuildSyncPlan：仅计算计划，不写盘。
func BuildSyncPlan(projectRoot string, sources []funcs.Skill, target funcs.SkillSource, overwrite bool) (*funcs.SkillSyncPlan, error) {
	if len(sources) == 0 {
		return nil, errors.New("no source asset selected")
	}
	dir, ok := funcs.SkillTargetDirMap[target]
	if !ok {
		return nil, fmt.Errorf("unsupported target: %s", target)
	}
	return &funcs.SkillSyncPlan{
		Sources:     sources,
		Target:      target,
		ProjectRoot: projectRoot,
		TargetRoot:  filepath.Join(projectRoot, filepath.FromSlash(dir)),
		Overwrite:   overwrite,
	}, nil
}


// ExecuteSyncPlan：按目标 dialect 翻译 frontmatter，逐条资源（skill/rule/workflow）写入磁盘；
// skill 资源额外复制源文件夹下的 references/ 与 scripts/ 子目录。
func ExecuteSyncPlan(plan *funcs.SkillSyncPlan) error {
	adapter, err := funcs.GetAdapter(plan.Target)
	if err != nil {
		return err
	}
	for _, src := range plan.Sources {
		rel := funcs.TargetAssetRelPath(src, plan.Target)
		if rel == "" {
			plan.Failed = append(plan.Failed, fmt.Sprintf("[%s/%s] %s: unsupported kind", src.Source, src.Kind, src.Name))
			continue
		}
		outFile := filepath.Join(plan.ProjectRoot, filepath.FromSlash(rel))

		if _, statErr := os.Stat(outFile); statErr == nil && !plan.Overwrite {
			plan.Skipped = append(plan.Skipped, rel)
			continue
		}
		if mkErr := os.MkdirAll(filepath.Dir(outFile), 0755); mkErr != nil {
			plan.Failed = append(plan.Failed, rel+": "+mkErr.Error())
			continue
		}
		fm, body := adapter.Translate(src)
		content := funcs.RenderSkillFile(fm, body)
		if werr := os.WriteFile(outFile, []byte(content), 0644); werr != nil {
			plan.Failed = append(plan.Failed, rel+": "+werr.Error())
			continue
		}
		plan.Written = append(plan.Written, rel)

		if src.Kind == funcs.AssetKindSkill {
			copySkillCompanions(plan, src, outFile)
		}
	}
	return nil
}

// copySkillCompanions：把源 skill 文件夹下的 references/ 与 scripts/ 子目录复制到目标 skill 文件夹。
// 每个被复制的文件以「相对项目根」追加进 plan.Written/Skipped/Failed，便于报告清单可追溯。
func copySkillCompanions(plan *funcs.SkillSyncPlan, src funcs.Skill, dstSkillFile string) {
	srcDir := filepath.Dir(src.Path)
	dstDir := filepath.Dir(dstSkillFile)
	for _, sub := range []string{"references", "scripts"} {
		srcSub := filepath.Join(srcDir, sub)
		info, err := os.Stat(srcSub)
		if err != nil || !info.IsDir() {
			continue
		}
		dstSub := filepath.Join(dstDir, sub)
		_ = filepath.Walk(srcSub, func(path string, fi os.FileInfo, werr error) error {
			if werr != nil || fi == nil {
				return nil
			}
			relInside, _ := filepath.Rel(srcSub, path)
			target := filepath.Join(dstSub, relInside)
			relForReport := mustRel(plan.ProjectRoot, target)
			if fi.IsDir() {
				if mkErr := os.MkdirAll(target, 0755); mkErr != nil {
					plan.Failed = append(plan.Failed, relForReport+": "+mkErr.Error())
				}
				return nil
			}
			if _, statErr := os.Stat(target); statErr == nil && !plan.Overwrite {
				plan.Skipped = append(plan.Skipped, relForReport)
				return nil
			}
			if mkErr := os.MkdirAll(filepath.Dir(target), 0755); mkErr != nil {
				plan.Failed = append(plan.Failed, relForReport+": "+mkErr.Error())
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				plan.Failed = append(plan.Failed, relForReport+": "+rerr.Error())
				return nil
			}
			if werr := os.WriteFile(target, data, 0644); werr != nil {
				plan.Failed = append(plan.Failed, relForReport+": "+werr.Error())
				return nil
			}
			plan.Written = append(plan.Written, relForReport)
			return nil
		})
	}
}

// mustRel：算相对路径，失败时退回绝对路径并统一为正斜杠形式。
func mustRel(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == "" {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(rel)
}

// printSyncPreview：写盘前打印计划摘要。
func printSyncPreview(plan *funcs.SkillSyncPlan) {
	fmt.Printf("\nTarget : %s\n", plan.Target)
	fmt.Printf("Sources: %d asset(s)\n", len(plan.Sources))
	for _, s := range plan.Sources {
		dst := funcs.TargetAssetRelPath(s, plan.Target)
		fmt.Printf("  - [%s/%s] %s\n      src: %s\n      dst: %s\n", s.Source, s.Kind, s.Name, s.RelPath, dst)
	}
	if plan.Overwrite {
		fmt.Println("Overwrite: ON")
	} else {
		fmt.Println("Overwrite: off (existing files will be skipped)")
	}
}

// printSyncResult：写盘后打印 Written/Skipped/Failed 三段。
func printSyncResult(plan *funcs.SkillSyncPlan) {
	fmt.Println()
	fmt.Printf("Written: %d\n", len(plan.Written))
	for _, p := range plan.Written {
		fmt.Println("  + " + p)
	}
	if len(plan.Skipped) > 0 {
		fmt.Printf("Skipped: %d (already exist, pass --overwrite to replace)\n", len(plan.Skipped))
		for _, p := range plan.Skipped {
			fmt.Println("  · " + p)
		}
	}
	if len(plan.Failed) > 0 {
		fmt.Printf("Failed : %d\n", len(plan.Failed))
		for _, p := range plan.Failed {
			fmt.Println("  ! " + p)
		}
	}
}

// confirmStdin：读取一行，y/Y/回车视为确认。
func confirmStdin(prompt string) bool {
	fmt.Printf("\n%s [Y/n]: ", prompt)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	t := strings.ToLower(strings.TrimSpace(line))
	return t == "" || t == "y" || t == "yes"
}

// saveSyncReport：把同步结果以 Markdown 写入 seeed-cli/skills-sync-时间戳.md。
func saveSyncReport(projectRoot string, plan *funcs.SkillSyncPlan) error {
	now := time.Now()
	fileName := "skills-sync-" + now.Format("2006-01-02_15_04_05") + ".md"
	ssPath := filepath.Join(projectRoot, "seeed-cli")
	if _, e := os.Stat(ssPath); os.IsNotExist(e) {
		if mkErr := os.MkdirAll(ssPath, 0755); mkErr != nil {
			return mkErr
		}
	}
	outPath := filepath.Join(ssPath, fileName)
	content := renderSyncReport(plan)
	if werr := os.WriteFile(outPath, []byte(content), 0644); werr != nil {
		return werr
	}
	fmt.Println("\nReport: " + outPath)
	return nil
}

// renderSyncReport：拼装报告 Markdown。
func renderSyncReport(plan *funcs.SkillSyncPlan) string {
	var sb strings.Builder
	sb.WriteString("# Skills 同步报告\n\n")
	sb.WriteString("> 由 seeed-cli skills-sync 自动生成。\n\n")
	sb.WriteString("## 概览\n\n")
	sb.WriteString(fmt.Sprintf("- 目标工具：`%s`\n", plan.Target))
	sb.WriteString(fmt.Sprintf("- 项目根：`%s`\n", filepath.ToSlash(plan.ProjectRoot)))
	sb.WriteString(fmt.Sprintf("- 覆盖模式：%v\n", plan.Overwrite))
	sb.WriteString(fmt.Sprintf("- 同步时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	sb.WriteString("## 源 → 目标\n\n")
	sb.WriteString("| 源 source | 类别 | 名称 | 源路径 | 目标路径 |\n|---|---|---|---|---|\n")
	for _, s := range plan.Sources {
		dst := funcs.TargetAssetRelPath(s, plan.Target)
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | `%s` | `%s` |\n", s.Source, s.Kind, s.Name, s.RelPath, dst))
	}
	sb.WriteString("\n")

	sb.WriteString("> skill 资源除 SKILL.md 外，源文件夹下若存在 `references/` 或 `scripts/` 目录会一并复制到目标 skill 文件夹下；下述清单逐项列出实际被写入/跳过/失败的相对路径。\n\n")

	sb.WriteString("## 结果\n\n")
	writeBlock(&sb, "Written", plan.Written)
	writeBlock(&sb, "Skipped", plan.Skipped)
	writeBlock(&sb, "Failed", plan.Failed)
	return sb.String()
}

func writeBlock(sb *strings.Builder, title string, items []string) {
	sb.WriteString(fmt.Sprintf("### %s（%d）\n\n", title, len(items)))
	if len(items) == 0 {
		sb.WriteString("_(无)_\n\n")
		return
	}
	for _, it := range items {
		sb.WriteString("- `" + it + "`\n")
	}
	sb.WriteString("\n")
}
