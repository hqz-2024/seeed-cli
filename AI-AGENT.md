# AI 协作说明
由工具根据仓库代码扫描归纳生成，可人工修订。

## 1. 技术栈与核心依赖
- **运行时**：Go
- **CLI 框架**：`github.com/urfave/cli/v3`
- **终端交互 (TUI)**：`charm.land/bubbletea/v2`、`charm.land/lipgloss/v2`
- **渲染与动效**：`charm.land/glamour/v2`、`charm.land/bubbles/v2/spinner`、`charm.land/bubbles/v2/viewport`
- **配置解析**：`github.com/BurntSushi/toml`
- **LLM SDK**：`github.com/openai/openai-go`（底层基地址硬编码为阿里云百炼/DashScope 兼容端点：`https://dashscope.aliyuncs.com/compatible-mode/v1`）
- **安装交付**：原生 Bash 脚本结合 `curl`，原生支持 Darwin/Linux + amd64/arm64 的二进制下载与权限配置。

## 2. 目录结构与模块划分
| 路径 | 职责说明 |
|:---|:---|
| `main.go` | CLI 命令注册、全局初始化与启动入口 |
| `install.sh` | 一键环境准备与二进制部署脚本 |
| `commands/` | 集中承载所有子命令的 Handler 与核心业务逻辑（架构分析、安全扫描、代码质量、日报/提交生成、AI-Agent 文档自举等） |
| `commands/configs/` | 独立配置管理包，封装 TOML 读写、用户主目录路径计算与结构化数据映射 |
| `commands/funcs/` | 通用能力层，包含 LLM 流式客户端封装、文件后缀/目录黑白名单判断、全量统计逻辑等 |

> 📦 **包导入约定**：所有内部引用统一使用 `seeed-cli/commands/...` 前缀。后续新增或重构模块时，请严格保持该 Go Module Name 路径一致性。

## 3. 配置与环境约定
- **存储格式与路径**：TOML 格式，默认生效路径为 `~/.seeed-cli/config.toml`。
- **数据结构设计**：扁平化。包含基础元数据字段（`Name` / `Version` / `Desc`）与单一 Provider 节点。当前仅实现 `BaiLian` 配置块，内含 `LLMConfig`（字段：`Model`、`ApiKey`）。
- **初始化与加载策略**：
  1. 优先读取默认路径 `~/.seeed-cli/config.toml`。
  2. 若默认路径缺失，尝试兜底解析同级相对路径 `config.toml`。
  3. 无论加载来源如何，成功解析后均会序列化并写入默认路径。若目标文件已存在，将执行覆盖写入。
- **参数传递机制**：**不依赖**环境变量或命令行标志。模型名称与 API Key 完全通过本地配置文件注入，客户端初始化时通过 `option.WithAPIKey` 与 `option.WithBaseURL` 传入。

## 4. 编码规范与架构模式
- **命名惯例**：
  - CLI 路由入口：`HandleXxx`
  - 后台异步执行逻辑：`runXxxWork`
  - 配置读写接口：`GetXxx` / `SetXxx`
  - LLM 调用封装：`FetchLLM` / `FetchLLMStream`
- **常量与规则前置**：Prompt 模板、截断阈值、忽略目录列表、受支持后缀等静态规则均提取为包级常量或 `map[string]struct{}`，禁止在业务逻辑中硬编码魔法字符串。
- **批处理与累加器模式**：文件遍历采用内存累加策略。达到预设阈值或遍历结束时触发 `flush`。单次累积超限直接物理截断，并在尾部追加 `_truncated_` 标记以保证下游解析安全。
- **字符串组装**：高频使用 `strings.Builder` 拼接上下文，手动控制换行符与分隔符，确保 LLM 输入结构的稳定性。
- **注释风格**：中英混用。核心函数上方采用 `/** */` 块注释声明职责，其余以单行 `//` 为主。多处保留防御性注释（如 `/* 避免把本工具输出扫进语料 */`、`// 原本就不存在…`），重构时请务必保留。

## 5. AI 交互与上下文控制约束
- **Prompt 强格式控制**：各子命令内置严格输出规范，代码生成/改写时必须遵守：
  - `commit` 类：限定 `Emoji + 类型 + 中文简述` 格式。
  - `daily-report` 类：**禁止** Markdown 语法，仅限纯文本。
  - `arch-review` 类：强制章节顺序，Mermaid 图表需符合语法限制。
- **防幻觉与边界声明**：Prompt 链中反复注入系统指令：`“不要编造”`、`“未在上下文中出现则写未知”`、`“冲突以后者/更具体者为准”`。代码层面对超长语料实施物理截断（如切片限制 `corpus[:200000]`），杜绝幻觉扩散。
- **流式 UI 隔离**：LLM 响应通过 Bubbletea 自定义消息（`repStreamDeltaMsg` / `repStreamEndMsg`）异步推送至 `Viewport`。TTY 环境下，Mermaid 代码围栏会被 `glamour_tty.go` 透明替换为纯文本以防渲染错乱，文件落盘时自动恢复原始 Markdown 语法。
- **分批合并机制**：`gen-ai-agent` 模块采用两阶段流程：① 分块获取草稿 → ② 调用二次 LLM 进行去重、结构对齐与冲突消解 → 输出标准化 `AI-AGENT.md`。
- **上下文负载控制**：严格限制扫描深度（目录树 `depth ≤ 5`）、单文件大小上限（`MaxFileSize = 1MB`）及单次 Prompt Token 预算，防止推理阻塞或溢出。*(注：内部批处理阈值含 200KB/512KB 等多档，具体动态阈值视场景而定，以实际运行日志为准。)*

## 6. 测试与质量保障
- **单元测试现状**：当前仓库未引入 `*_test.go` 文件，暂无 Mock 组织方式或标准测试套件。
- **运行时自检与修复**：依赖 LLM 动态生成报告与自举文档。Prompt 内置 `“先肯定后建议”`、`“明确列出涉及文件”` 等约束，利用大模型自身能力完成部分逻辑校验与自我修正。
- **静态过滤约定**：通过代码层的 `CodeExt`（支持后缀白名单）与 `IgnoreDir`（噪声目录黑名单）映射表提前拦截非源码文件，降低无效分析成本。

---

### 🚫 禁止事项
- **严禁编造**：不得虚构仓库中不存在的 API、路径、配置文件键值、第三方依赖或环境变量。无法确定的逻辑一律标记为 `待确认`，禁止臆测填充。
- **严禁越权修改**：不得修改或绕过已有的 TUI 渲染逻辑、LLM 客户端配置注入方式及 `~/.seeed-cli/config.toml` 的加载与覆盖协议。
- **遵守输出限制**：生成或改造内容必须严格遵守各子命令绑定的 Prompt 格式规范（如纯文本日报、特定 Commit 前缀等），不得随意添加 Markdown 装饰或变更排版结构。
- **上下文安全红线**：遇到超阈值文件或超长上下文时，必须沿用现有的截断与累加器机制，不可自行改变分块策略或移除 `_truncated_` 标记。