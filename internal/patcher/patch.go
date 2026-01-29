// Package patcher provides template patching utilities for the lad command line tool.
package patcher

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
)

const (
	// PatchStartMarker is the start marker for patch content
	PatchStartMarker = "# >>>>>> DEPLOY_SCRIPT_PATCH_START <<<<<<"
	// PatchEndMarker is the end marker for patch content
	PatchEndMarker = "# >>>>>> DEPLOY_SCRIPT_PATCH_END <<<<<<"
	// LineModifyMarker is the marker for modified lines (original line preserved as comment)
	LineModifyMarker = "# <<< LAD_ORIGINAL:"
	// LineModifyEndMarker is the end marker for modified lines
	LineModifyEndMarker = ">>>"
)

// PatchOptions patch 命令选项
type PatchOptions struct {
	TemplatePath string // 模板文件路径，默认 template.yaml
	FunctionName string // 函数资源名称，默认 Function
	DryRun       bool   // 仅预览，不实际修改
}

// PatchResult contains the result of a patch operation
type PatchResult struct {
	BackupPath string
	ExitCode   int
}

// Patch 执行补丁操作
func Patch(opts PatchOptions) *PatchResult {
	result := &PatchResult{ExitCode: exitcode.Success}

	output.Info("执行 patch 命令")
	output.Info("模板文件: %s", opts.TemplatePath)
	output.Info("函数资源: %s", opts.FunctionName)
	if opts.DryRun {
		output.Info("模式: dry-run (仅预览)")
	}
	fmt.Println()

	// 1. 验证模板文件
	if err := ValidateTemplate(opts.TemplatePath); err != nil {
		output.Error("%s", err.Error())
		result.ExitCode = exitcode.ParamError
		return result
	}
	output.Success("模板文件有效")

	// 读取模板内容
	content, err := os.ReadFile(opts.TemplatePath)
	if err != nil {
		output.Error("读取模板文件失败: %s", err.Error())
		result.ExitCode = exitcode.ParamError
		return result
	}
	contentStr := string(content)

	// 2. 检查是否已打补丁（通过标记）
	if HasPatchMarker(contentStr) {
		output.Error("模板文件已包含补丁标记")
		output.Info("如需重新打补丁，请先执行: lad unpatch --template %s", opts.TemplatePath)
		result.ExitCode = exitcode.ParamError
		return result
	}

	// 3. 检查是否已存在版本/别名资源（即使没有标记）
	if HasAliasResources(contentStr) {
		existingResources := GetExistingAliasResources(contentStr)
		fmt.Println()
		output.Warning("检测到模板中已存在以下版本/别名资源:")
		for _, res := range existingResources {
			output.Info("  - %s", res)
		}
		fmt.Println()
		output.Info("选项:")
		output.Info("  1. 使用 'lad unpatch --template %s' 移除现有资源后重新打补丁", opts.TemplatePath)
		output.Info("  2. 手动检查并决定是否需要打补丁")
		result.ExitCode = exitcode.ParamError
		return result
	}
	output.Success("模板文件未包含版本/别名资源")

	// 4. 检查函数资源是否存在
	if !CheckFunctionExists(contentStr, opts.FunctionName) {
		output.Error("未找到函数资源: %s", opts.FunctionName)
		output.Info("请使用 --function 参数指定正确的函数资源名称")
		output.Info("提示: 查看 template.yaml 中 Type: AWS::Serverless::Function 的资源名称")
		result.ExitCode = exitcode.ParamError
		return result
	}
	output.Success("找到函数资源: %s", opts.FunctionName)

	// 5. 检查 Description 参数
	needDescriptionParam := !CheckDescriptionParam(contentStr)
	if needDescriptionParam {
		output.Warning("模板缺少 Description 参数，将自动添加")
	}

	// 6. 检测触发器资源
	httpApis := DetectHttpApis(contentStr)
	schedules := DetectSchedules(contentStr)
	scheduleRoles := DetectScheduleRoles(contentStr)

	hasTriggers := len(httpApis) > 0 || len(schedules) > 0
	if hasTriggers {
		fmt.Println()
		output.Info("检测到以下触发器资源:")
		for _, api := range httpApis {
			output.Info("  - HttpApi: %s", api)
		}
		for _, sch := range schedules {
			output.Info("  - Schedule: %s", sch)
		}
	}

	// 7. 生成补丁内容（用于显示）
	patchContent := GeneratePatchContent(opts.FunctionName)

	// 如果有 HttpApi，添加相关资源到显示内容
	var httpApiContent string
	if len(httpApis) > 0 {
		httpApiContent = GenerateHttpApiPatch(opts.FunctionName, httpApis[0])
	}

	// 8. 显示补丁内容
	fmt.Println()
	output.Separator()
	output.Info("将添加以下内容到 Resources 部分末尾:")
	output.Separator()
	fmt.Println(patchContent)
	if httpApiContent != "" {
		fmt.Println(httpApiContent)
	}
	fmt.Println(PatchEndMarker)

	if needDescriptionParam {
		fmt.Println()
		output.Separator()
		output.Info("将添加以下内容到 Parameters 部分:")
		output.Separator()
		fmt.Println(GenerateDescriptionParam())
	}

	// 显示将要修改的触发器
	if hasTriggers {
		fmt.Println()
		output.Separator()
		output.Info("将修改以下触发器指向 LiveAlias:")
		output.Separator()
		for _, sch := range schedules {
			output.Info("  - %s: Target.Arn -> !Ref LiveAlias", sch)
		}
		for _, role := range scheduleRoles {
			output.Info("  - %s: Resource -> ${%s.Arn}:live", role, opts.FunctionName)
		}
	}

	// 9. 如果是 dry-run，到此结束
	if opts.DryRun {
		fmt.Println()
		output.Separator()
		output.Info("Dry-run 完成，未修改任何文件")
		output.Separator()
		return result
	}

	// 10. 备份原文件
	backupPath, err := BackupFile(opts.TemplatePath)
	if err != nil {
		output.Error("备份文件失败: %s", err.Error())
		result.ExitCode = exitcode.ParamError
		return result
	}
	result.BackupPath = backupPath
	output.Success("已备份原文件到: %s", backupPath)

	// 11. 添加 Description 参数（如果需要）
	if needDescriptionParam {
		contentStr = addDescriptionParam(contentStr)
		output.Success("已添加 Description 参数")
	}

	// 12. 修改触发器配置
	if len(schedules) > 0 {
		contentStr = patchSchedules(contentStr, opts.FunctionName)
		output.Success("已修改 Schedule 触发器指向 LiveAlias")
	}

	if len(scheduleRoles) > 0 {
		contentStr = patchIAMRoles(contentStr, opts.FunctionName)
		output.Success("已修改 IAM Role 权限指向 :live 别名")
	}

	// 13. 在文件末尾添加补丁内容（包括 HttpApi 资源和结束标记）
	contentStr += patchContent
	if httpApiContent != "" {
		contentStr += httpApiContent
	}
	contentStr += "\n" + PatchEndMarker + "\n"

	// 写入文件
	if err := os.WriteFile(opts.TemplatePath, []byte(contentStr), 0644); err != nil {
		output.Error("写入模板文件失败: %s", err.Error())
		result.ExitCode = exitcode.ParamError
		return result
	}
	output.Success("已添加版本和别名资源")

	// 14. 输出结果
	fmt.Println()
	output.Separator()
	output.Info("补丁应用完成!")
	output.Separator()
	output.Info("模板文件: %s", opts.TemplatePath)
	output.Info("备份文件: %s", backupPath)
	fmt.Println()
	output.Info("已添加资源:")
	output.Info("  - %sVersion (Lambda 版本)", opts.FunctionName)
	output.Info("  - LiveAlias (生产流量别名)")
	output.Info("  - PreviousAlias (回退版本别名)")
	output.Info("  - LatestAlias (测试版本别名)")
	if len(httpApis) > 0 {
		output.Info("  - LiveAliasHttpApiPermission (HttpApi 调用权限)")
		output.Info("  - HttpApiLiveRoute (HttpApi 路由)")
		output.Info("  - HttpApiLiveIntegration (HttpApi 集成)")
	}
	fmt.Println()
	if hasTriggers {
		output.Info("已修改触发器:")
		if len(schedules) > 0 {
			output.Info("  - Schedule 资源已指向 LiveAlias")
		}
		if len(scheduleRoles) > 0 {
			output.Info("  - IAM Role 权限已指向 :live 别名")
		}
		fmt.Println()
	}
	output.Info("下一步:")
	output.Info("  1. 检查 template.yaml 确认补丁内容正确")
	output.Info("  2. 执行 'lad deploy --env test' 部署测试环境")
	fmt.Println()
	output.Info("如需移除补丁: lad unpatch --template %s", opts.TemplatePath)

	return result
}

// ValidateTemplate 验证模板文件
// 检查: 文件存在、包含 AWS::Serverless、包含 Resources
func ValidateTemplate(path string) error {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", path)
	}

	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("无法读取文件: %s", err.Error())
	}

	contentStr := string(content)

	// 检查是否是 SAM 模板
	if !strings.Contains(contentStr, "AWS::Serverless") {
		return fmt.Errorf("不是有效的 SAM 模板文件: %s", path)
	}

	// 检查是否有 Resources 部分
	resourcesPattern := regexp.MustCompile(`(?m)^Resources:`)
	if !resourcesPattern.MatchString(contentStr) {
		return fmt.Errorf("模板文件缺少 Resources 部分: %s", path)
	}

	return nil
}

// HasPatchMarker 检查是否存在补丁标记
func HasPatchMarker(content string) bool {
	return strings.Contains(content, PatchStartMarker)
}

// HasAliasResources 检查是否存在版本/别名资源
func HasAliasResources(content string) bool {
	// 检查是否存在 LiveAlias 资源
	liveAliasPattern := regexp.MustCompile(`(?m)^  LiveAlias:`)
	if liveAliasPattern.MatchString(content) {
		return true
	}

	// 检查是否存在 AWS::Lambda::Alias 类型
	if strings.Contains(content, "Type: AWS::Lambda::Alias") {
		return true
	}

	// 检查是否存在 AWS::Lambda::Version 类型
	if strings.Contains(content, "Type: AWS::Lambda::Version") {
		return true
	}

	return false
}

// GetExistingAliasResources 获取已存在的别名资源列表
func GetExistingAliasResources(content string) []string {
	var resources []string

	// 查找所有 AWS::Lambda::Version 资源
	versionPattern := regexp.MustCompile(`(?m)^  ([A-Za-z0-9]+):\s*\n\s+Type:\s*AWS::Lambda::Version`)
	versionMatches := versionPattern.FindAllStringSubmatch(content, -1)
	for _, match := range versionMatches {
		if len(match) > 1 {
			resources = append(resources, match[1])
		}
	}

	// 查找所有 AWS::Lambda::Alias 资源
	aliasPattern := regexp.MustCompile(`(?m)^  ([A-Za-z0-9]+):\s*\n\s+Type:\s*AWS::Lambda::Alias`)
	aliasMatches := aliasPattern.FindAllStringSubmatch(content, -1)
	for _, match := range aliasMatches {
		if len(match) > 1 {
			resources = append(resources, match[1])
		}
	}

	return resources
}

// CheckFunctionExists 检查函数资源是否存在
func CheckFunctionExists(content, functionName string) bool {
	// 检查是否存在指定的函数资源
	// 匹配格式: "  FunctionName:" 后面跟着 "Type:.*Function"
	pattern := regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(functionName) + `:\s*\n[\s\S]*?Type:\s*.*Function`)
	return pattern.MatchString(content)
}

// CheckDescriptionParam 检查 Description 参数是否存在
func CheckDescriptionParam(content string) bool {
	// 检查 Parameters 部分是否存在 Description 参数
	pattern := regexp.MustCompile(`(?m)^  Description:`)
	return pattern.MatchString(content)
}

// DetectHttpApis 检测 HttpApi 资源
func DetectHttpApis(content string) []string {
	var apis []string
	pattern := regexp.MustCompile(`(?m)^  ([A-Za-z0-9]+):\s*\n\s+Type:\s*AWS::Serverless::HttpApi`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			apis = append(apis, match[1])
		}
	}
	return apis
}

// DetectSchedules 检测 Schedule 资源
func DetectSchedules(content string) []string {
	var schedules []string
	pattern := regexp.MustCompile(`(?m)^  ([A-Za-z0-9]+):\s*\n\s+Type:\s*AWS::Scheduler::Schedule`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			schedules = append(schedules, match[1])
		}
	}
	return schedules
}

// DetectScheduleRoles 检测 Schedule 相关的 IAM Role
func DetectScheduleRoles(content string) []string {
	var roles []string
	// 查找包含 lambda:InvokeFunction 的 IAM Role
	// 首先找到所有 IAM Role 资源
	rolePattern := regexp.MustCompile(`(?m)^  ([A-Za-z0-9]+):\s*\n\s+Type:\s*AWS::IAM::Role`)
	roleMatches := rolePattern.FindAllStringSubmatch(content, -1)

	for _, match := range roleMatches {
		if len(match) > 1 {
			roleName := match[1]
			// 检查该 Role 是否包含 lambda:InvokeFunction 权限
			// 找到该 Role 的定义范围
			roleDefPattern := regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(roleName) + `:\s*\n[\s\S]*?(?:^  [A-Za-z0-9]+:|$)`)
			roleDefMatch := roleDefPattern.FindString(content)
			if strings.Contains(roleDefMatch, "lambda:InvokeFunction") {
				roles = append(roles, roleName)
			}
		}
	}
	return roles
}

// GeneratePatchContent 生成补丁内容
func GeneratePatchContent(functionName string) string {
	return fmt.Sprintf(`
%s
# 以下资源由 lad patch 命令自动生成
# 请勿手动修改此区域内容
# 移除请使用: lad unpatch

  # Lambda Version - 版本发布配置
  %sVersion:
    Type: AWS::Lambda::Version
    Properties:
      FunctionName: !Ref %s
      Description: !Ref Description

  # Live 别名 - 生产流量
  LiveAlias:
    Type: AWS::Lambda::Alias
    Properties:
      FunctionName: !Ref %s
      FunctionVersion: !GetAtt %sVersion.Version
      Name: live

  # Previous 别名 - 回退版本
  PreviousAlias:
    Type: AWS::Lambda::Alias
    Properties:
      FunctionName: !Ref %s
      FunctionVersion: !GetAtt %sVersion.Version
      Name: previous

  # Latest 别名 - 测试版本
  LatestAlias:
    Type: AWS::Lambda::Alias
    Properties:
      FunctionName: !Ref %s
      FunctionVersion: !GetAtt %sVersion.Version
      Name: latest
`, PatchStartMarker, functionName, functionName, functionName, functionName, functionName, functionName, functionName, functionName)
}

// GenerateDescriptionParam 生成 Description 参数
func GenerateDescriptionParam() string {
	return `  Description:
    Type: String
    Default: Serverless Function Description
    Description: Description for the Lambda function`
}

// GenerateHttpApiPatch 生成 HttpApi 相关补丁
func GenerateHttpApiPatch(functionName, apiName string) string {
	return fmt.Sprintf(`
  # LiveAlias 的 HttpApi 调用权限
  LiveAliasHttpApiPermission:
    Type: AWS::Lambda::Permission
    Properties:
      FunctionName: !Ref LiveAlias
      Action: lambda:InvokeFunction
      Principal: apigateway.amazonaws.com
      SourceArn: !Sub "arn:aws:execute-api:${AWS::Region}:${AWS::AccountId}:%s/*"

  # HttpApi 路由到 LiveAlias
  HttpApiLiveRoute:
    Type: AWS::ApiGatewayV2::Route
    Properties:
      ApiId: !Ref %s
      RouteKey: "$default"
      Target: !Sub "integrations/${HttpApiLiveIntegration}"

  # HttpApi 集成到 LiveAlias
  HttpApiLiveIntegration:
    Type: AWS::ApiGatewayV2::Integration
    Properties:
      ApiId: !Ref %s
      IntegrationType: AWS_PROXY
      IntegrationUri: !Ref LiveAlias
      PayloadFormatVersion: "2.0"
`, apiName, apiName, apiName)
}

// BackupFile 备份文件，返回备份文件路径
// 格式: {path}.bak.{timestamp}
func BackupFile(path string) (string, error) {
	timestamp := time.Now().Format("20060102150405")
	backupPath := fmt.Sprintf("%s.bak.%s", path, timestamp)

	// 打开源文件
	src, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return backupPath, nil
}

// addDescriptionParam 在 Parameters 部分添加 Description 参数
func addDescriptionParam(content string) string {
	descParam := GenerateDescriptionParam()

	// 在 Parameters: 后面添加 Description 参数
	parametersPattern := regexp.MustCompile(`(?m)^Parameters:\s*\n`)
	if parametersPattern.MatchString(content) {
		return parametersPattern.ReplaceAllString(content, "Parameters:\n"+descParam+"\n")
	}

	// 如果没有 Parameters 部分，在 Resources 前添加
	resourcesPattern := regexp.MustCompile(`(?m)^Resources:`)
	if resourcesPattern.MatchString(content) {
		return resourcesPattern.ReplaceAllString(content, "Parameters:\n"+descParam+"\n\nResources:")
	}

	return content
}

// patchSchedules 修改 Schedule 资源指向 LiveAlias
// 保留原始行作为注释，便于 unpatch 时还原
func patchSchedules(content, functionName string) string {
	// 只替换 Target 下的 Arn（不是 RoleArn）
	// 匹配: Arn: !GetAtt FunctionName.Arn
	pattern := regexp.MustCompile(`(?m)^(\s+)(Arn: !GetAtt [A-Za-z0-9]+\.Arn)`)
	return pattern.ReplaceAllStringFunc(content, func(match string) string {
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		indent := submatches[1]
		originalValue := submatches[2]
		// 格式: 注释保留原始行 + 新行
		return fmt.Sprintf("%s%s %s %s\n%sArn: !Ref LiveAlias", indent, LineModifyMarker, originalValue, LineModifyEndMarker, indent)
	})
}

// patchIAMRoles 修改 IAM Role 资源的 Lambda 权限
// 保留原始行作为注释，便于 unpatch 时还原
func patchIAMRoles(content, functionName string) string {
	// 修改 Resource: !GetAtt Function.Arn 改为 !Sub "${Function.Arn}:live"
	pattern1 := regexp.MustCompile(`(?m)^(\s+)(Resource: !GetAtt ` + regexp.QuoteMeta(functionName) + `\.Arn)`)
	content = pattern1.ReplaceAllStringFunc(content, func(match string) string {
		submatches := pattern1.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		indent := submatches[1]
		originalValue := submatches[2]
		newValue := fmt.Sprintf(`Resource: !Sub "${%s.Arn}:live"`, functionName)
		return fmt.Sprintf("%s%s %s %s\n%s%s", indent, LineModifyMarker, originalValue, LineModifyEndMarker, indent, newValue)
	})

	// 如果已经是 !Sub 格式但没有 :live，添加 :live
	pattern2 := regexp.MustCompile(`(?m)^(\s+)(Resource: !Sub "\${` + regexp.QuoteMeta(functionName) + `\.Arn}")`)
	content = pattern2.ReplaceAllStringFunc(content, func(match string) string {
		submatches := pattern2.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		indent := submatches[1]
		originalValue := submatches[2]
		// 检查是否已经有 :live
		if strings.Contains(originalValue, ":live") {
			return match
		}
		newValue := fmt.Sprintf(`Resource: !Sub "${%s.Arn}:live"`, functionName)
		return fmt.Sprintf("%s%s %s %s\n%s%s", indent, LineModifyMarker, originalValue, LineModifyEndMarker, indent, newValue)
	})

	return content
}
