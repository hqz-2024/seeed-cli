# 程序员的 AI 小伙伴 - seeed-cli

## 是什么

`seeed-cli` 在项目里统一调度 AI：把任务写成命令，复用同一套上下文，少做「每次从头讲一遍」的重复沟通。

安全扫描、质量检查、架构梳理、面向 AI 的说明、日报与提交说明、接口文档等，尽量一条链路走完。


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
| 9   | skill 扩展                                            | —                 | x    |
| 10  | 炫酷 ai 聊天                                          | —                 | x    |
| 11  | 想象中...                                             | —                 | -    |


## 安装

``` sh
curl -fsSL https://github.com/wangzongming/seeed-cli/blob/main/install.sh | bash
```
  
## 开发环境

``` shell
go mod tidy

# 本地安装测试
make install
```