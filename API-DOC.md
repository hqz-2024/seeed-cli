# 接口文档
> 由工具根据仓库代码扫描归纳生成，可人工修订。

## 📜 通用约定
- **协议与基础路径**：本工程为纯终端 CLI 工具（基于 `` `urfave/cli/v3` `` 构建），**未暴露任何 HTTP Server / Base URL**。以下“接口”实为本地可执行的 CLI 子命令。
- **鉴权方式**：依赖本地配置文件 `` `~/.seeed-cli/config.toml` ``。密钥通过 `` `set-ak` `` 命令写入，内部直连 LLM 兼容端点（如 `` `https://dashscope.aliyuncs.com/compatible-mode/v1` ``），无需传递网络鉴权头。
- **交互与返回格式**：同步终端执行。成功时向 stdout 打印文本或返回 `` `nil` ``；异常时打印 Usage/Error 并返回非零退出码。部分命令内置全屏 `` `BubbleTea` `` UI 与流式渲染。
- **数据落盘**：分析/报告类命令默认将结果保存至 `` `seeed-cli/` `` 目录或工程根目录的对应后缀文件中。

---

## 🛠 配置管理

### 命令：`set-ak`
- **简要说明**：设置指定平台的 LLM API Key 至本地 TOML 配置文件。
- **参数定义**：
  - `` `Args[0]` `` `` `provider` `` `` `string` ``：平台标识（如 `` `BaiLian` ``，对应源码枚举 `` `configs.Provider.BaiLian` ``）
  - `` `Args[1]` `` `` `ak` `` `` `string` ``：API Key 凭证字符串
- **响应形态**：成功时标准输出明文提示并返回 `` `nil` ``；缺失参数或存储异常时打印 usage 或返回 `` `error` ``。
- **调用示例（curl）**：
  ```bash
  seeed-cli set-ak BaiLian sk-xxxxxxxxxxxxxxxxxxxxxxxx
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "set-ak",
    "request": {
      "provider": "BaiLian",
      "ak": "sk-example-placeholder"
    },
    "response": {
      "status": "success",
      "message": "设置 api key 成功 \"%s\"",
      "_note": "示例值；实际输出为标准终端字符串"
    }
  }
  ```

### 命令：`get-ak`
- **简要说明**：读取并打印指定平台的已配置 API Key。
- **参数定义**：
  - `` `Args[0]` `` `` `provider` `` `` `string` ``：平台标识（如 `` `BaiLian` ``）
- **响应形态**：成功时标准输出明文提示并返回 `` `nil` ``；未查询到或配置不存在时打印提示或返回 `` `error` ``。
- **调用示例（curl）**：
  ```bash
  seeed-cli get-ak BaiLian
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "get-ak",
    "request": {
      "provider": "BaiLian"
    },
    "response": {
      "status": "success",
      "data": {
        "api_key": "sk-example-placeholder"
      },
      "_note": "示例值；实际输出为标准终端字符串"
    }
  }
  ```

---

## 🔍 项目分析与扫描

### 命令：`frame`
- **简要说明**：采集当前工作目录源码片段与目录树，流式调用 LLM 生成《项目框架分析报告》并落盘。
- **参数定义**：无（默认使用 `` `os.Getwd()` `` 定位当前目录）
- **响应形态**：全屏 BubbleTea UI 展示进度与 LLM 流式输出；完成后返回文件保存路径。
- **调用示例（curl）**：
  ```bash
  seeed-cli frame
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "frame",
    "request": {},
    "response": {
      "status": "success",
      "output_file": "seeed-cli/frame-YYYY-MM-DD_HH_mm_ss.md",
      "_note": "示例值；实际交互为终端全屏 UI 与 Markdown 文件写入"
    }
  }
  ```

### 命令：`safe-scan`（别名：`ss`）
- **简要说明**：遍历当前目录源代码，分批送审 LLM 进行安全漏洞检查，报告落盘。
- **参数定义**：无
- **响应形态**：全屏 UI 展示扫描进度与安全审查摘要；成功后返回落盘路径。
- **调用示例（curl）**：
  ```bash
  seeed-cli safe-scan
  # 或
  seeed-cli ss
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "safe-scan",
    "request": {},
    "response": {
      "status": "success",
      "output_file": "seeed-cli/scan-safe-YYYY-MM-DD_HH_mm_ss.md",
      "_note": "示例值"
    }
  }
  ```

### 命令：`quality`（别名：`q`）
- **简要说明**：遍历源代码分批调用 LLM 进行宽松风格的质量点评，报告落盘。
- **参数定义**：无
- **响应形态**：同 `` `safe-scan` ``，UI 展示与 Markdown 报告输出逻辑一致。
- **调用示例（curl）**：
  ```bash
  seeed-cli quality
  # 或
  seeed-cli q
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "quality",
    "request": {},
    "response": {
      "status": "success",
      "output_file": "seeed-cli/scan-quality-YYYY-MM-DD_HH_mm_ss.md",
      "_note": "示例值"
    }
  }
  ```

---

## 📝 代码与文档生成

### 命令：`gen-commit`（别名：`gc`）
- **简要说明**：读取 `` `git status` `` 与 `` `git diff --cached` ``，经 LLM 生成符合规范的单行 Commit Message 并输出至 stdout。
- **参数定义**：无（依赖 Git 暂存区状态）
- **响应形态**：直接打印生成的提交描述字符串；若暂存区为空或 LLM 失败则返回 `` `error` ``。
- **调用示例（curl）**：
  ```bash
  git add . && seeed-cli gen-commit
  # 或
  seeed-cli gc
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "gen-commit",
    "request": {},
    "response": {
      "status": "success",
      "commit_msg": "✨ Feat: 新增用户注册功能",
      "_note": "示例值；实际输出为标准终端单行文本"
    }
  }
  ```

### 命令：`gen-daily`（别名：`gd`）
- **简要说明**：拉取当日 Git 提交记录，经 LLM 生成纯文本工作日报并落盘。
- **参数定义**：无（默认取当天日期范围）
- **响应形态**：全屏 UI 展示生成过程；完成后返回 `` `.txt` `` 落盘路径。
- **调用示例（curl）**：
  ```bash
  seeed-cli gen-daily
  # 或
  seeed-cli gd
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "gen-daily",
    "request": {},
    "response": {
      "status": "success",
      "output_file": "seeed-cli/daily-YYYY-MM-DD_HH_mm_ss.txt",
      "_note": "示例值"
    }
  }
  ```

### 命令：`gen-ai-agent`（别名：`gaa`）
- **简要说明**：扫描当前目录受支持的源码文件，分批归纳技术栈/规范/约束，合并生成 `` `AI-AGENT.md` ``。
- **参数定义**：无
- **响应形态**：全屏 UI 展示扫描与合并进度；完成后返回 `` `AI-AGENT.md` `` 落盘路径。
- **调用示例（curl）**：
  ```bash
  seeed-cli gen-ai-agent
  # 或
  seeed-cli gaa
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "gen-ai-agent",
    "request": {},
    "response": {
      "status": "success",
      "output_file": "AI-AGENT.md",
      "_note": "示例值"
    }
  }
  ```

### 命令：`gen-api-doc`（别名：`gad`）
- **简要说明**：扫描当前目录源码，提取路由/Handler/结构体等信息，分批归纳并合并生成 `` `API-DOC.md` ``（含示例段）。
- **参数定义**：无
- **响应形态**：同 `` `gen-ai-agent` ``，输出文件名为 `` `API-DOC.md` ``。
- **调用示例（curl）**：
  ```bash
  seeed-cli gen-api-doc
  # 或
  seeed-cli gad
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "gen-api-doc",
    "request": {},
    "response": {
      "status": "success",
      "output_file": "API-DOC.md",
      "_note": "示例值"
    }
  }
  ```

---

## 🧹 系统维护

### 命令：`clear`
- **简要说明**：删除本地配置文件（`` `~/.seeed-cli/config.toml` ``），重置全部密钥与设置。
- **参数定义**：无
- **响应形态**：成功返回 `` `nil` `` 并打印提示；文件不存在则静默跳过。
- **调用示例（curl）**：
  ```bash
  seeed-cli clear
  ```
- **请求/响应 JSON**：
  ```json
  {
    "command": "clear",
    "request": {},
    "response": {
      "status": "success",
      "message": "信息清除成功",
      "_note": "示例值；实际输出为标准终端字符串"
    }
  }
  ```

> **架构备注**：本批次代码均为终端 CLI 实现，未对外提供 Web API 网关、gRPC Server 或 GraphQL Schema。上述条目已严格对照 `` `main.go` `` 路由注册表与对应 Handler 源码提取，可直接用于本地自动化脚本调用或开发环境辅助工作流。