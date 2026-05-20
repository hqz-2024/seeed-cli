// skills_dialect：跨工具 frontmatter 方言适配 —— 字段白名单 + 透传 + 剔除字段下沉到正文。
package funcs

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// DialectAdapter：某个目标工具的 frontmatter 翻译策略。
type DialectAdapter interface {
	Source() SkillSource
	Translate(src Skill) (fm map[string]string, body string)
	SkillDirRel() string
}

// 工具特有字段集合：目标若非此工具，剔除并下沉到正文。
var toolSpecificFields = map[SkillSource]map[string]struct{}{
	SourceCursor:   {"paths": {}, "globs": {}},
	SourceClaude:   {"allowed-tools": {}, "allowed_tools": {}},
	SourceWindsurf: {},
	SourceAugment:  {},
}

// commonFields：通用字段（任何目标都保留）。
var commonFields = map[string]struct{}{
	"name":        {},
	"description": {},
	"when":        {},
	"trigger":     {},
}

type baseAdapter struct {
	target SkillSource
}

func (a baseAdapter) Source() SkillSource { return a.target }

func (a baseAdapter) SkillDirRel() string { return SkillTargetDirMap[a.target] }

// Translate：按 target 规则裁剪 frontmatter；被剔除字段以引用块形式追加到正文顶部。
func (a baseAdapter) Translate(src Skill) (map[string]string, string) {
	out := map[string]string{}
	var dropped []string

	for k, v := range src.Frontmatter {
		if _, isCommon := commonFields[k]; isCommon {
			out[k] = v
			continue
		}
		if owner, owned := ownerOfField(k); owned {
			if owner == a.target {
				out[k] = v
			} else {
				dropped = append(dropped, fmt.Sprintf("> _来自 %s 的 `%s` 字段：%s_", owner, k, v))
			}
			continue
		}
		out[k] = v
	}

	if _, ok := out["name"]; !ok && src.Name != "" {
		out["name"] = src.Name
	}
	if _, ok := out["description"]; !ok && src.Description != "" {
		out["description"] = src.Description
	}

	body := src.Body
	if len(dropped) > 0 {
		sort.Strings(dropped)
		body = strings.Join(dropped, "\n") + "\n\n" + body
	}
	return out, body
}

// ownerOfField：判断 key 属于哪个工具的专属字段；不命中则非专属。
func ownerOfField(key string) (SkillSource, bool) {
	for src, set := range toolSpecificFields {
		if _, ok := set[key]; ok {
			return src, true
		}
	}
	return "", false
}

// GetAdapter：按目标 source 返回对应适配器（仅四个已知工具）。
func GetAdapter(target SkillSource) (DialectAdapter, error) {
	if _, ok := SkillTargetDirMap[target]; !ok {
		return nil, fmt.Errorf("unsupported target: %s", target)
	}
	return baseAdapter{target: target}, nil
}

// PreferredRuleExt：目标工具的 rule 文件首选扩展名（Cursor 用 .mdc，其余 .md）。
func PreferredRuleExt(target SkillSource) string {
	if target == SourceCursor {
		return ".mdc"
	}
	return ".md"
}

// TargetAssetRelPath：基于资源 kind 与 SubPath，计算落到目标工具下的相对路径（项目根相对）。
//   - skill   → {tool}/skills/<name>/SKILL.md
//   - rule    → {tool}/rules/<subpath 调整扩展名>
//   - workflow→ {tool}/workflows/<subpath 扩展名归一为 .md>
func TargetAssetRelPath(asset Skill, target SkillSource) string {
	toolRoot, ok := SourceRootDirs[target]
	if !ok {
		return ""
	}
	switch asset.Kind {
	case AssetKindSkill:
		name := asset.SubPath
		if name == "" {
			name = asset.Name
		}
		return filepath.ToSlash(filepath.Join(toolRoot, AssetKindDirs[AssetKindSkill], name, "SKILL.md"))
	case AssetKindRule:
		sub := asset.SubPath
		if sub == "" {
			sub = asset.Name + PreferredRuleExt(target)
		} else {
			sub = stripExt(sub) + PreferredRuleExt(target)
		}
		return filepath.ToSlash(filepath.Join(toolRoot, AssetKindDirs[AssetKindRule], sub))
	case AssetKindWorkflow:
		sub := asset.SubPath
		if sub == "" {
			sub = asset.Name + ".md"
		} else {
			sub = stripExt(sub) + ".md"
		}
		return filepath.ToSlash(filepath.Join(toolRoot, AssetKindDirs[AssetKindWorkflow], sub))
	}
	return ""
}

// stripExt：去掉路径末段的扩展名（仅一层）。
func stripExt(p string) string {
	ext := filepath.Ext(p)
	if ext == "" {
		return p
	}
	return strings.TrimSuffix(p, ext)
}

// RenderSkillFile：把 frontmatter + body 序列化回 SKILL.md。
func RenderSkillFile(fm map[string]string, body string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	preferred := []string{"name", "description", "when", "trigger", "paths", "allowed-tools"}
	written := map[string]bool{}
	for _, k := range preferred {
		if v, ok := fm[k]; ok {
			sb.WriteString(k)
			sb.WriteString(": ")
			sb.WriteString(escapeFMValue(v))
			sb.WriteString("\n")
			written[k] = true
		}
	}
	for _, k := range keys {
		if written[k] {
			continue
		}
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(escapeFMValue(fm[k]))
		sb.WriteString("\n")
	}
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimLeft(body, "\n"))
	if !strings.HasSuffix(sb.String(), "\n") {
		sb.WriteString("\n")
	}
	return sb.String()
}

// escapeFMValue：若值包含冒号 / # 等敏感字符，加双引号包裹。
func escapeFMValue(v string) string {
	if v == "" {
		return "\"\""
	}
	if strings.ContainsAny(v, ":#\n\"") {
		v = strings.ReplaceAll(v, "\"", "\\\"")
		return "\"" + v + "\""
	}
	return v
}
