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
		return errors.New("no skills detected in project")
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

// filterSkillsByNames：按 name 精确匹配；多个同名 skill 全部纳入。
func filterSkillsByNames(skills []funcs.Skill, names []string) ([]funcs.Skill, error) {
	if len(names) == 0 {
		return nil, errors.New("--skills is empty")
	}
	want := map[string]struct{}{}
	for _, n := range names {
		want[n] = struct{}{}
	}
	var out []funcs.Skill
	matched := map[string]bool{}
	for _, sk := range skills {
		if _, ok := want[sk.Name]; ok {
			out = append(out, sk)
			matched[sk.Name] = true
		}
	}
	var missing []string
	for n := range want {
		if !matched[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return out, fmt.Errorf("skills not found: %s", strings.Join(missing, ", "))
	}
	return out, nil
}

// BuildSyncPlan：仅计算计划，不写盘。
func BuildSyncPlan(projectRoot string, sources []funcs.Skill, target funcs.SkillSource, overwrite bool) (*funcs.SkillSyncPlan, error) {
	if len(sources) == 0 {
		return nil, errors.New("no source skill selected")
	}
	dir, ok := funcs.SkillTargetDirMap[target]
	if !ok {
		return nil, fmt.Errorf("unsupported target: %s", target)
	}
	return &funcs.SkillSyncPlan{
		Sources:    sources,
		Target:     target,
		TargetRoot: filepath.Join(projectRoot, filepath.FromSlash(dir)),
		Overwrite:  overwrite,
	}, nil
}


// ExecuteSyncPlan：按目标 dialect 翻译 frontmatter，逐个 skill 写入磁盘。
func ExecuteSyncPlan(plan *funcs.SkillSyncPlan) error {
	adapter, err := funcs.GetAdapter(plan.Target)
	if err != nil {
		return err
	}
	if mkErr := os.MkdirAll(plan.TargetRoot, 0755); mkErr != nil {
		return mkErr
	}
	for _, src := range plan.Sources {
		fm, body := adapter.Translate(src)
		content := funcs.RenderSkillFile(fm, body)
		outDir := filepath.Join(plan.TargetRoot, src.Name)
		outFile := filepath.Join(outDir, "SKILL.md")
		rel := filepath.ToSlash(filepath.Join(adapter.SkillDirRel(), src.Name, "SKILL.md"))

		if _, statErr := os.Stat(outFile); statErr == nil && !plan.Overwrite {
			plan.Skipped = append(plan.Skipped, rel)
			continue
		}
		if mkErr := os.MkdirAll(outDir, 0755); mkErr != nil {
			plan.Failed = append(plan.Failed, rel+": "+mkErr.Error())
			continue
		}
		if werr := os.WriteFile(outFile, []byte(content), 0644); werr != nil {
			plan.Failed = append(plan.Failed, rel+": "+werr.Error())
			continue
		}
		plan.Written = append(plan.Written, rel)
	}
	return nil
}

// printSyncPreview：写盘前打印计划摘要。
func printSyncPreview(plan *funcs.SkillSyncPlan) {
	fmt.Printf("\nTarget : %s  →  %s\n", plan.Target, filepath.ToSlash(plan.TargetRoot))
	fmt.Printf("Sources: %d skill(s)\n", len(plan.Sources))
	for _, s := range plan.Sources {
		fmt.Printf("  - [%s] %s  (%s)\n", s.Source, s.Name, s.RelPath)
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
	sb.WriteString(fmt.Sprintf("- 目标目录：`%s`\n", filepath.ToSlash(plan.TargetRoot)))
	sb.WriteString(fmt.Sprintf("- 覆盖模式：%v\n", plan.Overwrite))
	sb.WriteString(fmt.Sprintf("- 同步时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	sb.WriteString("## 源 → 目标\n\n")
	sb.WriteString("| 源 source | 源 name | 源路径 | 目标路径 |\n|---|---|---|---|\n")
	for _, s := range plan.Sources {
		dst := filepath.ToSlash(filepath.Join(funcs.SkillTargetDirMap[plan.Target], s.Name, "SKILL.md"))
		sb.WriteString(fmt.Sprintf("| %s | %s | `%s` | `%s` |\n", s.Source, s.Name, s.RelPath, dst))
	}
	sb.WriteString("\n")

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
