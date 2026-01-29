# Implementation Plan: lad Buildspec Integration

## Overview

在 flow 工程的 `lambda.buildspec.yaml` 中集成 lad 工具，添加 OptLad 和 OptCanary 参数支持，实现 Lambda 函数的灰度发布功能。

## Tasks

- [ ] 1. 更新 buildspec 文件头部注释
  - 在 Opt参数 部分添加 OptLad 和 OptCanary 参数说明
  - _Requirements: 1.1-1.7, 2.1-2.15_

- [ ] 2. 添加 lad 工具安装和执行逻辑
  - [ ] 2.1 在 SAM deploy 之后添加 lad 操作代码块
    - 添加 OptLad 值检查
    - 添加 lad 工具安装逻辑（go install）
    - 添加安装验证
    - _Requirements: 3.1, 3.2, 3.3_
  
  - [ ] 2.2 实现 OptCanary 解析逻辑
    - 实现 canary 策略解析（canary_0 到 canary_100）
    - 实现 auto 等待时间解析（auto_1_minute 到 auto_1_day）
    - _Requirements: 2.1-2.13_
  
  - [ ] 2.3 实现 lad 命令执行逻辑
    - 实现 deploy 模式执行
    - 实现 canary 模式执行（带策略参数）
    - 实现 promote 模式执行
    - 实现 rollback 模式执行
    - 实现 auto 模式执行（带等待时间参数）
    - _Requirements: 1.2, 1.3, 1.4, 1.5, 1.6_
  
  - [ ] 2.4 实现错误处理和日志输出
    - 添加无效 OptLad 值处理
    - 添加无效 OptCanary 值处理
    - 添加命令执行前后日志
    - 添加命令失败处理
    - _Requirements: 1.7, 2.14, 2.15, 5.1, 5.2, 5.3, 5.4_

- [ ] 3. Checkpoint - 验证 buildspec 语法
  - 确保 YAML 语法正确
  - 确保所有 shell 脚本语法正确
  - 如有问题请提出

## Notes

- 所有修改都在 `flow/aws/codebuild/lambda.buildspec.yaml` 文件中进行
- lad 操作在 SAM deploy 成功后执行
- 使用 `go install` 安装 lad 工具，需要 Go 环境（buildspec 中已有 Go 环境）
- 环境参数 `--env` 从 OptRuntime 推断（test 或 prod）
