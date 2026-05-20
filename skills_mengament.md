# seeed-cli · Skills Management 开发文档

> 命令一：`skills-scan`（别名 `skills` / `sk`）—— **扫描 + 质量评分 + 缺口分析**，一次执行同步输出三段结果。
> 命令二：`skills-sync`（别名 `sync`）—— **跨工具 Skills 同步**，交互式选择源 skill 集合与目标工具，生成对应的 `SKILL.md` 文件包。
> 扫描范围：**只扫描 `.cursor/`、`.claude/`、`.windsurf/`、`.augment/` 四个工具配置根目录**，在每个根目录下分析三类资源：`skills/`（含 `SKILL.md` 的子文件夹）、`rules/`（`.md` / `.mdc` 文件）、`workflows/`（`.md` 文件）。

---

## 1. 功能目标

### 1.1 `skills-scan`（综合分析，单命令同步输出）

在用户当前项目目录下：

1. **探测**四个工具配置根目录（`.cursor` / `.claude` / `.windsurf` / `.augment`），缺失则跳过该工具；
2. **解析**每个工具下三类资源：
   - `skills/`：每个含 `SKILL.md` 的子文件夹算一个 skill 单元；
   - `rules/`：递归收集 `.md` / `.mdc` 文件；
   - `workflows/`：递归收集 `.md` 文件。
3. **归纳（基础视图）**：输出名称 / 来源工具 / 资源类别（skill/rule/workflow）/ 应用场景 / 触发条件 / 简要说明；
4. **质量评分**：对每条资源给出 4 维评分（描述清晰度、触发明确度、正文完备度、综合分，0–10 分），并标注主要扣分项；
5. **缺口分析**：结合项目技术栈（复用 `frame` 采集的清单文件 + 目录树），列出**项目里缺失但建议补充**的资源类目（如测试、安全扫描、部署、提交规范等），并给出建议名称与适用工具；
6. **展示**：复用现有 `repModel` 全屏 TUI，流式展示 LLM 输出；
7. **落盘**：生成 `seeed-cli/skills-YYYY-MM-DD_HH_mm_ss.md` 报告（含 4 个章节：总览 / 按工具分组 / 质量评分 / 缺口分析）。

### 1.2 `skills-sync`（跨工具同步）

1. **扫描**当前项目所有源 skill（复用 `ScanSkills`）；
2. **选择源 skill**：交互式多选 TUI（空格选中 / `a` 全选 / `Enter` 确认），或通过 `--skills name1,name2` 非交互指定；
3. **选择目标工具**：交互式单选 TUI，或通过 `--target cursor|claude|windsurf|augment` 指定；
4. **格式适配**：通过 `DialectAdapter` 将源 frontmatter 字段翻译到目标工具方言；
5. **写入**：在目标工具目录下创建对应 `<skill>/SKILL.md`；已存在则按 `--overwrite` / 默认跳过策略处理；
6. **报告**：控制台打印同步清单，并落盘 `seeed-cli/skills-sync-YYYY-MM-DD_HH_mm_ss.md`。

---

## 2. 扫描策略

### 2.1 扫描方式

- 入口：`projectRoot = os.Getwd()`；
- 路径范围：**只看四个工具根目录** —— `.cursor/`、`.claude/`、`.windsurf/`、`.augment/`；任一根目录不存在则跳过该工具；
- 在每个工具根目录下分别处理三类资源（任一子目录不存在则跳过）：
  - `{tool}/skills/`：调用 `os.ReadDir` 取一级子文件夹，若该文件夹内含**严格大小写**的 `SKILL.md` 则识别为一个 skill 单元（resource kind = `skill`）；
  - `{tool}/rules/`：`filepath.Walk` 递归收集 `.md` / `.mdc` 文件，每个文件即一条 rule（resource kind = `rule`）；
  - `{tool}/workflows/`：`filepath.Walk` 递归收集 `.md` 文件，每个文件即一条 workflow（resource kind = `workflow`）。
- 跳过条件：单文件 size > `funcs.MaxFileSize`（1 MiB）记入 `SkippedNo`；
- 单文件解析失败仅记日志，**不影响其他文件**。

### 2.2 Source 来源映射

来源由所在工具根目录决定，**无需路径推断**：

| 工具根目录 | 归属 source |
|---|---|
| `.cursor/`   | `cursor`   |
| `.claude/`   | `claude`   |
| `.windsurf/` | `windsurf` |
| `.augment/`  | `augment`  |

> `InferSourceFromPath` 仍保留为兜底工具函数（如旧报告路径回填使用），但 `ScanSkills` 主流程不再依赖路径推断。

### 2.3 名称解析

- 通用规则：优先取 `frontmatter.name`；
- 兜底：
  - `skill` 资源 → `SKILL.md` 所在的子文件夹名（与 Cursor / Claude 官方约定一致）；
  - `rule` / `workflow` 资源 → 文件名去扩展名（`.md` / `.mdc`）。

### 2.4 空结果处理

四个工具下三类资源加总为 0 → 控制台输出 `no skills/rules/workflows detected under .cursor/.claude/.windsurf/.augment`，**不调用 LLM**，正常退出。

---

## 3. 资源文件格式约定

三类资源使用同一套**轻量级 frontmatter + Markdown 正文**约定，解析器不区分 kind，差异在「兜底名称」与「目标目录」。

```markdown
---
name: deploy-staging
description: 部署到 staging 环境前的检查与发布
when: 用户提及 deploy / staging / 发布 时触发
---

正文为该资源的具体指令、流程、示例……
```

Frontmatter 字段（容错处理，缺失则用 LLM 推断）：

| 字段 | 必需 | 用途 |
|---|---|---|
| `name`        | 否 | 显式名；skill 缺失时取文件夹名，rule/workflow 缺失时取文件名（去扩展名） |
| `description` | 否 | 一句话说明 → 映射为 Scenario |
| `when` / `trigger` / `paths` / `globs` | 否 | 触发条件 → 映射为 Trigger |
| 其他          | 否 | 透传到原始字段保留 |

---

## 4. 模块与文件划分

新增以下文件，沿用现有包约定：

```
commands/
├── skills.go              # skills-scan：Handler、Prompt、Worker、报告落盘
├── skills_sync.go         # skills-sync：Handler、交互选择、写盘逻辑
├── skills_select_ui.go    # skills-sync 专用 bubbletea 多选/单选 UI
└── funcs/
    ├── skills_scan.go     # 路径探测、SKILL.md 解析、Skill 结构定义
    └── skills_dialect.go  # 各工具 frontmatter 方言适配器
```

辅助改造（小范围）：
- `commands/frame.go` 中的 `collectFrameCorpus` 改为**导出版本** `CollectFrameCorpus`，供 `skills-scan` 的缺口分析复用项目技术栈采集逻辑；保留原 `runFrameWork` 内部仍调本函数，不破坏既有行为。

注册入口：在 `main.go` 的 `_commands` 切片中追加两项（`skills-scan` 与 `skills-sync`，紧邻 `who` 命令位置）。

---

## 5. 数据结构（`commands/funcs/skills_scan.go`）

```go
// SkillSource 标识资源来源工具。
type SkillSource string

const (
    SourceCursor   SkillSource = "cursor"
    SourceClaude   SkillSource = "claude"
    SourceWindsurf SkillSource = "windsurf"
    SourceAugment  SkillSource = "augment"
    SourceGeneric  SkillSource = "generic" // 仅供 InferSourceFromPath 兜底使用
)

// AssetKind 资源类别。
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

// SkillTargetDirMap：已知工具 → skills 写盘根目录（相对项目根）。
// 仅供 skills-sync 写盘时使用。
var SkillTargetDirMap = map[SkillSource]string{
    SourceCursor:   ".cursor/skills",
    SourceClaude:   ".claude/skills",
    SourceWindsurf: ".windsurf/skills",
    SourceAugment:  ".augment/skills",
}

// SourcePathMarkers：路径子串 → source。InferSourceFromPath 的兜底匹配表。
var SourcePathMarkers = map[string]SkillSource{
    ".cursor/":   SourceCursor,
    ".claude/":   SourceClaude,
    ".windsurf/": SourceWindsurf,
    ".augment/":  SourceAugment,
}

// Skill 表示一个被发现并解析后的资源（skill / rule / workflow 通用）。
type Skill struct {
    Source      SkillSource       // 来源工具
    Kind        AssetKind         // 资源类别
    Name        string            // 名称（frontmatter.name 或 文件夹/文件名）
    Path        string            // 文件绝对路径
    RelPath     string            // 相对项目根的路径
    Description string            // frontmatter.description
    Trigger     string            // frontmatter.when / trigger / paths（合并）
    Frontmatter map[string]string // 原始 frontmatter
    Body        string            // 正文（截断后）
}

// SkillScanResult 一次扫描的聚合结果。
type SkillScanResult struct {
    Root      string        // 项目根
    Skills    []Skill       // 全部解析成功的资源（含 skill/rule/workflow）
    Detected  []SkillSource // 检测到至少存在目录的工具
    SkippedNo []string      // 跳过的路径（超大或解析失败）
}

// 辅助方法
// func (r *SkillScanResult) FilterByKind(kind AssetKind) []Skill
// func (r *SkillScanResult) CountByKind(kind AssetKind) int

// SkillQualityScore 单个 skill 的质量评分（0–10 整数）。
type SkillQualityScore struct {
    SkillRef     string   // "<source>/<name>"
    Description  int      // 描述清晰度
    Trigger      int      // 触发明确度
    Body         int      // 正文完备度（是否含示例/步骤）
    Overall      int      // 综合分
    Issues       []string // 主要扣分原因（LLM 给出）
}

// SkillGapSuggestion 缺口分析中的一条建议。
type SkillGapSuggestion struct {
    Category    string        // 缺失类目（如 testing / security / commit）
    Reason      string        // 为什么项目应当补充
    SkillName   string        // 建议的 skill 名（短横线命名）
    SuggestFor  []SkillSource // 建议落地到哪些工具
}

// SkillAnalysisReport 一次 skills-scan 的完整产物。
type SkillAnalysisReport struct {
    Scan     *SkillScanResult
    Scores   []SkillQualityScore
    Gaps     []SkillGapSuggestion
    Raw      string            // LLM 原始 Markdown 输出
}

// SkillSyncPlan 一次 skills-sync 的执行计划与结果。
type SkillSyncPlan struct {
    Sources    []Skill        // 用户选中的源 skill
    Target     SkillSource    // 目标工具
    TargetRoot string         // 目标目录绝对路径（如 <pwd>/.cursor/skills）
    Overwrite  bool           // 文件已存在时是否覆盖
    Written    []string       // 实际写入的相对路径
    Skipped    []string       // 跳过的（已存在且未指定 overwrite）
    Failed     []string       // 写入失败的（含原因）
}
```

---

## 6. 公开 API（函数签名）

### 6.1 `commands/funcs/skills_scan.go`

```go
// ScanSkills 自 projectRoot 扫描 .cursor/.claude/.windsurf/.augment 四个工具根，
// 在每个工具根下处理 skills/、rules/、workflows/ 三类资源。
// - skills/：os.ReadDir 取一级子目录，含 SKILL.md（严格大小写）则识别为一个 skill 单元；
// - rules/：filepath.Walk 递归收集 .md / .mdc 文件；
// - workflows/：filepath.Walk 递归收集 .md 文件；
// - 超过 funcs.MaxFileSize 的文件跳过并记入 SkippedNo；
// - 单文件解析失败仅记日志，不影响其他文件；
// - 顺带填充 Detected：实际出现过资源的 SkillSource 去重集合。
func ScanSkills(projectRoot string) (*SkillScanResult, error)

// collectSkillFolders / collectMarkdownFiles：ScanSkills 内部使用的两类收集器，
// 分别对应「skills 文件夹单元」与「rules/workflows 文件单元」。
func collectSkillFolders(skillsDir, projectRoot string, source SkillSource, res *SkillScanResult, seen map[SkillSource]struct{})
func collectMarkdownFiles(root, projectRoot string, source SkillSource, kind AssetKind, res *SkillScanResult, seen map[SkillSource]struct{})

// FilterByKind / CountByKind：扫描结果的按 kind 过滤与计数辅助。
func (r *SkillScanResult) FilterByKind(kind AssetKind) []Skill
func (r *SkillScanResult) CountByKind(kind AssetKind) int

// ParseAssetFile 解析单个资源文件（skill / rule / workflow 通用）。
// kind = AssetKindSkill 时 defaultName 取所在文件夹名；其余 kind 取文件名（去扩展名）。
func ParseAssetFile(absPath, projectRoot string, source SkillSource, kind AssetKind) (*Skill, error)

// ParseSkillFile 保留为兼容入口，等价于 ParseAssetFile(..., AssetKindSkill)。
func ParseSkillFile(absPath, projectRoot string, source SkillSource) (*Skill, error)

// InferSourceFromPath 兜底：按 SourcePathMarkers 子串匹配，无命中 → SourceGeneric。
// ScanSkills 主流程不依赖此函数，仅外部调用时使用。
func InferSourceFromPath(absPath string) SkillSource

// parseFrontmatter 提取顶部 YAML frontmatter（轻量级 key: value）。
func parseFrontmatter(raw string) (fm map[string]string, body string)

// resolveSkillName / resolveTrigger 内部辅助函数。
func resolveSkillName(defaultName string, fm map[string]string) string
func resolveTrigger(fm map[string]string) string
```

> `skills-sync` 仅消费 `FilterByKind(AssetKindSkill)` 的结果；rule / workflow 不会被同步（同步语义未定义且会破坏目标工具加载规则）。

### 6.2 `commands/funcs/skills_dialect.go`（跨工具方言适配）

```go
// DialectAdapter 描述某个目标工具的 frontmatter 翻译策略。
type DialectAdapter interface {
    Source() SkillSource
    // Translate 接收源 skill，返回适配后的 frontmatter 与正文。
    // 例如：Cursor 保留 paths；Claude 移除 paths、保留 allowed-tools；
    // 未识别字段一律透传。
    Translate(src Skill) (fm map[string]string, body string)
    // SkillDirRel 返回该工具的 skills 根目录（相对项目根）。
    SkillDirRel() string
}

// GetAdapter 按目标 source 返回对应适配器。
func GetAdapter(target SkillSource) (DialectAdapter, error)

// RenderSkillFile 将 frontmatter + body 序列化回 SKILL.md 文本。
func RenderSkillFile(fm map[string]string, body string) string
```

### 6.3 `commands/skills.go`（skills-scan）

```go
// HandleSkills CLI 入口，复用 runRepUI 全屏 TUI。
func HandleSkills(ctx context.Context, cmd *cli.Command) error

// runSkillsWork 后台：扫描 → 采集项目技术栈 → LLM 综合分析 → 落盘。
func runSkillsWork(m *repModel)

// buildSkillsCorpus 拼接 skill 列表 + 项目技术栈摘要（来自 CollectFrameCorpus）。
// 单 skill 正文最多 8KB；技术栈摘要最多 32KB；总体上限 200KB。
func buildSkillsCorpus(scan *funcs.SkillScanResult, frameCorpus string) string

// parseAnalysisOutput 从 LLM Markdown 输出中提取结构化的 Scores / Gaps，
// 解析失败不影响 Raw 字段的展示。
func parseAnalysisOutput(raw string) (scores []funcs.SkillQualityScore, gaps []funcs.SkillGapSuggestion)

// renderSkillsMarkdown 合并扫描元数据 + LLM 输出为最终报告。
func renderSkillsMarkdown(rep *funcs.SkillAnalysisReport) string
```

### 6.4 `commands/skills_sync.go`（skills-sync）

```go
// HandleSkillsSync CLI 入口，支持 --target / --skills / --overwrite / --yes 四个 flag。
// 缺少 flag 时进入交互式选择 UI。
func HandleSkillsSync(ctx context.Context, cmd *cli.Command) error

// runSkillsSyncWork 执行写盘并打印结果。
func runSkillsSyncWork(plan *funcs.SkillSyncPlan) error

// SelectSourceSkillsInteractive 多选 TUI：返回用户勾选的源 skill 切片。
func SelectSourceSkillsInteractive(skills []funcs.Skill) ([]funcs.Skill, error)

// SelectTargetInteractive 单选 TUI：返回目标工具。
func SelectTargetInteractive() (funcs.SkillSource, error)

// BuildSyncPlan 根据用户选择构造同步计划（仅计算，不写盘）。
func BuildSyncPlan(projectRoot string, sources []funcs.Skill, target funcs.SkillSource, overwrite bool) (*funcs.SkillSyncPlan, error)

// ExecuteSyncPlan 真正写入磁盘，按 plan 顺序处理 Written / Skipped / Failed。
func ExecuteSyncPlan(plan *funcs.SkillSyncPlan) error

// renderSyncReport 把同步结果落盘为 seeed-cli/skills-sync-时间戳.md。
func renderSyncReport(plan *funcs.SkillSyncPlan) string
```

### 6.5 `commands/frame.go`（小改造）

```go
// CollectFrameCorpus 由原 collectFrameCorpus 改为导出，供 skills-scan 缺口分析复用。
func CollectFrameCorpus(root string) string
```

---

## 7. LLM Prompt（`commands/skills.go` 顶部常量）

```text
const skillsPrompt = `你是 AI 编程工具配置审计助手。下面会给你两段输入：
（A）若干 AI 配置资源条目（每条标注 source、kind ∈ {skill,rule,workflow}、name、相对路径、frontmatter、正文片段）。
    - skill：来自 .cursor/.claude/.windsurf/.augment 下 skills/ 子目录中含 SKILL.md 的文件夹；
    - rule：来自上述四工具的 rules/ 目录（.md / .mdc 文件）；
    - workflow：来自上述四工具的 workflows/ 目录（.md 文件）。
（B）项目技术栈摘要（清单文件 + 目录树片段）。

请输出一份 Markdown 报告，**严格按以下四章顺序与标题**：

## 1. 总览
- 表格列出：来源工具 | 类别 | 名称 | 应用场景（一句话） | 触发条件（一句话）

## 2. 按工具分组
对每个出现过的来源工具（Cursor/Claude/Windsurf/Augment）输出一个二级小节，
小节内按 **skills / rules / workflows** 三个三级小节分别罗列；每条资源给出：
- **应用场景**：基于 description 与正文归纳，禁止编造；
- **触发条件**：基于 when / trigger / paths / globs / alwaysApply 等字段；缺失则写「未声明，建议补充」；
- **风险/重叠提示**：多个资源描述高度相似时标注「与 XXX 可能重叠」。

## 3. 质量评分
- 表格列出：资源引用（source/kind/name）| 描述清晰度 | 触发明确度 | 正文完备度 | 综合分 | 主要扣分项
- 评分维度均为 0–10 整数；综合分 = 三项均值四舍五入；
- 表格之后，挑选综合分 < 6 的资源，逐条给出**修复建议**（每条 ≤ 2 句）。

## 4. 缺口分析
- 基于（B）推断项目类型（语言、框架、构建/部署方式）。
- 对照（A）已有资源类目，列出**项目应当具备但目前缺失**的 skill / rule / workflow 类目；
- 表格列出：缺失类目 | 类别 | 缺失原因（一句话）| 建议名称 | 建议落地工具；
- 严禁臆测项目不需要的方向；技术栈摘要中无明显证据时写「无明显缺口」即可。

要求：
- 不修改原文、不新增不存在的资源；
- 全文中文，标准 Markdown；
- 第 3、4 章的表格列与字段名必须**逐字一致**，便于后续解析。`
```

模型沿用 `cfg.Provider.BaiLian.Model`，与 `who` / `frame` 一致；调用方式 `m.RunLLMStream(...)`。

> `parseAnalysisOutput` 仅做尽力解析：按章节标题与表头匹配抽取行，解析失败时返回空切片，Raw 字段保证最终报告完整可读。

---

## 8. 输出与落盘

### 8.1 `skills-scan` 报告

- 控制台：流式渲染 LLM 输出至 viewport；
- 报告路径：`<pwd>/seeed-cli/skills-2006-01-02_15_04_05.md`（与 `frame` 一致的目录与命名）；
- 报告结构（由 `renderSkillsMarkdown` 拼接）：
  1. 扫描元信息表（项目根、扫描范围、按 kind 分项的资源数、来源工具数、扫描时间）；
  2. 已发现的资源清单（按 skill / rule / workflow 三段分别列出 #/来源/名称/相对路径）；
  3. LLM 输出（4 章节正文）；
  4. 附录：结构化后的 `SkillQualityScore` / `SkillGapSuggestion`（JSON 代码块，便于后续工具消费）。

### 8.2 `skills-sync` 报告

- 控制台：实时打印 `Written` / `Skipped` / `Failed` 三段；
- 报告路径：`<pwd>/seeed-cli/skills-sync-2006-01-02_15_04_05.md`；
- 内容：源→目标对应表、frontmatter 字段映射差异（哪些字段被剔除/重命名）、完整文件清单。

---

## 9. CLI 注册（修改 `main.go`）

在 `_commands` 切片内追加两项：

```go
{
    Name:    "skills-scan",
    Usage:   "扫描并分析 AI 工具 skills：场景/触发条件 + 质量评分 + 缺口分析",
    Aliases: []string{"skills", "sk"},
    Action:  commands.HandleSkills,
},
{
    Name:    "skills-sync",
    Usage:   "跨工具同步 skills：交互选择源 skill 与目标工具，生成 SKILL.md 包",
    Aliases: []string{"sync"},
    Flags: []cli.Flag{
        &cli.StringFlag{Name: "target", Usage: "目标工具：cursor|claude|windsurf|augment"},
        &cli.StringFlag{Name: "skills", Usage: "逗号分隔的源 skill 名（缺省进入交互式多选）"},
        &cli.BoolFlag{Name: "overwrite", Usage: "目标文件已存在时是否覆盖"},
        &cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "跳过确认提示"},
    },
    Action: commands.HandleSkillsSync,
},
```

---

## 10. 跨工具 Skills 同步（`skills-sync` 专章）

### 10.1 交互流程

```
用户执行 seeed-cli skills-sync
        │
        ├─ ScanSkills(pwd) → 得到所有源 skill
        │
        ├─ 若 --skills 缺省 → SelectSourceSkillsInteractive（多选 TUI）
        │   - 列表：[源工具] skill 名  ·  应用场景一句话
        │   - 操作键：↑/↓ 移动、Space 选/取消、a 全选、Enter 确认、q 取消
        │
        ├─ 若 --target 缺省 → SelectTargetInteractive（单选 TUI）
        │   - 四个固定选项：Cursor / Claude / Windsurf / Augment
        │   - 已是源工具的选项标灰但仍可选（同工具内 copy/rename 场景）
        │
        ├─ BuildSyncPlan → 计算每个源 skill 的目标路径：
        │   <pwd>/<TargetSkillDir>/<skill name>/SKILL.md
        │
        ├─ 若非 --yes → 控制台打印计划摘要并等待 [Y/n]
        │
        └─ ExecuteSyncPlan → 写文件，分类记入 Written/Skipped/Failed
```

### 10.2 方言适配规则（`skills_dialect.go`）

所有四款工具的 `SKILL.md` 文件结构本身一致（folder + SKILL.md + YAML frontmatter + body），差异仅在 frontmatter 字段名约定上。`DialectAdapter` 默认采用**字段白名单 + 透传**策略：

| 字段 | Cursor | Claude | Windsurf | Augment | 说明 |
|---|:-:|:-:|:-:|:-:|---|
| `name`         | ✓ | ✓ | ✓ | ✓ | 必填（缺失时由 folder 名补齐） |
| `description`  | ✓ | ✓ | ✓ | ✓ | 必填 |
| `paths`        | ✓ | – | – | – | Cursor 独有，目标若非 Cursor 则**剔除**并改写到正文「适用范围」段 |
| `allowed-tools`| – | ✓ | – | – | Claude 独有，目标若非 Claude 则**剔除**并改写到正文「工具约束」段 |
| `when` / `trigger` | ✓ | ✓ | ✓ | ✓ | 通用，原样透传 |
| 未识别字段     | ✓ | ✓ | ✓ | ✓ | 透传，避免信息丢失 |

被剔除字段会在生成文件的**正文顶部**以「> _来自 cursor 的 `paths` 字段：xxx_」形式追加为引用块注释，确保信息可追溯。

### 10.3 写盘策略

- 目标目录不存在 → `os.MkdirAll(... , 0755)`；
- 目标 `SKILL.md` 已存在：
  - `--overwrite=true` → 覆盖，记入 `Written`；
  - 否则 → 跳过，记入 `Skipped`；
- 写入失败（权限/磁盘等）→ 记入 `Failed`，**不影响后续 skill 处理**；
- 同源同目标（如把 Cursor 的 skill 同步到 Cursor 自己）→ 视作 rename/clone 场景，目标 folder 名允许与源一致；若一致且未 overwrite 则跳过。

### 10.4 非交互模式示例

```bash
seeed-cli skills-sync --target claude --skills deploy-staging,api-review --overwrite -y
```

---

## 11. 实现步骤（建议顺序）

1. **`commands/funcs/skills_scan.go`**：常量、结构体（含 `SourceGeneric` / `SkillTargetDirMap` / `SourcePathMarkers`）、`ScanSkills`（全工程 `filepath.Walk`，不应用 `IgnoreDir`）/ `InferSourceFromPath` / `ParseSkillFile` / `parseFrontmatter` / `resolveSkillName` / `resolveTrigger`。
2. **`commands/funcs/skills_dialect.go`**：四个 `DialectAdapter` 实现、`GetAdapter`、`RenderSkillFile`（frontmatter + body 序列化回 SKILL.md）。
3. **`commands/frame.go` 改造**：把 `collectFrameCorpus` 改名为导出 `CollectFrameCorpus`，原 `runFrameWork` 内部调用同步更新。
4. **`commands/skills.go`**：Prompt 常量、`HandleSkills`、`runSkillsWork`、`buildSkillsCorpus`（拼 skill 内容 + 项目技术栈摘要）、`parseAnalysisOutput`（解析 Scores/Gaps）、`renderSkillsMarkdown`。
5. **`commands/skills_select_ui.go`**：bubbletea 多选 + 单选 UI，参考 `commands/rep_ui.go` 的消息/Model 结构。
6. **`commands/skills_sync.go`**：`HandleSkillsSync` 解析 flag → 决定交互或非交互；`BuildSyncPlan` / `ExecuteSyncPlan` / `renderSyncReport`。`--target` 仅接受 `SkillTargetDirMap` 中的 4 个已知工具。
7. **`main.go`**：注册 `skills-scan` 与 `skills-sync` 两条命令（含 sync 的四个 flag）。
8. **空结果与异常路径覆盖**：`skills-scan` 全工程零 `SKILL.md` → 直接退出；`skills-sync` 源 skill 为 0 → 报错退出；`--target` 取值不合法（含 `generic`）→ 提前校验报错。
9. **`funcs/util.go` 复用**：`MaxFileSize` 用于跳过异常超大 SKILL.md；本两条命令均**不复用** `IgnoreDir`，亦不修改 `CodeExt`。

---

## 12. 测试与验收

### 12.1 `skills-scan`

- **空仓库**：全工程零 `SKILL.md` → 控制台输出 `no skills detected`，无 LLM 调用，退出码 0。
- **单工具**：仅 `.cursor/skills/foo/SKILL.md` 存在 → 报告含 1 个 skill；`Detected = [cursor]`。
- **多工具混合**：四个目录各放 1 个 SKILL.md → 报告分四个二级小节，总览表 4 行。
- **Generic 来源**：在 `docs/team/SKILL.md` 放一个文件 → 被识别为 `SourceGeneric`，能正常进入报告并参与质量评分。
- **路径推断单测**：对 `InferSourceFromPath` 输入 `.cursor/skills/x/SKILL.md`、`.claude/skills/y/SKILL.md`、`docs/z/SKILL.md`、`a/.cursor/skills/x/SKILL.md`（嵌套也命中）等用例，断言返回值正确，且对 Windows 反斜杠路径同样成立。
- **大小写敏感**：在仓库放置 `skill.md`、`Skill.md` 不应被采集；只有 `SKILL.md` 被采集。
- **frontmatter 缺失**：仅有正文 → 评分中 `Description` / `Trigger` 自动得低分，且 `Issues` 含「未声明 description / 触发条件」。
- **质量评分解析**：构造一份固定 LLM 输出文本喂入 `parseAnalysisOutput`，断言 `Scores` 长度与字段值。
- **缺口分析有效性**：构造一个仅含 `.cursor/skills/lint/SKILL.md` 的 Go 项目（含 `go.mod`），期望 LLM 输出至少给出 `testing` 或 `commit` 类目缺口（端到端测试，允许人工确认）。
- **超大文件 / 非 SKILL.md**：跳过、忽略行为同前。
- **海量目录性能**：包含 `node_modules/` / `.git/` 的真实仓库 → 全工程扫描应能在可接受时间（≤ 10s 量级）完成，不报错。

### 12.2 `skills-sync`

- **非交互全参数**：`--target claude --skills a,b --yes` → 检查目标目录下生成 `a/SKILL.md`、`b/SKILL.md`，frontmatter `paths` 字段被剔除并下沉到正文。
- **交互选择**：模拟 stdin 按键序列 `↓ Space ↓ Space Enter` → `SelectSourceSkillsInteractive` 返回 2 条选中项（可通过对 UI Model 单测，绕过真实 TTY）。
- **冲突跳过**：目标文件已存在且 `--overwrite=false` → `Skipped` 含该路径，源文件不动。
- **冲突覆盖**：`--overwrite=true` → 目标文件内容更新，`Written` 记录。
- **失败容错**：把目标目录设为只读 → `Failed` 含错误描述，进程退出码 0（与 `who` / `frame` 的容错策略一致），报告写入正常。
- **同源同目标**：`--target cursor` 同步 Cursor 自己的 skill → 行为合法，遵循覆盖策略。

### 12.3 单元测试覆盖目标

`commands/funcs/skills_scan_test.go`：`parseFrontmatter`、`resolveSkillName`、`resolveTrigger`、`ScanSkills`（用 `t.TempDir()` 造目录树）。
`commands/funcs/skills_dialect_test.go`：四个 adapter 的 `Translate` 字段剔除/透传行为、`RenderSkillFile` 往返一致性。
`commands/skills_sync_test.go`：`BuildSyncPlan` 路径计算、`ExecuteSyncPlan` 在临时目录下的 Written/Skipped/Failed 分类。
