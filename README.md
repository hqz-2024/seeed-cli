# 程序员的 AI 小伙伴 - seeed-cli

![help](./imgs/help.png) 

## 是什么

`seeed-cli` 在项目里统一调度 AI：把任务写成命令，复用同一套上下文，少做「每次从头讲一遍」的重复沟通。

安全扫描、质量检查、架构梳理、面向 AI 的说明、日报与提交说明、接口文档等，尽量一条链路走完。


> 基于AI的CLI工具: seeed-cli ，围绕项目开发时常见场景提供免自行驱动AI方案

## 功能/命令

| #   | 功能                                                  | 命令              | 完成 |
| --- | ----------------------------------------------------- | ----------------- | ---- |
| 1   | 项目安全扫描                                          | `sc safe-scan`    | ✓    |
| 2   | 项目代码质量检查                                      | `sc quality`      | ✓    |
| 3   | 项目架构分析                                          | `sc frame`        | ✓    |
| 4   | 项目说明生成(针对AI)，作为 AI 修改本项目的说明书      | `sc gen-ai-agent` | ✓    |
| 5   | 根据提交历史生成日报                                  | `sc gen-daily`    | ✓    |
| 5   | 根据修改文件生成提交的 commit 说明(git commit 的描述) | `sc gen-commit`   | ✓    |
| 6   | 接口文档生成                                          | `sc gen-api-doc`  | ✓    |
| 7   | 代码出处分析                                          | `sc who`          | ✓    |
| 8   | 多端兼容(windows、linux、macos)                       | —                 | ✓    |
| 9   | AI skills/rules/workflows 扫描 + 评分 + 缺口分析      | `sc skills-scan`  | ✓    |
| 10  | 跨工具 skills 同步（生成对应 SKILL.md 文件包）        | `sc skills-sync`  | ✓    |
| 11  | 联网拉取高星 skill（内嵌 Top-50 索引）                | `sc skills-pull`  | ✓    |
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
  
## 百炼 ak 获取教程

1. 打开 https://bailian.console.aliyun.com/cn-beijing?tab=model#/api-key 如果没有注册就注册登录
2. 点击右上角 创建 API Key
3. 复制您的 api key 并且执行 `seeed-cli set-ak BaiLian xxx(替换为您的ak)`

ps： 注意您的账户不能是欠费状态，欠费的账号无法调用大模型。

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

