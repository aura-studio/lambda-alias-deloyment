# 需求文档

## 简介

本项目旨在将现有的 `deploy.sh` 脚本用 Go 语言直接翻译重新实现，创建一个名为 `lad` (Lambda Alias Deployment) 的独立命令行工具。该工具用于管理 AWS Lambda 函数的版本和别名，支持灰度发布、回退和版本切换等功能。新项目位于 `lambda-alias-deployment/` 目录下。

## 术语表

- **LAD**: Lambda Alias Deployment，本项目的命令行工具名称
- **Lambda_Version**: AWS Lambda 函数的不可变版本快照
- **Lambda_Alias**: 指向特定 Lambda 版本的指针，支持流量路由配置
- **Live_Alias**: 生产环境流量指向的别名
- **Previous_Alias**: 保存上一个稳定版本的别名，用于回退
- **Latest_Alias**: 最新部署版本的别名
- **Canary_Deployment**: 灰度发布，将部分流量路由到新版本进行验证
- **Routing_Config**: Lambda 别名的流量路由配置，用于灰度发布
- **SAM**: AWS Serverless Application Model，用于定义无服务器应用的框架
- **Template_File**: SAM 模板文件（template.yaml）
- **SAMConfig**: SAM 配置文件（samconfig.toml），包含部署参数

## 需求

### 需求 1: 项目结构

**用户故事:** 作为开发者，我希望有一个独立的 Go 项目，以便于维护和分发。

#### 验收标准

1. THE LAD 项目 SHALL 位于 `lambda-alias-deployment/` 目录下
2. THE LAD 项目 SHALL 使用独立的 go.mod 文件，模块路径为 `github.com/aura-studio/lambda-alias-deployment`
3. THE LAD 项目 SHALL 编译输出名为 `lad` 的可执行文件
4. THE LAD 项目 SHALL 使用 cobra 库实现命令行接口
5. THE LAD 项目 SHALL 使用 AWS SDK for Go v2 访问 Lambda 服务

### 需求 2: 通用选项

**用户故事:** 作为运维人员，我希望能够指定目标环境和查看帮助信息，以便于在不同环境中操作。

#### 验收标准

1. WHEN 用户指定 `--env` 选项 THEN THE LAD SHALL 使用指定的环境值（test 或 prod）
2. WHEN 用户未指定 `--env` 选项 THEN THE LAD SHALL 默认使用 test 环境
3. IF 用户指定无效的环境值 THEN THE LAD SHALL 返回参数错误并显示有效值列表
4. WHEN 用户指定 `--help` 选项 THEN THE LAD SHALL 显示相应命令的帮助信息

### 需求 3: 函数名检测

**用户故事:** 作为运维人员，我希望工具能够自动检测 Lambda 函数名，以便于在不同项目中使用。

#### 验收标准

1. THE LAD SHALL 从 samconfig.toml 文件中读取 stack_name 配置
2. THE LAD SHALL 根据 stack_name 和环境参数动态生成 Lambda 函数名
3. THE LAD SHALL 支持通过 `--function` 选项手动指定函数名
4. IF samconfig.toml 不存在或无法解析 THEN THE LAD SHALL 要求用户通过 `--function` 指定函数名
5. THE LAD SHALL 从 samconfig.toml 中读取 AWS profile 配置

### 需求 4: Deploy 命令

**用户故事:** 作为开发者，我希望能够部署新版本并更新 latest 别名，以便于准备灰度发布。

#### 验收标准

1. WHEN 执行 deploy 命令 THEN THE LAD SHALL 检查是否存在未完成的灰度发布
2. IF 存在未完成的灰度发布 THEN THE LAD SHALL 返回错误并提示先执行 promote 或 rollback
3. WHEN 灰度检查通过 THEN THE LAD SHALL 执行 SAM build 命令
4. WHEN SAM build 成功 THEN THE LAD SHALL 执行 SAM deploy 命令，传递 Runtime 和 Description 参数
5. WHEN SAM 部署成功 THEN THE LAD SHALL 创建新的 Lambda 版本，描述包含部署时间
6. WHEN 新版本创建成功 THEN THE LAD SHALL 更新 latest 别名指向新版本
7. WHEN deploy 命令成功完成 THEN THE LAD SHALL 显示部署结果和下一步操作提示

### 需求 5: Canary 命令

**用户故事:** 作为运维人员，我希望能够进行灰度发布，以便于安全地验证新版本。

#### 验收标准

1. WHEN 执行 canary 命令 THEN THE LAD SHALL 要求指定 `--strategy` 参数
2. THE LAD SHALL 支持以下灰度策略: canary10 (10%), canary25 (25%), canary50 (50%), canary75 (75%)
3. IF 指定无效的策略 THEN THE LAD SHALL 返回参数错误并显示有效策略列表
4. WHEN 执行 canary 命令 THEN THE LAD SHALL 获取 live 和 latest 别名的版本
5. IF live 和 latest 指向同一版本 THEN THE LAD SHALL 返回错误提示先执行 deploy
6. WHEN 版本检查通过 THEN THE LAD SHALL 配置 live 别名的流量路由，主版本为 live 当前版本，灰度版本为 latest 版本
7. WHEN 指定 `--auto-promote` 且策略为 canary75 THEN THE LAD SHALL 自动执行 promote
8. IF 指定 `--auto-promote` 但策略不是 canary75 THEN THE LAD SHALL 返回参数错误
9. WHEN canary 命令成功完成 THEN THE LAD SHALL 显示流量分配比例和下一步操作提示

### 需求 6: Promote 命令

**用户故事:** 作为运维人员，我希望能够完成灰度发布，将流量完全切换到新版本。

#### 验收标准

1. WHEN 执行 promote 命令 THEN THE LAD SHALL 获取 live 和 latest 别名的版本
2. IF live 和 latest 指向同一版本 THEN THE LAD SHALL 显示已是最新版本并正常退出
3. WHEN 未指定 `--skip-canary` 且没有活跃灰度 THEN THE LAD SHALL 显示警告但继续执行
4. WHEN 版本不同时 THEN THE LAD SHALL 更新 previous 别名指向原 live 版本
5. WHEN previous 更新成功 THEN THE LAD SHALL 更新 live 别名指向 latest 版本并清除灰度配置
6. WHEN 指定 `--skip-canary` 选项 THEN THE LAD SHALL 跳过灰度状态检查
7. WHEN promote 命令成功完成 THEN THE LAD SHALL 显示版本变更信息

### 需求 7: Rollback 命令

**用户故事:** 作为运维人员，我希望能够紧急回退到上一个稳定版本，以便于快速恢复服务。

#### 验收标准

1. WHEN 执行 rollback 命令 THEN THE LAD SHALL 获取 live 和 previous 别名的版本
2. IF live 和 previous 指向同一版本 THEN THE LAD SHALL 显示无需回退并正常退出
3. WHEN 版本不同时 THEN THE LAD SHALL 更新 live 别名指向 previous 版本并清除灰度配置
4. WHEN rollback 成功 THEN THE LAD SHALL 记录回退日志到脚本目录下的 rollback.log 文件
5. THE 回退日志 SHALL 包含时间戳、环境、原版本、目标版本、原因和操作人（从 USER 环境变量获取）
6. WHEN 用户指定 `--reason` 选项 THEN THE LAD SHALL 在日志中记录回退原因
7. IF 未指定 `--reason` THEN THE LAD SHALL 使用"未指定原因"作为默认值
8. WHEN rollback 命令成功完成 THEN THE LAD SHALL 显示回退结果和下一步操作提示

### 需求 8: Switch 命令

**用户故事:** 作为运维人员，我希望能够在极端情况下切换到指定版本，以便于处理紧急问题。

#### 验收标准

1. WHEN 执行 switch 命令 THEN THE LAD SHALL 要求指定 `--version` 参数
2. WHEN 执行 switch 命令 THEN THE LAD SHALL 显示警告信息，说明此操作绕过正常发布流程
3. WHEN 执行 switch 命令 THEN THE LAD SHALL 验证指定版本是否存在
4. IF 指定版本不存在 THEN THE LAD SHALL 返回资源不存在错误
5. IF live 已指向目标版本 THEN THE LAD SHALL 显示无需切换并正常退出
6. WHEN 版本验证通过 THEN THE LAD SHALL 更新 live 别名指向指定版本并清除灰度配置
7. THE Switch 命令 SHALL NOT 更新 previous 别名
8. WHEN switch 命令成功完成 THEN THE LAD SHALL 显示注意事项（绕过正常流程、previous 未更新等）

### 需求 9: Status 命令

**用户故事:** 作为运维人员，我希望能够查看当前别名状态，以便于了解系统状态。

#### 验收标准

1. WHEN 执行 status 命令 THEN THE LAD SHALL 显示 live、previous、latest 三个别名的版本
2. IF 获取别名版本失败 THEN THE LAD SHALL 显示"未配置"
3. WHEN 存在活跃的灰度配置 THEN THE LAD SHALL 显示灰度状态和流量分配比例
4. WHEN 不存在灰度配置且 live 等于 latest THEN THE LAD SHALL 提示系统处于稳定状态
5. WHEN 不存在灰度配置且 live 不等于 latest THEN THE LAD SHALL 提示有新版本待发布

### 需求 10: Patch 命令

**用户故事:** 作为开发者，我希望能够自动给 template.yaml 添加版本和别名资源，以便于快速配置项目。

#### 验收标准

1. THE Patch 命令 SHALL NOT 需要 `--env` 参数
2. WHEN 执行 patch 命令 THEN THE LAD SHALL 验证 template.yaml 是否有效（存在且包含 AWS::Serverless 和 Resources）
3. IF 模板文件不存在 THEN THE LAD SHALL 返回参数错误
4. IF 模板不是有效的 SAM 模板 THEN THE LAD SHALL 返回参数错误
5. IF 模板已包含补丁标记 THEN THE LAD SHALL 返回错误提示先执行 unpatch
6. IF 模板已存在版本/别名资源但无标记 THEN THE LAD SHALL 返回错误并列出现有资源
7. WHEN 验证通过 THEN THE LAD SHALL 检查函数资源是否存在
8. WHEN 函数资源存在 THEN THE LAD SHALL 添加 Lambda Version 和三个 Alias 资源（live、previous、latest）
9. WHEN 模板缺少 Description 参数 THEN THE LAD SHALL 自动添加该参数
10. WHEN 检测到 HttpApi 资源 THEN THE LAD SHALL 添加相应的权限、路由和集成配置
11. WHEN 检测到 Schedule 资源 THEN THE LAD SHALL 修改其 Target.Arn 指向 LiveAlias
12. WHEN 检测到 Schedule 相关的 IAM Role THEN THE LAD SHALL 修改其 Resource 指向函数的 :live 别名
13. WHEN 指定 `--dry-run` 选项 THEN THE LAD SHALL 仅显示将要添加的内容而不实际修改
14. WHEN 指定 `--template` 选项 THEN THE LAD SHALL 使用指定的模板文件路径，默认为 template.yaml
15. WHEN 指定 `--function` 选项 THEN THE LAD SHALL 使用指定的函数资源名称，默认为 Function
16. WHEN patch 成功 THEN THE LAD SHALL 创建原文件的备份（格式: template.yaml.bak.时间戳）

### 需求 11: Unpatch 命令

**用户故事:** 作为开发者，我希望能够移除 template.yaml 中的补丁内容，以便于恢复原始状态。

#### 验收标准

1. THE Unpatch 命令 SHALL NOT 需要 `--env` 参数
2. WHEN 执行 unpatch 命令 THEN THE LAD SHALL 检查模板是否包含补丁标记
3. IF 模板包含补丁标记 THEN THE LAD SHALL 移除标记之间的所有内容
4. IF 模板不包含补丁标记但存在版本/别名资源 THEN THE LAD SHALL 要求 `--force` 确认
5. WHEN 指定 `--force` 选项 THEN THE LAD SHALL 移除所有版本/别名资源（即使无标记）
6. WHEN 指定 `--dry-run` 选项 THEN THE LAD SHALL 仅显示将要移除的内容而不实际修改
7. WHEN unpatch 成功 THEN THE LAD SHALL 创建原文件的备份
8. IF 模板不包含任何补丁内容和版本/别名资源 THEN THE LAD SHALL 显示无需移除并正常退出

### 需求 12: 错误处理

**用户故事:** 作为用户，我希望工具能够提供清晰的错误信息，以便于排查问题。

#### 验收标准

1. THE LAD SHALL 定义以下退出码: 0=成功, 1=参数错误, 2=AWS错误, 3=资源不存在, 4=网络错误
2. IF 发生网络错误（包含 unable to locate credentials、could not connect、connection refused、network、timeout、timed out、unreachable 关键词）THEN THE LAD SHALL 显示网络错误信息并建议检查网络连接
3. IF 发生资源不存在错误（包含 ResourceNotFoundException、does not exist、not found、cannot find 关键词）THEN THE LAD SHALL 显示资源不存在信息并建议执行初始化
4. IF 发生其他 AWS 错误 THEN THE LAD SHALL 显示 AWS 错误详情
5. THE LAD SHALL 将错误信息输出到 stderr，格式为"错误: {message}"
6. THE LAD SHALL 将正常信息输出到 stdout

### 需求 13: AWS 配置

**用户故事:** 作为运维人员，我希望工具能够正确使用 AWS 配置，以便于访问 Lambda 服务。

#### 验收标准

1. THE LAD SHALL 使用 AWS SDK for Go v2 访问 Lambda 服务
2. THE LAD SHALL 从 samconfig.toml 中读取 profile 配置
3. THE LAD SHALL 支持通过 `--profile` 选项覆盖 samconfig.toml 中的配置
4. IF 未指定 profile 且 samconfig.toml 中无配置 THEN THE LAD SHALL 使用默认的 AWS 配置
5. THE LAD SHALL 正确处理 AWS 凭证错误

