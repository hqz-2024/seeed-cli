---
name: seeed-cli-onboard-skill
description: >-
  新项目上手分析。在对陌生项目做任何代码修改前，先用 seeed-cli 完成架构分析、AI 协作文档生成、安全/质量扫描，确保有充分上下文再动手。触发：新项目、陌生代码库、项目分析、架构理解。
---

# /seeed-cli-onboard — 新项目分析后修改

新接触一个项目时，**禁止直接改代码**。必须先用 `seeed-cli` 完成分析，建立全局理解。

## 触发条件（按优先级判定）

1. **用户显式请求**：用户说"分析项目""先了解一下这个项目""新项目上手"等
2. **首次进入项目**：当前项目根目录不存在 `seeed-cli/` 目录，或 `seeed-cli/` 下无任何 `.md` 分析产物
3. **产物过期**：`seeed-cli/` 下最新 `.md` 文件的日期早于最近一次 git commit 的日期超过 7 天

## 工作流程

### Step 1: 检查 seeed-cli 是否就绪

```
seeed-cli -h
```

检查 API Key：

```
seeed-cli get-ak BaiLian && seeed-cli get-ak DeepSeek && seeed-cli get-ak GPT
```

至少一个 provider 有 key 即可。若全部为空，提示用户执行 `seeed-cli set-ak <provider> <key>` 后停止。

### Step 2: 架构分析 (必做)

```
SEEED_PLAIN=1 seeed-cli frame
```

等待产物 `seeed-cli/frame-*.md` 出现后读取。

### Step 3: AI 协作文档 (必做)

```
SEEED_PLAIN=1 seeed-cli gen-ai-agent
```

等待产物 `AI-AGENT.md` 更新后读取。

### Step 4: 安全/质量扫描 (按需)

若用户目标是修改代码（非纯调研），追加执行：

```
SEEED_PLAIN=1 seeed-cli safe-scan
SEEED_PLAIN=1 seeed-cli quality
```

等待 `seeed-cli/scan-safe-*.md` 和 `seeed-cli/scan-quality-*.md` 出现后读取。

### Step 5: 确定后再修改

读完所有产物后才能改代码，修改时遵守 `AI-AGENT.md` 约束。

---

## 执行策略

### 并行化

Step 2 + Step 3 可同时启动（互不依赖），Step 4 的两条命令也可同时启动。所有命令均用后台进程执行。

### 环境变量 SEEED_PLAIN=1

**所有 seeed-cli 命令必须加 `SEEED_PLAIN=1` 前缀**。

`seeed-cli` 默认启动 bubbletea TUI，在非交互式终端（agent 环境/CI）下 TUI 会卡死等待按键退出。设置 `SEEED_PLAIN=1` 强制跳过 TUI，直接执行工作逻辑并落盘 `.md` 文件。

| 命令 | 产物路径 | 等多久 |
|------|---------|--------|
| `SEEED_PLAIN=1 seeed-cli frame` | `seeed-cli/frame-YYYY-MM-DD_HH_MM_SS.md` | 1-3 分钟 |
| `SEEED_PLAIN=1 seeed-cli gen-ai-agent` | `./AI-AGENT.md` | 2-5 分钟 |
| `SEEED_PLAIN=1 seeed-cli safe-scan` | `seeed-cli/scan-safe-*.md` | 2-5 分钟 |
| `SEEED_PLAIN=1 seeed-cli quality` | `seeed-cli/scan-quality-*.md` | 2-5 分钟 |

### 产物完成的判定

**唯一依据是目标 `.md` 文件是否已出现**，不要解析终端输出（plain 模式下终端无输出）。

1. 用 `Glob` 检查目标产物是否在 `seeed-cli/` 目录下出现
2. 每 30-60 秒 Glob 一次，LLM 调用慢时需耐心等待，最多等 5 分钟
3. 一旦 Glob 返回目标文件，立即 `Read` 读取
4. 同类产物（如 frame-*.md）有多个时，读日期最新的

### 跳过的判定

若 Glob 发现产物已存在（同日生成），跳过对应命令，直接读已有文件。

### 读完后做的事

将核心发现写入项目记忆 (`memory/`)，更新 `MEMORY.md` 索引：
- `project_overview.md` — 技术栈 + 架构 + 目录结构
- `ai_collaboration_rules.md` — AI-AGENT.md 约束摘要
- `security_issues.md` — 安全扫描结果
- `quality_issues.md` — 质量评审结果

## 快速模式

只做调研不改代码时，执行 Step 1 + Step 2 + Step 3 即可（生成 AI-AGENT.md 对后续协作有价值）。

## 冷知识

- 所有分析产物在 `seeed-cli/` 目录下
- `frame` 报告可以在 IDE 中直接预览 Mermaid 图
- `gen-ai-agent` 生成的 `AI-AGENT.md` 会被 Claude Code 自动加载为上下文约束
- 可以用 `s` 简写代替 `seeed-cli`（需先 `seeed-cli install-s`）

## 参考

See `references/seeed-cli-commands.md` for full command reference.
