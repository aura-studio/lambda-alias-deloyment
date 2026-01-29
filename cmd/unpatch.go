// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"os"

	"github.com/aura-studio/lambda-alias-deployment/internal/patcher"
	"github.com/spf13/cobra"
)

var (
	// unpatch 命令选项
	unpatchTemplate string
	unpatchDryRun   bool
	unpatchForce    bool
	unpatchNoBackup bool
)

var unpatchCmd = &cobra.Command{
	Use:   "unpatch",
	Short: "移除 template.yaml 中的补丁内容",
	Long: `移除 template.yaml 中的补丁内容。

该命令会执行以下操作：
1. 检查模板是否包含补丁标记
2. 移除标记之间的所有内容
3. 如果没有标记但存在版本/别名资源，需要 --force 确认

注意：此命令不需要 --env 参数`,
	Run: runUnpatch,
}

func init() {
	unpatchCmd.Flags().StringVar(&unpatchTemplate, "template", "template.yaml", "模板文件路径")
	unpatchCmd.Flags().BoolVar(&unpatchDryRun, "dry-run", false, "仅预览，不实际修改")
	unpatchCmd.Flags().BoolVar(&unpatchForce, "force", false, "强制移除（即使无标记）")
	unpatchCmd.Flags().BoolVar(&unpatchNoBackup, "no-backup", false, "不创建备份文件")
	rootCmd.AddCommand(unpatchCmd)
}

func runUnpatch(cmd *cobra.Command, args []string) {
	opts := patcher.UnpatchOptions{
		TemplatePath: unpatchTemplate,
		DryRun:       unpatchDryRun,
		Force:        unpatchForce,
		NoBackup:     unpatchNoBackup,
	}

	result := patcher.Unpatch(opts)
	os.Exit(result.ExitCode)
}
