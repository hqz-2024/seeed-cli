# 接口文档
由工具根据仓库代码扫描归纳生成，可人工修订。

## 约定
- **Base URL**：不适用（本项目为本地 CLI 工具，无 HTTP/gRPC/GraphQL 服务端点）
- **鉴权方式**：无网络鉴权。依赖本地配置文件 `config.toml` 及当前终端环境变量/Git 工作区状态。
- **调用协议**：本地进程调用。下文按标准 RESTful 规范将各子命令等价映射为 `POST /cli/{command}` 接口，实际执行请参见终端示例。
- **数据类型**：参数与响应体均遵循 JSON 序列化规范，类型标注以代码中的 Go Struct Tag 为准。
- **示例说明**：因源码未暴露网络服务，以下 `cURL` 为概念映射示例；实际使用请直接运行终端命令。无法从代码直接推断的字段已标注 `[示例占位]`。

---

## 模块：CLI 核心命令集 (`main.go` & `commands/*.go`)

### `POST /cli/set-ak` (别名: `set-ak`)
- **说明**：设置指定平台的 LLM API Key，持久化至用户目录 `config.toml`。
- **请求参数**：
  | 字段 | 类型 | 必填 | 说明 |
  |------|------|------|------|
  | `provider` | `string` | 是 | 平台标识，如 `BaiLian` |
  | `api_key` | `string` | 是 | 实际密钥值 |
- **成功响应**：更新配置文件，终端打印设置结果。
- **错误响应**：参数缺失时返回 Usage 提示；写入失败返回 `error`。
- **示例**：
  ```bash
  # cURL 概念调用
  curl -X POST /cli/set-ak -H "Content-Type: application/json" \
       -d '{"provider":"BaiLian","api_key":"sk-proj-xxxxxxxxxxxxxxxx"}'
  
  # 实际 CLI 终端调用
  seeed-cli set-ak BaiLian sk-proj-xxxxxxxxxxxxxxxx
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": { "provider": "BaiLian", "api_key": "sk-proj-xxxxxxxxxxxxxxxx" },
    "response": {
      "status": "success",
      "message": "设置 api key 成功 \"sk-proj-xxxxxxxxxxxxxxxx\"",
      "config_file_updated": "~/.seeed-cli/config.toml"
    }
  }
  ```

### `POST /cli/get-ak` (别名: `get-ak`)
- **说明**：查询已配置的指定平台 API Key。
- **请求参数**：
  | 字段 | 类型 | 必填 | 说明 |
  |------|------|------|------|
  | `provider` | `string` | 是 | 平台标识 |
- **成功响应**：返回对应密钥及存储位置。若未配置则返回未查询到提示。
- **示例**：
  ```bash
  # cURL 概念调用
  curl -X POST /cli/get-ak -H "Content-Type: application/json" -d '{"provider":"BaiLian"}'
  # 实际 CLI 终端调用
  seeed-cli get-ak BaiLian
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": { "provider": "BaiLian" },
    "response": {
      "status": "success",
      "provider": "BaiLian",
      "api_key": "[示例占位] sk-proj-xxxxxxxxxxxxxxxx",
      "source": "~/.seeed-cli/config.toml"
    }
  }
  ```

### `POST /cli/frame` (别名: `f`)
- **说明**：采集当前工作区清单文件、目录树与入口源码片段，单次调用 LLM 生成架构报告，落盘至 `seeed-cli/frame-<time>.md`。
- **请求参数**：无（默认基于 `os.Getwd()`）。
- **成功响应**：生成包含 Mermaid 图表的 Markdown 文件；终端流式展示进度。
- **示例**：
  ```bash
  seeed-cli frame
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "output_path": "./seeed-cli/frame-[日期时间].md",
      "content_structure": ["技术栈分析", "架构与数据流(Mermaid)", "关键模块分析", "快速上手路线"],
      "corpus_truncated_limit_bytes": 200000
    }
  }
  ```

### `POST /cli/safe-scan` (别名: `safe`)
- **说明**：遍历代码文件分批调用 LLM 进行安全审计，报告写回 `seeed-cli/scan-safe-<time>.md`。
- **请求参数**：无。
- **示例**：
  ```bash
  seeed-cli safe-scan
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "output_path": "./seeed-cli/scan-safe-[日期时间].md",
      "prompt_focus": "安全问题排查(数量/文件地址/代码/描述/修复方案)"
    }
  }
  ```

### `POST /cli/quality` (别名: `q`)
- **说明**：遍历代码文件分批调用 LLM 进行宽松代码质量点评，报告写回 `seeed-cli/scan-quality-<time>.md`。
- **请求参数**：无。
- **示例**：
  ```bash
  seeed-cli quality
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "output_path": "./seeed-cli/scan-quality-[日期时间].md",
      "evaluation_aspects": ["亮点", "可改进点(低/中优先级)", "文件路径列表"]
    }
  }
  ```

### `POST /cli/gen-commit` (别名: `commit`)
- **说明**：读取 `git status --porcelain` 与 `git diff --cached`，经 LLM 生成带类型前缀与图标的单行 Commit Message。全文落盘，stdout 仅输出首行（便于管道复用）。
- **前置依赖**：环境变量需包含 `git` 命令；暂存区需有变更。
- **请求参数**：
  | 字段 | 类型 | 必填 | 说明 |
  |------|------|------|------|
  | `git_status_scope` | `string` | 否 | 扫描范围，默认为 `cached_diff` |
- **示例**：
  ```bash
  git add . && seeed-cli gen-commit
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": { "git_status_scope": "cached_diff" },
    "response": {
      "status": "success",
      "stdout_single_line": "[示例占位] ✨ Feat: 新增批量代码架构分析与报告生成功能",
      "output_file": "./seeed-cli/gen-commit-[日期时间].md",
      "supported_types": ["Feat", "Fix", "Docs", "Style", "Refactor", "Chore", "Perf", "Test"]
    }
  }
  ```

### `POST /cli/gen-daily` (别名: `daily`)
- **说明**：拉取当日 Git 提交记录，经 LLM 生成纯文本工作日报，落盘至 `seeed-cli/daily-<time>.txt`。
- **请求参数**：无（自动计算当日 UTC+0 起始时间戳）。
- **示例**：
  ```bash
  seeed-cli gen-daily
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": { "date_range": "today_00:00_to_23:59" },
    "response": {
      "status": "success",
      "output_file": "./seeed-cli/daily-[日期时间].txt",
      "format": "plain_text_no_markdown",
      "structure": ["工作日报 YYYY-MM-DD", "主要工作内容", "小结"]
    }
  }
  ```

### `POST /cli/gen-ai-agent` (别名: `agent`)
- **说明**：扫描当前目录源码，按 `512KB` 批次分批送 LLM 归纳，最终合并生成 `./AI-AGENT.md`。
- **请求参数**：无。
- **示例**：
  ```bash
  seeed-cli gen-ai-agent
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "output_file": "./AI-AGENT.md",
      "document_title": "# AI 协作说明",
      "sections_inferrable_from_code": ["技术栈与依赖", "目录与模块", "代码风格", "测试与质量", "配置与环境"]
    }
  }
  ```

### `POST /cli/gen-api-doc` (别名: `api`)
- **说明**：扫描当前目录源码，按批次归纳路由/Handler/Struct Tag，合并生成 `./API-DOC.md`（强制含 curl 与 JSON 示例占位）。
- **请求参数**：无。
- **示例**：
  ```bash
  seeed-cli gen-api-doc
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "output_file": "./API-DOC.md",
      "document_title": "# 接口文档",
      "grouping_logic": "按模块/服务分组，同 path+method 去重",
      "example_format": "curl + JSON (缺省值标注示例值)"
    }
  }
  ```

### `POST /cli/who` (别名: `who`)
- **说明**：粗略统计源码中“古法/手写”与“偏AI辅助”风格占比，终端绘制 ASCII 比例条并打印百分比。
- **请求参数**：无。
- **示例**：
  ```bash
  seeed-cli who
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "traditional_estimate_pct": "[示例占位] 35",
      "ai_style_estimate_pct": "[示例占位] 65",
      "visual_output": "[示例占位] ███████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░",
      "caveat": "LLM 主观估计，非精确科学验证"
    }
  }
  ```

### `POST /cli/clear` (别名: `clear`)
- **说明**：强制删除用户主目录下的配置文件 (`~/.seeed-cli/config.toml`)。若文件不存在则静默成功。
- **请求参数**：无。
- **示例**：
  ```bash
  seeed-cli clear
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "deleted_path": "~/.seeed-cli/config.toml",
      "message": "信息清除成功"
    }
  }
  ```

### `POST /cli/install-s` (别名: `install-s`)
- **说明**：在可执行文件同级目录创建名为 `s` (Windows 为 `s.exe`) 的软链接指向本程序，便于快捷调用。
- **请求参数**：无。
- **示例**：
  ```bash
  seeed-cli install-s
  ```
  **请求/响应 JSON 示例**：
  ```json
  {
    "request": {},
    "response": {
      "status": "success",
      "symlink_created": "[示例占位] /usr/local/bin/s",
      "target_binary": "[示例占位] /usr/local/bin/seeed-cli",
      "windows_note": "可能需要启用开发人员模式或管理员权限"
    }
  }
  ```

---
**注**：本批次代码仅包含 CLI 客户端逻辑与本地文件/LLM 交互封装，未定义服务端监听端口或网络协议路由。以上归纳已将 CLI 子命令按标准 API 规范结构化映射，字段名、参数类型与示例值严格对齐源码中的 `struct` 定义、`flag` 解析逻辑及标准输出格式。实际业务接入请直接使用终端命令执行。