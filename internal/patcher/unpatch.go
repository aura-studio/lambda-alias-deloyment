// Package patcher provides template patching utilities for the lad command line tool.
package patcher

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
)

// UnpatchOptions unpatch 命令选项
type UnpatchOptions struct {
	TemplatePath string // 模板文件路径，默认 template.yaml
	DryRun       bool   // 仅预览，不实际修改
	Force        bool   // 强制移除（即使无标记）
	NoBackup     bool   // 不创建备份文件
}

// UnpatchResult contains the result of an unpatch operation
type UnpatchResult struct {
	BackupPath string
	ExitCode   int
}

// Unpatch 执行移除补丁操作
func Unpatch(opts UnpatchOptions) *UnpatchResult {
	result := &UnpatchResult{ExitCode: exitcode.Success}

	output.Info("执行 unpatch 命令")
	output.Info("模板文件: %s", opts.TemplatePath)
	if opts.DryRun {
		output.Info("模式: dry-run (仅预览)")
	}
	if opts.Force {
		output.Info("模式: force (强制移除)")
	}
	fmt.Println()

	// 1. 验证模板文件存在
	if _, err := os.Stat(opts.TemplatePath); os.IsNotExist(err) {
		output.Error("文件不存在: %s", opts.TemplatePath)
		result.ExitCode = exitcode.ParamError
		return result
	}

	// 读取模板内容
	content, err := os.ReadFile(opts.TemplatePath)
	if err != nil {
		output.Error("读取模板文件失败: %s", err.Error())
		result.ExitCode = exitcode.ParamError
		return result
	}
	contentStr := string(content)

	hasPatchMarker := HasPatchMarker(contentStr)
	hasAliasResources := HasAliasResources(contentStr)
	hasLineModifications := HasLineModifications(contentStr)

	// 2. 检查是否有内容需要移除
	if !hasPatchMarker && !hasAliasResources && !hasLineModifications {
		output.Success("模板文件不包含任何补丁内容或版本/别名资源，无需移除")
		return result
	}

	// 3. 如果没有补丁标记但有别名资源，需要 --force
	if !hasPatchMarker && hasAliasResources && !hasLineModifications {
		existingResources := GetExistingAliasResources(contentStr)
		fmt.Println()
		output.Warning("检测到模板中存在版本/别名资源，但没有补丁标记:")
		for _, res := range existingResources {
			output.Info("  - %s", res)
		}
		fmt.Println()

		if !opts.Force {
			output.Error("需要 --force 参数才能移除没有标记的资源")
			output.Info("使用: lad unpatch --template %s --force", opts.TemplatePath)
			result.ExitCode = exitcode.ParamError
			return result
		}
		output.Warning("使用 --force 模式移除资源")
	}

	// 4. 显示将要移除的内容
	fmt.Println()
	output.Separator()
	output.Info("将移除以下内容:")
	output.Separator()

	var newContent string = contentStr

	// 4.1 还原行级别的修改
	if hasLineModifications {
		modifiedLines := GetModifiedLines(contentStr)
		output.Info("- 还原 %d 处行级别修改:", len(modifiedLines))
		for _, line := range modifiedLines {
			output.Info("    %s", line)
		}
		newContent = RestoreModifiedLines(newContent)
	}

	// 4.2 移除补丁标记内容
	if hasPatchMarker {
		output.Info("- 补丁标记之间的所有内容")
		newContent = RemovePatchMarkerContent(newContent)
	} else if opts.Force && hasAliasResources {
		// 强制移除别名资源
		existingResources := GetExistingAliasResources(contentStr)
		for _, res := range existingResources {
			output.Info("- 资源: %s", res)
		}
		newContent = RemoveAliasResources(newContent, existingResources)
	}

	// 5. 如果是 dry-run，到此结束
	if opts.DryRun {
		fmt.Println()
		output.Separator()
		output.Info("Dry-run 完成，未修改任何文件")
		output.Separator()
		return result
	}

	// 6. 备份原文件（如果未禁用）
	if !opts.NoBackup {
		backupPath, err := BackupFile(opts.TemplatePath)
		if err != nil {
			output.Error("备份文件失败: %s", err.Error())
			result.ExitCode = exitcode.ParamError
			return result
		}
		result.BackupPath = backupPath
		output.Success("已备份原文件到: %s", backupPath)
	}

	// 7. 写入新内容
	if err := os.WriteFile(opts.TemplatePath, []byte(newContent), 0644); err != nil {
		output.Error("写入模板文件失败: %s", err.Error())
		result.ExitCode = exitcode.ParamError
		return result
	}
	output.Success("已移除补丁内容")

	// 8. 输出结果
	fmt.Println()
	output.Separator()
	output.Info("Unpatch 完成!")
	output.Separator()
	output.Info("模板文件: %s", opts.TemplatePath)
	if result.BackupPath != "" {
		output.Info("备份文件: %s", result.BackupPath)
	}
	fmt.Println()
	output.Info("下一步:")
	output.Info("  重新打补丁: lad patch --template %s", opts.TemplatePath)

	return result
}

// RemovePatchMarkerContent 移除标记之间的内容
func RemovePatchMarkerContent(content string) string {
	// 查找开始标记和结束标记
	startIdx := strings.Index(content, PatchStartMarker)
	endIdx := strings.Index(content, PatchEndMarker)

	if startIdx == -1 || endIdx == -1 {
		return content
	}

	// 找到开始标记之前的换行符位置
	lineStartIdx := startIdx
	for lineStartIdx > 0 && content[lineStartIdx-1] != '\n' {
		lineStartIdx--
	}

	// 找到结束标记之后的换行符位置
	lineEndIdx := endIdx + len(PatchEndMarker)
	for lineEndIdx < len(content) && content[lineEndIdx] != '\n' {
		lineEndIdx++
	}
	// 包含换行符
	if lineEndIdx < len(content) {
		lineEndIdx++
	}

	// 移除标记之间的内容（包括标记本身）
	return content[:lineStartIdx] + content[lineEndIdx:]
}

// RemoveAliasResources 移除版本/别名资源
func RemoveAliasResources(content string, resources []string) string {
	for _, resourceName := range resources {
		// 匹配资源定义（从资源名到下一个同级资源或文件末尾）
		// 格式: "  ResourceName:\n    Type: ...\n    Properties: ..."
		pattern := regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(resourceName) + `:\s*\n(?:    .*\n)*`)
		content = pattern.ReplaceAllString(content, "")
	}
	return content
}

// HasLineModifications 检查是否存在行级别的修改标记
func HasLineModifications(content string) bool {
	return strings.Contains(content, LineModifyMarker)
}

// GetModifiedLines 获取所有被修改的原始行
func GetModifiedLines(content string) []string {
	var lines []string
	// 匹配格式: # <<< LAD_ORIGINAL: xxx >>>
	pattern := regexp.MustCompile(regexp.QuoteMeta(LineModifyMarker) + ` (.+?) ` + regexp.QuoteMeta(LineModifyEndMarker))
	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			lines = append(lines, match[1])
		}
	}
	return lines
}

// RestoreModifiedLines 还原所有被修改的行
// 删除注释行和新行，恢复原始行
func RestoreModifiedLines(content string) string {
	// 匹配格式:
	// {indent}# <<< LAD_ORIGINAL: {original_content} >>>
	// {indent}{new_content}
	// 替换为:
	// {indent}{original_content}
	pattern := regexp.MustCompile(`(?m)^(\s*)` + regexp.QuoteMeta(LineModifyMarker) + ` (.+?) ` + regexp.QuoteMeta(LineModifyEndMarker) + `\n\s*.+`)
	return pattern.ReplaceAllString(content, "${1}${2}")
}
