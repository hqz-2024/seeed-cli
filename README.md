# 程序员的 AI 小伙伴 - seeed-cli

## What ?
让 `seeed-cli` 帮你使唤 `AI`，无需每次再手动鞭策 AI 干活。

`seeed-cli` 利用 `AI` 对你的项目进行一站式管理.


## 功能点

| #   | 功能                                                  | 命令              | 完成 |
| --- | ----------------------------------------------------- | ----------------- | ---- |
| 1   | 项目安全扫描                                          | `sc safe-scan`    | ✓    |
| 2   | 项目代码质量检查                                      | `sc quality`      | ✓    |
| 3   | 项目架构分析                                          | `sc frame`        | ✓    |
| 4   | 项目说明生成(针对AI)，作为 AI 修改本项目的说明书      | `sc gen-ai-agent` | ✓    |
| 5   | 根据提交历史生成日报                                  | `sc gen-daily`    | ✓    |
| 5   | 根据修改文件生成提交的 commit 说明(git commit 的描述) | `sc gen-commit`   | ✓    |
| 6   | 接口文档生成                                          | `sc gen-api-doc`  | ✓    |
| 7   | 代码出处分析                                          | `sc who`          |      |
| 8   | skill 扩展                                            | —                 |      |
| 9   | 炫酷 ai 聊天                                          | —                 |      |
| 10  | 多端兼容(windows、linux、macos)                       | —                 | ✓    |



## 技术栈

开发语言 `go`
cli 库  `cli.urfave`


## 安装

``` sh
curl -fsSL https://raw.githubusercontent.com/wangzongmming/seeed-cli/main/install.sh | bash
```

## 快速开始
ing...


## 开发环境

``` shell
go mod tidy
make install
```