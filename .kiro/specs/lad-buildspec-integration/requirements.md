# Requirements Document

## Introduction

本文档定义了在 flow 工程的 `lambda.buildspec.yaml` 中集成 lad (Lambda Alias Deployment) 工具的需求。通过添加 `OptLad` 和 `OptCanary` 参数，实现 Lambda 函数的灰度发布功能。

## Glossary

- **Buildspec**: AWS CodeBuild 的构建规范文件，定义构建阶段和命令
- **lad**: Lambda Alias Deployment 工具，用于管理 Lambda 函数版本和别名的灰度发布
- **OptLad**: 控制 lad 工具操作模式的参数
- **OptCanary**: 指定灰度策略的参数
- **Canary_Strategy**: 灰度策略，定义流量分配比例
- **Auto_Strategy**: 自动递进策略，按时间间隔自动递进灰度比例
- **SAM_Deploy**: AWS SAM CLI 的部署命令

## Requirements

### Requirement 1: OptLad 参数支持

**User Story:** As a DevOps engineer, I want to control lad operations through the OptLad parameter, so that I can choose different deployment strategies during the build process.

#### Acceptance Criteria

1. WHEN OptLad is set to "none", THE Buildspec SHALL skip all lad operations
2. WHEN OptLad is set to "deploy", THE Buildspec SHALL execute `lad deploy` to deploy a new version
3. WHEN OptLad is set to "canary", THE Buildspec SHALL execute `lad canary` with the specified strategy
4. WHEN OptLad is set to "promote", THE Buildspec SHALL execute `lad promote` to complete the release
5. WHEN OptLad is set to "rollback", THE Buildspec SHALL execute `lad rollback` to revert to the previous version
6. WHEN OptLad is set to "auto", THE Buildspec SHALL execute `lad auto` for automatic progressive canary release
7. IF OptLad is set to an invalid value, THEN THE Buildspec SHALL log an error and skip lad operations

### Requirement 2: OptCanary 参数支持

**User Story:** As a DevOps engineer, I want to specify canary strategies through the OptCanary parameter, so that I can control traffic distribution during canary releases.

#### Acceptance Criteria

1. WHEN OptCanary is set to "canary_0", THE Buildspec SHALL pass `--strategy canary0` to lad canary command
2. WHEN OptCanary is set to "canary_10", THE Buildspec SHALL pass `--strategy canary10` to lad canary command
3. WHEN OptCanary is set to "canary_25", THE Buildspec SHALL pass `--strategy canary25` to lad canary command
4. WHEN OptCanary is set to "canary_50", THE Buildspec SHALL pass `--strategy canary50` to lad canary command
5. WHEN OptCanary is set to "canary_75", THE Buildspec SHALL pass `--strategy canary75` to lad canary command
6. WHEN OptCanary is set to "canary_100", THE Buildspec SHALL pass `--strategy canary100` to lad canary command
7. WHEN OptCanary is set to "auto_1_minute", THE Buildspec SHALL pass `--wait 1m` to lad auto command
8. WHEN OptCanary is set to "auto_10_minutes", THE Buildspec SHALL pass `--wait 10m` to lad auto command
9. WHEN OptCanary is set to "auto_30_minutes", THE Buildspec SHALL pass `--wait 30m` to lad auto command
10. WHEN OptCanary is set to "auto_1_hour", THE Buildspec SHALL pass `--wait 1h` to lad auto command
11. WHEN OptCanary is set to "auto_4_hour", THE Buildspec SHALL pass `--wait 4h` to lad auto command
12. WHEN OptCanary is set to "auto_12_hour", THE Buildspec SHALL pass `--wait 12h` to lad auto command
13. WHEN OptCanary is set to "auto_1_day", THE Buildspec SHALL pass `--wait 24h` to lad auto command
14. IF OptLad is "canary" and OptCanary is not a valid canary strategy, THEN THE Buildspec SHALL log an error and skip lad operations
15. IF OptLad is "auto" and OptCanary is not a valid auto strategy, THEN THE Buildspec SHALL log an error and skip lad operations

### Requirement 3: lad 工具安装

**User Story:** As a DevOps engineer, I want lad tool to be automatically installed during the build process, so that I can use it without manual setup.

#### Acceptance Criteria

1. WHEN OptLad is not "none", THE Buildspec SHALL install lad tool using `go install github.com/aura-studio/lambda-alias-deployment@latest`
2. WHEN lad installation fails, THE Buildspec SHALL log an error and exit with non-zero status
3. THE Buildspec SHALL verify lad installation by running `lad --help`

### Requirement 4: lad 命令执行顺序

**User Story:** As a DevOps engineer, I want lad operations to execute after SAM deploy, so that the Lambda function is deployed before alias management.

#### Acceptance Criteria

1. THE Buildspec SHALL execute lad operations only after successful SAM deploy
2. WHEN SAM deploy fails, THE Buildspec SHALL skip lad operations
3. THE Buildspec SHALL pass `--env` parameter based on OptRuntime (test or prod)
4. THE Buildspec SHALL pass `--function` parameter based on the deployed Lambda function name

### Requirement 5: 错误处理和日志

**User Story:** As a DevOps engineer, I want clear error messages and logs, so that I can troubleshoot deployment issues.

#### Acceptance Criteria

1. WHEN lad command fails, THE Buildspec SHALL log the error message and exit with non-zero status
2. THE Buildspec SHALL log the lad command being executed before execution
3. THE Buildspec SHALL log the lad command result after execution
4. WHEN OptLad or OptCanary has invalid values, THE Buildspec SHALL log a descriptive error message
