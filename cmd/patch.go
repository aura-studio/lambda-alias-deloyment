// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"os"

	"github.com/aura-studio/lambda-alias-deployment/internal/patcher"
	"github.com/spf13/cobra"
)

var (
	// patch 命令选项
	patchTemplate     string
	patchFunctionName string
	patchDryRun       bool
	patchNoBackup     bool
)

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: "给 template.yaml 添加版本和别名资源",
	Long: `给 template.yaml 添加版本和别名资源。

该命令会执行以下操作：
1. 验证模板文件是否有效
2. 检查是否已存在补丁内容
3. 检测函数资源和触发器
4. 添加 Lambda Version 和三个 Alias 资源（live、previous、latest）
5. 修改触发器指向 LiveAlias

注意：此命令不需要 --env 参数`,
	Run: runPatch,
}

func init() {
	patchCmd.Flags().StringVar(&patchTemplate, "template", "template.yaml", "模板文件路径")
	patchCmd.Flags().StringVar(&patchFunctionName, "function", "Function", "函数资源名称")
	patchCmd.Flags().BoolVar(&patchDryRun, "dry-run", false, "仅预览，不实际修改")
	patchCmd.Flags().BoolVar(&patchNoBackup, "no-backup", false, "不创建备份文件")
	rootCmd.AddCommand(patchCmd)
}

func runPatch(cmd *cobra.Command, args []string) {
	opts := patcher.PatchOptions{
		TemplatePath: patchTemplate,
		FunctionName: patchFunctionName,
		DryRun:       patchDryRun,
		NoBackup:     patchNoBackup,
	}

	result := patcher.Patch(opts)
	os.Exit(result.ExitCode)
}
