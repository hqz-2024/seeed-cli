// skills_remote：把 IndexEntry 指向的 GitHub 子目录递归拉到临时目录，并解析成内存中的 funcs.Skill。
package funcs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	githubAPIBase = "https://api.github.com"
	githubUA      = "seeed-cli-skills-pull"
)

// ghContent：GitHub Contents API 的最小返回模型。
type ghContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // file | dir | symlink
	DownloadURL string `json:"download_url"`
}

// RemoteFetchResult：一次远程拉取的产物。
type RemoteFetchResult struct {
	Entry     IndexEntry
	Skill     Skill  // Path 指向临时目录里的 SKILL.md，ExecuteSyncPlan 可直接复用
	LocalDir  string // 临时根目录（调用方应在 sync 完成后清理）
	FileCount int
}

// DownloadRemoteSkills：批量下载，单个失败不阻断整体；返回成功的结果与错误清单。
func DownloadRemoteSkills(entries []IndexEntry) ([]RemoteFetchResult, []error) {
	var out []RemoteFetchResult
	var errs []error
	for _, e := range entries {
		r, err := DownloadRemoteSkill(e)
		if err != nil {
			errs = append(errs, fmt.Errorf("[%s] %w", e.ID, err))
			continue
		}
		out = append(out, *r)
	}
	return out, errs
}

// DownloadRemoteSkill：把单个 IndexEntry 指向的远程目录下载到 ${TMP}/seeed-cli-pull-<ts>/<owner>__<repo>__<name>/，
// 并解析其中的 SKILL.md 返回 funcs.Skill。
func DownloadRemoteSkill(e IndexEntry) (*RemoteFetchResult, error) {
	tmpRoot, err := os.MkdirTemp("", "seeed-cli-pull-")
	if err != nil {
		return nil, fmt.Errorf("mkdtemp: %w", err)
	}
	safe := strings.NewReplacer("/", "__", " ", "_").Replace(e.Owner + "__" + e.Repo + "__" + e.Name)
	localDir := filepath.Join(tmpRoot, safe)
	if mkErr := os.MkdirAll(localDir, 0755); mkErr != nil {
		return nil, mkErr
	}

	n, err := fetchDirRecursive(e.Owner, e.Repo, e.Branch, e.Path, localDir)
	if err != nil {
		return nil, err
	}

	skillMD := filepath.Join(localDir, "SKILL.md")
	if _, statErr := os.Stat(skillMD); statErr != nil {
		return nil, fmt.Errorf("SKILL.md not found in %s/%s@%s/%s", e.Owner, e.Repo, e.Branch, e.Path)
	}
	sk, perr := ParseAssetFile(skillMD, tmpRoot, SkillSource("remote:"+e.Owner), AssetKindSkill)
	if perr != nil {
		return nil, perr
	}
	if sk.Name == "" || sk.Name == "SKILL" {
		sk.Name = e.Name
	}
	sk.SubPath = e.Name
	sk.RelPath = fmt.Sprintf("github.com/%s/%s@%s/%s/SKILL.md", e.Owner, e.Repo, e.Branch, strings.TrimPrefix(e.Path, "/"))
	return &RemoteFetchResult{Entry: e, Skill: *sk, LocalDir: localDir, FileCount: n}, nil
}

// fetchDirRecursive：用 GitHub Contents API 列出目录，把所有文件按相对结构写到 localDir。
func fetchDirRecursive(owner, repo, branch, subPath, localDir string) (int, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", githubAPIBase, owner, repo, url.PathEscape(strings.TrimPrefix(subPath, "/")), url.QueryEscape(branch))
	var items []ghContent
	if err := httpGetJSON(endpoint, &items); err != nil {
		return 0, err
	}
	count := 0
	for _, it := range items {
		relInside := strings.TrimPrefix(strings.TrimPrefix(it.Path, subPath), "/")
		target := filepath.Join(localDir, filepath.FromSlash(relInside))
		switch it.Type {
		case "dir":
			if mkErr := os.MkdirAll(target, 0755); mkErr != nil {
				return count, mkErr
			}
			nested := path.Join(subPath, it.Name)
			n, err := fetchDirRecursive(owner, repo, branch, nested, localDir)
			if err != nil {
				return count, err
			}
			count += n
		case "file":
			if mkErr := os.MkdirAll(filepath.Dir(target), 0755); mkErr != nil {
				return count, mkErr
			}
			if err := httpDownload(it.DownloadURL, target); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

// httpGetJSON：GET + JSON 反序列化；遇 GITHUB_TOKEN 自动附加 Authorization。
func httpGetJSON(u string, out interface{}) error {
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", githubUA)
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	cli := &http.Client{Timeout: 30 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("github api %s: %s | %s", u, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// httpDownload：把 URL 内容流式写入本地 target 文件。
func httpDownload(u, target string) error {
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("User-Agent", githubUA)
	cli := &http.Client{Timeout: 60 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("download %s: %s", u, resp.Status)
	}
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
