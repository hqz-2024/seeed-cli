# seeed-cli 命令参考

## API Key 管理

| 命令 | 说明 |
|------|------|
| `seeed-cli set-ak [BaiLian\|DeepSeek\|GPT] <key>` | 设置 API Key |
| `seeed-cli get-ak [BaiLian\|DeepSeek\|GPT]` | 查看 API Key |

## 分析命令

| 命令 | 别名 | 输出 | 说明 |
|------|------|------|------|
| `seeed-cli frame` | `f` | `seeed-cli/frame-*.md` | 架构分析（含 Mermaid 图） |
| `seeed-cli safe-scan` | `safe` | `seeed-cli/scan-safe-*.md` | 安全扫描 |
| `seeed-cli quality` | `q` | `seeed-cli/scan-quality-*.md` | 代码质量点评 |
| `seeed-cli who` | - | TUI 展示 | 代码出处（古法 vs AI）分析 |

## 文档生成

| 命令 | 别名 | 输出 | 说明 |
|------|------|------|------|
| `seeed-cli gen-ai-agent` | `agent` | `AI-AGENT.md` | AI 协作说明（项目根目录） |
| `seeed-cli gen-api-doc` | `api` | `API-DOC.md` | 接口文档（项目根目录） |
| `seeed-cli gen-commit` | `commit` | `seeed-cli/gen-commit-*.md` | 提交说明生成 |
| `seeed-cli gen-daily` | `daily` | `seeed-cli/daily-*.txt` | 日报生成 |

## Skills 管理

| 命令 | 别名 | 说明 |
|------|------|------|
| `seeed-cli skills-scan` | `skills`, `sk` | AI 工具 skills 扫描 + 评分 + 缺口分析 |
| `seeed-cli skills-sync` | `sync` | 跨工具 skills 同步 |
| `seeed-cli skills-pull` | `pull` | 联网拉取高星 skill |

## 平台与模型

默认使用 BaiLian（百炼），可通过 `~/.seeed-cli/config.toml` 中的 `default_provider` 切换到 DeepSeek 或 GPT。

| Provider | Base URL | 默认模型 |
|----------|----------|---------|
| BaiLian | `dashscope.aliyuncs.com/compatible-mode/v1` | qwen3.6-flash |
| DeepSeek | `api.deepseek.com/v1` | deepseek-v4-pro |
| GPT | `api.openai.com/v1` | gpt-4o |
