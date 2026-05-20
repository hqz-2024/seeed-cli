// skills_index：解析 assets/skills-top50.yaml 内嵌索引，并提供搜索/查找辅助。
package funcs

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"seeed-cli/assets"
)

// IndexEntry：一条 skill 索引（指向某个 GitHub 仓库子目录下的 SKILL.md 文件夹）。
type IndexEntry struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Owner       string   `yaml:"owner"`
	Repo        string   `yaml:"repo"`
	Branch      string   `yaml:"branch"`
	Path        string   `yaml:"path"`
	Stars       int      `yaml:"stars"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
}

// Index：完整的索引清单。
type Index struct {
	Version int          `yaml:"version"`
	Updated string       `yaml:"updated"`
	Skills  []IndexEntry `yaml:"skills"`
}

// LoadEmbeddedIndex：解析编译期内嵌的 top50.yaml，并按 stars 降序排序。
func LoadEmbeddedIndex() (*Index, error) {
	var idx Index
	if err := yaml.Unmarshal(assets.TopYAML, &idx); err != nil {
		return nil, fmt.Errorf("parse embedded skills index: %w", err)
	}
	sort.SliceStable(idx.Skills, func(i, j int) bool {
		return idx.Skills[i].Stars > idx.Skills[j].Stars
	})
	return &idx, nil
}

// SearchIndex：按关键词在 id / name / description / tags 中过滤（大小写无关）。
func SearchIndex(entries []IndexEntry, q string) []IndexEntry {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return entries
	}
	var out []IndexEntry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.ID), q) ||
			strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Description), q) {
			out = append(out, e)
			continue
		}
		for _, t := range e.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// FindByTokens：按 token 命中 IndexEntry，支持 id / name / owner/name 三种写法。
// 返回命中条目与未命中 token 列表。
func FindByTokens(entries []IndexEntry, tokens []string) ([]IndexEntry, []string) {
	var out []IndexEntry
	var missing []string
	for _, raw := range tokens {
		tk := strings.TrimSpace(raw)
		if tk == "" {
			continue
		}
		hit := false
		for _, e := range entries {
			if e.ID == tk || e.Name == tk || (e.Owner+"/"+e.Name) == tk {
				out = append(out, e)
				hit = true
			}
		}
		if !hit {
			missing = append(missing, tk)
		}
	}
	return out, missing
}
