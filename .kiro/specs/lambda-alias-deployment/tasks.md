# 实现计划：Lambda Alias Deployment (lad)

## 概述

本计划将 `lad` 命令行工具的实现分解为可执行的编码任务。该工具是 `deploy.sh` 脚本的 Go 语言直接翻译。

## 任务

- [x] 1. 初始化项目结构
  - [x] 1.1 创建项目目录和 go.mod
    - 创建 `lambda-alias-deployment/` 目录
    - 初始化 go.mod，模块路径为 `github.com/aura-studio/lambda-alias-deployment`
    - 添加依赖：cobra、aws-sdk-go-v2、go-toml/v2
    - _需求: 1.1, 1.2, 1.4, 1.5_
  - [x] 1.2 创建目录结构
    - 创建 cmd/、internal/aws/、internal/config/、internal/patcher/、internal/output/、internal/exitcode/ 目录
    - 创建 tests/cmd/、tests/internal/aws/、tests/internal/config/、tests/internal/patcher/ 测试目录
    - _需求: 1.1_

- [x] 2. 实现基础模块
  - [x] 2.1 实现退出码定义 (internal/exitcode/codes.go)
    - 定义 Success=0, ParamError=1, AWSError=2, ResourceNotFound=3, NetworkError=4
    - _需求: 12.1_
  - [x] 2.2 实现输出工具 (internal/output/printer.go)
    - 实现 Info、Error、Success、Warning、Separator 函数
    - Error 输出到 stderr，格式为"错误: {message}"
    - 其他输出到 stdout
    - _需求: 12.5, 12.6_
  - [x] 2.3 编写属性测试：输出流分离
    - 测试文件位置: tests/printer_test.go
    - **Property 10: 输出流分离**
    - **Validates: Requirements 12.5, 12.6**

- [x] 3. 实现配置模块
  - [x] 3.1 实现 SAMConfig 解析 (internal/config/samconfig.go)
    - 实现 LoadSAMConfig 加载 samconfig.toml
    - 实现 GetStackName、GetProfile 方法
    - 实现 GetFunctionName 根据 stack_name 和 env 生成函数名
    - _需求: 3.1, 3.2, 3.5, 13.2_
  - [x] 3.2 编写属性测试：SAMConfig 解析
    - 测试文件位置: tests/samconfig_test.go
    - **Property 2: SAMConfig 解析**
    - **Validates: Requirements 3.1, 3.2, 3.5, 13.2**

- [x] 4. 实现 AWS Lambda 客户端
  - [x] 4.1 实现 Lambda 客户端 (internal/aws/lambda.go)
    - 实现 NewClient 创建客户端
    - 实现 classifyError 错误分类函数
    - _需求: 12.2, 12.3, 12.4, 13.1_
  - [x] 4.2 实现 Lambda API 封装
    - 实现 CreateVersion 创建版本
    - 实现 GetAliasVersion 获取别名版本
    - 实现 UpdateAlias 更新别名
    - 实现 ConfigureCanary 配置灰度
    - 实现 CheckCanaryActive 检查灰度状态
    - 实现 VerifyVersionExists 验证版本存在
    - _需求: 4.1-4.6, 5.4-5.6, 6.1-6.5, 7.1-7.3, 8.3-8.6, 9.1-9.3_
  - [x] 4.3 编写属性测试：错误分类
    - 测试文件位置: tests/lambda_test.go
    - **Property 9: 错误分类和退出码**
    - **Validates: Requirements 12.1**

- [x] 5. 检查点 - 验证基础模块
  - 确保所有基础模块编译通过
  - 如有问题请询问用户

- [x] 6. 实现根命令和通用选项
  - [x] 6.1 实现根命令 (cmd/root.go)
    - 定义全局选项：--env、--profile、--function
    - 实现 ValidateEnv 验证环境参数
    - 实现 GetFunctionName 获取函数名（优先级：--function > samconfig）
    - 实现 GetProfile 获取 profile（优先级：--profile > samconfig）
    - _需求: 2.1-2.4, 3.3, 3.4, 13.3, 13.4_
  - [x] 6.2 实现程序入口 (main.go)
    - 调用 cmd.Execute()
    - _需求: 1.3_
  - [x] 6.3 编写属性测试：环境参数验证
    - 测试文件位置: tests/root_test.go
    - **Property 1: 环境参数验证**
    - **Validates: Requirements 2.1, 2.3**

- [x] 7. 实现灰度策略模块
  - [x] 7.1 实现灰度策略类型 (cmd/canary.go 或独立文件)
    - 定义 CanaryStrategy 类型和常量
    - 实现 Weight、IsValid、NextStrategy 方法
    - _需求: 5.2_
  - [x] 7.2 编写属性测试：灰度策略验证
    - 测试文件位置: tests/strategy_test.go
    - **Property 3: 灰度策略验证**
    - **Validates: Requirements 5.2, 5.3**

- [x] 8. 实现各命令
  - [x] 8.1 实现 deploy 命令 (cmd/deploy.go)
    - 检查是否有未完成的灰度
    - 执行 sam build 和 sam deploy
    - 创建新版本并更新 latest 别名
    - 显示部署结果和下一步提示
    - _需求: 4.1-4.7_
  - [x] 8.2 实现 canary 命令 (cmd/canary.go)
    - 验证 --strategy 参数
    - 获取 live 和 latest 版本
    - 配置灰度流量
    - 处理 --auto-promote 参数
    - 显示流量分配和下一步提示
    - _需求: 5.1-5.9_
  - [x] 8.3 编写属性测试：auto-promote 参数验证
    - 测试文件位置: tests/strategy_test.go
    - **Property 4: auto-promote 参数验证**
    - **Validates: Requirements 5.8**
  - [x] 8.4 实现 promote 命令 (cmd/promote.go)
    - 获取 live 和 latest 版本
    - 处理版本相同的情况
    - 更新 previous 和 live 别名
    - 处理 --skip-canary 参数
    - 显示版本变更信息
    - _需求: 6.1-6.7_
  - [x] 8.5 实现 rollback 命令 (cmd/rollback.go)
    - 获取 live 和 previous 版本
    - 处理版本相同的情况
    - 更新 live 别名
    - 记录回退日志
    - 处理 --reason 参数
    - 显示回退结果
    - _需求: 7.1-7.8_
  - [x] 8.6 编写属性测试：回退日志格式
    - 测试文件位置: tests/rollback_test.go
    - **Property 5: 回退日志格式**
    - **Validates: Requirements 7.4, 7.5**
  - [x] 8.7 实现 switch 命令 (cmd/switch.go)
    - 验证 --version 参数
    - 显示警告信息
    - 验证版本存在
    - 更新 live 别名（不更新 previous）
    - 显示注意事项
    - _需求: 8.1-8.8_
  - [x] 8.8 实现 status 命令 (cmd/status.go)
    - 获取三个别名版本
    - 检查灰度配置
    - 显示状态和可用操作
    - _需求: 9.1-9.5_

- [x] 9. 检查点 - 验证命令实现
  - 确保所有命令可以正确解析和执行
  - 如有问题请询问用户

- [x] 10. 实现模板补丁模块
  - [x] 10.1 实现 patch 逻辑 (internal/patcher/patch.go)
    - 实现 ValidateTemplate 验证模板
    - 实现 HasPatchMarker、HasAliasResources 检测函数
    - 实现 CheckFunctionExists、CheckDescriptionParam 检测函数
    - 实现 DetectHttpApis、DetectSchedules、DetectScheduleRoles 检测函数
    - 实现 GeneratePatchContent、GenerateDescriptionParam、GenerateHttpApiPatch 生成函数
    - 实现 BackupFile 备份函数
    - 实现 Patch 主函数
    - _需求: 10.2-10.16_
  - [x] 10.2 编写属性测试：模板验证
    - 测试文件位置: tests/patch_test.go
    - **Property 6: 模板验证**
    - **Validates: Requirements 10.2**
  - [x] 10.3 编写属性测试：补丁内容生成
    - 测试文件位置: tests/patch_test.go
    - **Property 7: 补丁内容生成**
    - **Validates: Requirements 10.8**
  - [x] 10.4 实现 unpatch 逻辑 (internal/patcher/unpatch.go)
    - 实现 RemovePatchMarkerContent 移除标记内容
    - 实现 RemoveAliasResources 移除资源
    - 实现 Unpatch 主函数
    - _需求: 11.2-11.8_
  - [x] 10.5 编写属性测试：移除补丁标记内容
    - 测试文件位置: tests/unpatch_test.go
    - **Property 8: 移除补丁标记内容**
    - **Validates: Requirements 11.3**

- [x] 11. 实现 patch 和 unpatch 命令
  - [x] 11.1 实现 patch 命令 (cmd/patch.go)
    - 解析 --template、--function、--dry-run 参数
    - 调用 patcher.Patch
    - _需求: 10.1, 10.13-10.16_
  - [x] 11.2 实现 unpatch 命令 (cmd/unpatch.go)
    - 解析 --template、--dry-run、--force 参数
    - 调用 patcher.Unpatch
    - _需求: 11.1, 11.4-11.7_

- [x] 12. 检查点 - 验证补丁功能
  - 确保 patch 和 unpatch 命令正常工作
  - 如有问题请询问用户

- [x] 13. 最终验证
  - [x] 13.1 编译验证
    - 执行 go build -o lad
    - 验证生成 lad 可执行文件
    - _需求: 1.3_
  - [x] 13.2 功能验证
    - 验证 --help 输出
    - 验证各命令的参数解析
    - _需求: 2.4_

- [x] 14. 最终检查点
  - 确保所有功能正常工作
  - 如有问题请询问用户

- [x] 15. 重构测试文件目录结构
  - [x] 15.1 创建 tests 目录
    - 创建 tests/ 目录（所有测试文件直接放在此目录下，不保留层级）
  - [x] 15.2 移动所有测试文件到 tests 目录
    - 移动 cmd/root_test.go 到 tests/root_test.go
    - 移动 cmd/rollback_test.go 到 tests/rollback_test.go
    - 移动 cmd/strategy_test.go 到 tests/strategy_test.go
    - 移动 internal/aws/lambda_test.go 到 tests/lambda_test.go
    - 移动 internal/config/samconfig_test.go 到 tests/samconfig_test.go
    - 移动 internal/patcher/patch_test.go 到 tests/patch_test.go
    - 移动 internal/patcher/unpatch_test.go 到 tests/unpatch_test.go
    - 所有测试文件使用 package tests 声明
    - 更新 import 路径引用源代码包
  - [x] 15.3 验证测试
    - 运行 go test ./tests 确保所有测试通过
    - 删除原位置的测试文件

## 备注

- 属性测试使用 rapid 库，每个测试运行至少 100 次迭代
- 集成测试需要 mock AWS API，可在后续迭代中添加
- 测试标签格式：**Feature: lambda-alias-deployment, Property N: {property_text}**
- 所有测试文件统一放置在 tests/ 目录下，不保留层级结构
- 运行测试命令：`go test ./tests`
