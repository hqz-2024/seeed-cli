# AI 协作说明
<sub>由工具根据仓库代码扫描归纳生成，可人工修订。</sub>

## 项目架构与目录规范
- **入口与路由**：`main.go` 为唯一程序入口。命令路由与业务处理器统一收敛至 `commands/` 目录。
- **模块划分**：
  - `commands/configs/`：配置读写、路径计算与默认值管理
  - `commands/funcs/`：跨命令共享核心逻辑（LLM 流式调用、技能解析、文件过滤、方言适配等）
  - `assets/`：静态资源声明与内嵌配置
- **遍历与过滤策略**：使用 `filepath.Walk` 时严格遵循全局 `IgnoreDir`（排除版本控制、依赖、缓存及 IDE 目录）与 `CodeExt`（限定可扫描后缀）。硬编码 `MaxFileSize = 1MB` 防止单次扫描占用过高内存。
- **输出落盘**：所有分析/报告类指令结果统一写入执行目录下的 `./seeed-cli/` 子目录，文件名需带时间戳前缀（格式如 `frame-2006-01-02_15_04_05.md`）。

## 技术栈与依赖清单
- **语言与运行时**：Go（重度依赖标准库，无重型业务框架）
- **CLI 框架**：`github.com/urfave/cli/v3`
- **TUI/UI 渲染**：核心交互循环使用 `charm.land/bubbletea/v2`；样式排版使用 `charm.land/lipgloss/v2`；Markdown 转 ANSI 使用 `charm.land/glamour/v2`；辅助组件含 `spinner` 与 `viewport`
- **LLM 客户端**：`github.com/openai/openai-go`（兼容 OpenAI 协议，实际对接百度百炼、DeepSeek、OpenAI）
- **序列化与解析**：`github.com/BurntSushi/toml`（配置）、`gopkg.in/yaml.v3`（索引清单）、`regexp`（响应文本提取）
- **资源内嵌**：通过 `go:embed` 将 `assets/skills-top50.yaml` 编译至二进制，消除首次运行时的网络依赖。

## 配置与环境变量
- **配置文件**：TOML 格式，默认路径 `$HOME/.seeed-cli/config.toml`。结构包含基础元信息（`Name`/`Version`/`Desc`）、默认 Provider 配置，以及按厂商分组的 Provider 结构体（各含 `Model` 与 `ApiKey`）。
- **环境变量**：
  - `GITHUB_TOKEN`：用于 GitHub Contents API 鉴权（自动附加至 HTTP Header）
  - `SEEED_PLAIN`：值为非空或检测到非交互式终端时，强制关闭 Glamour 富文本渲染，降级为纯文本换行输出
- **初始化逻辑**：`main()` 显式调用 `configs.Init()`。若配置文件缺失，则自动创建目录并写入内置默认模板。

## 终端 UI 与交互契约
- **全屏复用架构**：`rep_ui.go` 封装通用 Bubble Tea Model，提供标题栏、带行号侧栏的视口、底部状态条与 Spinner，供各子命令直接挂载调用。
- **快捷键统一规范**：菜单导航均支持 `↑/↓` 或 `k/j`，`Space` 切换选中，`a` 全选，`Enter` 确认，`q`/`Esc`/`Ctrl+C` 安全退出。
- **异步消息总线**：后台 Goroutine 产生的日志通过 `repAppendLogMsg`、`repStreamDeltaMsg` 等消息类型安全投递至 UI 视口，彻底避免并发写屏冲突。
- **动态适配与降级**：监听 `tea.WindowSizeMsg` 实时重算视口宽度与自动换行阈值；检测 fenced code block 中的 `mermaid` 标记时，`glamour_tty.go` 会临时替换为 `text` 并追加防错注释，落盘文件仍保留原始 Mermaid 语法。
- **日志规范**：不引入 `zap`/`logrus` 等外部日志库。进度反馈完全基于 `lipgloss` 封装的 `bootOKLine()`/`bootWaitLine()`，配合颜色与 `[ OK ]`/`[ .. ]` 标签实现流式 Delta 推送。

## LLM 集成与提示词工程
- **分批流式处理**：源码按 `512KB` 阈值切块喂给 LLM；第二阶段长文档合并时严格限制输入上下文上限为 `120KB`。
- **两阶段流水线**：第一阶段独立生成本地批次草稿 → 收集有效响应后触发第二阶段，进行去重、冲突仲裁与结构重组。
- **结构化提取**：依赖正则表达式（如 `reWhoLine`）与 Markdown 表格特征匹配解析 LLM 返回内容。解析失败时仅标注警告或回退，绝不做强类型断言。
- **强约束 Prompt 模板**：`gen-ai-agent.go`、`gen-api-doc.go`、`skills.go`、`who.go` 均内置严格 Prompt，强制规定章节顺序、表格列名、输出语言及示例格式。高频注入“严禁臆测”、“未出现内容不编造”、“读不清标待确认”等防幻觉指令。

## 编码规范与错误处理
- **命名约定**：导出函数严格 PascalCase（如 `HandleSetAK`、`ScanSkills`）；内部辅助函数优先动词开头（如 `runRepUI`、`flushBatch`）。文件命名采用连字符（命令入口 `api-key.go`）与下划线（内部功能 `skills_dialect.go`）区分。
- **注释风格**：包级文件头部强制一行说明职责；混用 `//` 单行与 `/** ... */` 多行注释；不使用完整 Godoc 块。
- **错误处理**：全线采用 `if err != nil { return err }` 逐层透传。仅在 `main.go` 初始化阶段的致命错误使用 `panic(err)`。
- **降级与兜底**：配置加载失败打印警告后终止；文档生成类命令在 LLM 合并失败时自动回退至“分批草稿拼接”模式；Provider 缺失时默认降级为 `BaiLian`。
- **上下文与安全**：Handler 接收 `context.Context` 并透传至 `repModel` 与 LLM 请求，保障超时与取消机制生效。路径操作强制使用 `filepath.Clean`、`os.IsNotExist`、`info.IsDir()` 及软链接校验，严防越权覆盖。

## 构建与安装规范
- **自动化安装**：`install.sh` 自动识别 macOS/Linux 与 x86_64/arm64 架构，从 GitHub Releases `latest/download/` 拉取对应二进制，赋予权限后安装至 `/usr/local/bin`。
- **快捷命令绑定**：内置 `install-s` 命令，在可执行文件同级目录幂等创建指向自身的符号链接 `s`（Windows 平台为 `s.exe`），并处理权限降级提示。

---

## ⚠️ 禁止事项（AI Agent 必读）
- **绝对禁止编造**：不得虚构本仓库中不存在的 API、函数签名、文件路径、配置项或第三方依赖。
- **严格遵循约定**：输出路径必须为 `./seeed-cli/` 且带时间戳；UI 键位与 TUI 交互逻辑不得擅自修改；Prompt 中的防幻觉约束为最高优先级。
- **无法确定时标注**：遇逻辑矛盾、缺失细节或架构不明处，统一标记为 `[待确认]` 并暂停推断，严禁自行脑补实现。