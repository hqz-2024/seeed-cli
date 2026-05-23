# 程序员的 AI 小伙伴 - seeed-cli

![help](./imgs/help.png) 

## 是什么

`seeed-cli` 在项目里统一调度 AI：把任务写成命令，复用同一套上下文，少做「每次从头讲一遍」的重复沟通。

安全扫描、质量检查、架构梳理、面向 AI 的说明、日报与提交说明、接口文档等，尽量一条链路走完。


> 基于AI的CLI工具: seeed-cli ，围绕项目开发时常见场景提供免自行驱动AI方案

## 功能/命令

| #   | 功能                                                  | 命令              | 完成 |
| --- | ----------------------------------------------------- | ----------------- | ---- |
| 1   | 项目安全扫描                                          | `s safe-scan`    | ✓    |
| 2   | 项目代码质量检查                                      | `s quality`      | ✓    |
| 3   | 项目架构分析                                          | `s frame`        | ✓    |
| 4   | 项目说明生成(针对AI)，作为 AI 修改本项目的说明书      | `s gen-ai-agent` | ✓    |
| 5   | 根据提交历史生成日报                                  | `s gen-daily`    | ✓    |
| 5   | 根据修改文件生成提交的 commit 说明(git commit 的描述) | `s gen-commit`   | ✓    |
| 6   | 接口文档生成                                          | `s gen-api-doc`  | ✓    |
| 7   | 代码出处分析                                          | `s who`          | ✓    |
| 8   | 多端兼容(windows、linux、macos)                       | —                 | ✓    |
| 9   | AI skills/rules/workflows 扫描 + 评分 + 缺口分析      | `s skills-scan`  | ✓    |
| 10  | 跨工具 skills 同步（生成对应 SKILL.md 文件包）        | `s skills-sync`  | ✓    |
| 11  | 联网拉取高星 skill（内嵌 Top-50 索引）                | `s skills-pull`  | ✓    |
| 12  | 炫酷 ai 聊天                                          | —                 | x    |
| 13  | 想象中...                                             | —                 | -    |


## AI Skills 管理

围绕 Cursor / Claude Code / Windsurf / Augment 这类 vibe coding 工具的 `SKILL.md` 协议，提供三条命令：

### skills-scan：扫描 + 评分 + 缺口分析

只扫描 `.cursor/`、`.claude/`、`.windsurf/`、`.augment/` 四个工具根目录下的三类资源：

- **skills**：`{tool}/skills/<name>/SKILL.md` —— 每个含 `SKILL.md` 的子文件夹算一个 skill 单元；
- **rules**：`{tool}/rules/**/*.md` 或 `*.mdc` —— 递归收集；
- **workflows**：`{tool}/workflows/**/*.md` —— 递归收集。

来源工具按所在根目录确定（`.cursor/` → cursor，`.claude/` → claude，`.windsurf/` → windsurf，`.augment/` → augment）。LLM 输出四章报告：总览表、按工具分组（skills/rules/workflows 三栏）、质量评分、缺口分析。

``` sh
seeed-cli skills-scan
# 别名
seeed-cli skills
seeed-cli sk
```

报告落盘：`<pwd>/seeed-cli/skills-YYYY-MM-DD_HH_mm_ss.md`，附录含结构化的 `scores` / `gaps` JSON。

### skills-sync：跨工具同步 SKILL.md 文件包

选择源 skill 与目标工具，按目标工具的 frontmatter 方言重写并写入目标目录（如 `.claude/skills/<name>/SKILL.md`）。被剔除的工具专属字段（Cursor 的 `paths`、Claude 的 `allowed-tools`）会以引用块形式追加到正文顶部，避免信息丢失。

**交互模式**（多选源 skill + 单选目标工具）：

``` sh
seeed-cli skills-sync
# 别名
seeed-cli sync
```

操作键：`↑/↓` 移动 · `Space` 选/取消 · `a` 全选 · `Enter` 确认 · `q` 取消。

**非交互模式**（CI / 脚本场景）：

``` sh
seeed-cli skills-sync --target claude --skills deploy-staging,api-review --overwrite -y
```

| flag | 说明 |
|------|------|
| `--target`    | 目标工具：`cursor` \| `claude` \| `windsurf` \| `augment` |
| `--skills`    | 逗号分隔的源 skill 名（缺省进入交互式多选） |
| `--overwrite` | 目标文件已存在时是否覆盖（默认跳过并记入 Skipped） |
| `--yes` / `-y`| 跳过确认提示 |

同步结果落盘：`<pwd>/seeed-cli/skills-sync-YYYY-MM-DD_HH_mm_ss.md`，含「源 → 目标」映射表与 Written / Skipped / Failed 三段清单。

### skills-pull：联网拉取高星 skill

内嵌一份 **Top-50 高星 skill 索引**（来自 `anthropics/skills`、`vercel-labs/agent-skills`、`ComposioHQ/awesome-claude-skills` 等仓库），按需从 GitHub 拉取 SKILL.md 及其 `references/` 与 `scripts/` 子目录，再按目标工具的方言安装到 `.cursor/.claude/.windsurf/.augment` 之一。

**列表 / 搜索**（不下载）：

``` sh
seeed-cli skills-pull --list
seeed-cli pull --search pdf --list
seeed-cli pull --search github --list
```

**交互模式**（多选远程 skill + 单选目标工具）：

``` sh
seeed-cli skills-pull
# 别名
seeed-cli pull
```

操作键：`↑/↓` 移动 · `Space` 选/取消 · `a` 全选 · `Enter` 确认 · `q` 取消。

**非交互模式**（CI / 脚本场景）：

``` sh
seeed-cli pull --ids anthropics/pdf,anthropics/docx --target claude -y --overwrite
```

| flag | 说明 |
|------|------|
| `--search`    | 在 `id` / `name` / `description` / `tags` 中模糊过滤 |
| `--ids`       | 逗号分隔的 skill id 或 name（缺省进入交互式多选） |
| `--target`    | 目标工具：`cursor` \| `claude` \| `windsurf` \| `augment` |
| `--list` / `-l` | 只列出当前匹配的 skill 索引，不下载 |
| `--overwrite` | 目标文件已存在时是否覆盖 |
| `--yes` / `-y`| 跳过确认提示 |

下载过程命中 GitHub API 速率限制时，可设置 `GITHUB_TOKEN` 环境变量提升配额；下载产物写入系统临时目录，安装完成后自动清理。安装结果复用 `skills-sync` 的报告：`<pwd>/seeed-cli/skills-sync-YYYY-MM-DD_HH_mm_ss.md`。


## 安装

``` sh 
curl -fsSL https://raw.githubusercontent.com/wangzongming/seeed-cli/refs/heads/main/install.sh | bash
```
  
## API Key 配置

支持三个 LLM 平台：**BaiLian**（百炼）、**DeepSeek**、**GPT**（OpenAI）。

默认使用 BaiLian，可通过修改 `~/.seeed-cli/config.toml` 中的 `default_provider` 切换。

### 设置 API Key

``` sh
# 百炼
seeed-cli set-ak BaiLian xxx
# DeepSeek
seeed-cli set-ak DeepSeek sk-xxx
# OpenAI GPT
seeed-cli set-ak GPT sk-xxx
```

### 查看已设置的 Key

``` sh
seeed-cli get-ak BaiLian
seeed-cli get-ak DeepSeek
seeed-cli get-ak GPT
```

### 各平台默认模型

| Provider  | 默认模型          | Base URL                                              |
| --------- | ----------------- | ----------------------------------------------------- |
| BaiLian   | qwen3.6-flash     | `https://dashscope.aliyuncs.com/compatible-mode/v1`   |
| DeepSeek  | deepseek-v4-pro   | `https://api.deepseek.com/v1`                         |
| GPT       | gpt-4o            | `https://api.openai.com/v1`                           |

模型可在 `~/.seeed-cli/config.toml` 中按 provider 单独配置。

### 百炼 AK 获取教程

1. 打开 https://bailian.console.aliyun.com/cn-beijing?tab=model#/api-key 如果没有注册就注册登录
2. 点击右上角 创建 API Key
3. 复制您的 api key 并且执行 `seeed-cli set-ak BaiLian xxx(替换为您的ak)`

ps：注意账户不能欠费，欠费账号无法调用大模型。

## 新项目上手 Skill

内置了一个 Claude Code 技能，确保接触陌生项目时**先分析、再修改**。

### 安装（本项目内）

项目 `.claude/skills/seeed-cli-onboard-skill/` 已包含该 skill，拉取项目后自动生效。

### 使用

```
/seeed-cli-onboard 分析这个项目
```

### 工作流程

1. 检查 `seeed-cli` 及 API Key 是否就绪
2. `seeed-cli frame` — 架构分析（技术栈、模块边界、Mermaid 图）
3. `seeed-cli gen-ai-agent` — 生成 AI 协作约束文件 `AI-AGENT.md`
4. (按需) `seeed-cli safe-scan` / `quality` — 安全/质量扫描
5. 读取所有分析产物后，再按 `AI-AGENT.md` 约束进行代码修改

### 跨项目使用

``` sh
cp -r .claude/skills/seeed-cli-onboard-skill ~/.claude/skills/
```

## 开发环境

``` shell
go mod tidy

# 本地安装测试
make install
```

## 赞助商


- **seeed** - https://www.seeedstudio.com/

## 预览

![safe-loading](./imgs/safe-loading.png)
![safe-res](./imgs/safe-res.png)
![who-res](./imgs/who-res.png)

