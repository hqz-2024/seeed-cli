// skills-pull：从内嵌的 Top-50 索引中检索 GitHub 上的高星 skill，下载到临时目录后复用 sync 通道安装到目标工具。
package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"seeed-cli/commands/funcs"
)

// HandleSkillsPull：CLI 入口；flag 缺省进入交互式多选。
func HandleSkillsPull(ctx context.Context, cmd *cli.Command) error {
	idx, err := funcs.LoadEmbeddedIndex()
	if err != nil {
		return err
	}

	search := strings.TrimSpace(cmd.String("search"))
	list := cmd.Bool("list")
	idsCSV := strings.TrimSpace(cmd.String("ids"))
	targetStr := strings.TrimSpace(cmd.String("target"))
	overwrite := cmd.Bool("overwrite")
	yes := cmd.Bool("yes")

	entries := funcs.SearchIndex(idx.Skills, search)
	if len(entries) == 0 {
		return fmt.Errorf("no skill matches query: %q", search)
	}

	if list {
		printIndexList(entries, idx.Updated)
		return nil
	}

	var selected []funcs.IndexEntry
	if idsCSV != "" {
		tokens := splitCSV(idsCSV)
		hit, miss := funcs.FindByTokens(entries, tokens)
		if len(miss) > 0 {
			return fmt.Errorf("unknown skill id(s): %s", strings.Join(miss, ", "))
		}
		selected = hit
	} else {
		selected, err = SelectRemoteEntriesInteractive(entries)
		if err != nil {
			return err
		}
	}
	if len(selected) == 0 {
		return errors.New("no skill selected")
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
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

	fmt.Printf("\nDownloading %d skill(s) from GitHub …\n", len(selected))
	results, errs := funcs.DownloadRemoteSkills(selected)
	for _, e := range errs {
		fmt.Println("  ! " + e.Error())
	}
	if len(results) == 0 {
		return errors.New("all downloads failed")
	}
	defer cleanupTempDirs(results)

	sources := make([]funcs.Skill, 0, len(results))
	for _, r := range results {
		sources = append(sources, r.Skill)
		fmt.Printf("  + [%s] %s (%d file%s)\n", r.Entry.ID, r.Entry.Name, r.FileCount, plural(r.FileCount))
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

// printIndexList：把搜索结果以纯文本表的形式打印到 stdout，给 --list 使用。
func printIndexList(entries []funcs.IndexEntry, updated string) {
	fmt.Printf("Skills index (updated: %s, total: %d)\n\n", updated, len(entries))
	for i, e := range entries {
		fmt.Printf("%3d.  %-44s  ★%-7d  %s\n", i+1, e.ID, e.Stars, e.Name)
		if e.Description != "" {
			fmt.Printf("      %s\n", e.Description)
		}
		fmt.Printf("      github.com/%s/%s@%s · %s\n", e.Owner, e.Repo, e.Branch, e.Path)
		if len(e.Tags) > 0 {
			fmt.Printf("      tags: %s\n", strings.Join(e.Tags, ", "))
		}
		fmt.Println()
	}
}

// cleanupTempDirs：sync 结束后清理下载临时目录。
func cleanupTempDirs(results []funcs.RemoteFetchResult) {
	seen := map[string]bool{}
	for _, r := range results {
		parent := r.LocalDir
		for i := 0; i < 2 && parent != ""; i++ {
			parent = parentDir(parent)
		}
		if parent == "" || seen[parent] {
			continue
		}
		seen[parent] = true
		_ = os.RemoveAll(parent)
	}
}

func parentDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return ""
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
