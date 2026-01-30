// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aura-studio/lad/internal/aws"
	"github.com/aura-studio/lad/internal/exitcode"
	"github.com/aura-studio/lad/internal/output"
	"github.com/spf13/cobra"
)

var (
	// switch 命令选项
	switchVersion string
)

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "在极端情况下切换到指定版本",
	Long: `在极端情况下切换到指定版本。

⚠ 警告：此命令绕过正常发布流程，仅用于紧急情况！

该命令会执行以下操作：
1. 验证指定版本是否存在
2. 检查 live 是否已指向目标版本
3. 更新 live 别名指向指定版本并清除灰度配置
4. 注意：此命令不会更新 previous 别名`,
	Run: runSwitch,
}

func init() {
	switchCmd.Flags().StringVar(&switchVersion, "version", "", "目标版本号 (必需)")
	switchCmd.MarkFlagRequired("version")
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. 验证 --version 参数 (需求 8.1)
	if switchVersion == "" {
		HandleParamError(fmt.Errorf("必须指定 --version 参数"))
		return
	}

	// 2. 验证环境参数
	if err := ValidateEnv(env); err != nil {
		HandleParamError(err)
		return
	}

	// 3. 获取函数名
	functionName, err := GetFunctionName(env)
	if err != nil {
		HandleParamError(err)
		return
	}

	// 4. 获取 AWS Profile
	awsProfile := GetProfile(env)

	output.Info("开始 Switch...")
	output.Info("环境: %s", env)
	output.Info("函数: %s", functionName)
	output.Info("目标版本: %s", switchVersion)
	if awsProfile != "" {
		output.Info("Profile: %s", awsProfile)
	}
	output.Separator()

	// 5. 显示警告信息 (需求 8.2)
	output.Warning("此操作绕过正常发布流程，仅用于紧急情况！")
	output.Warning("建议在正常情况下使用 deploy -> canary -> promote 流程")
	output.Separator()

	// 6. 创建 AWS Lambda 客户端
	lambdaClient, err := aws.NewClient(ctx, awsProfile)
	if err != nil {
		output.Error("创建 AWS 客户端失败: %v", err)
		os.Exit(exitcode.AWSError)
		return
	}

	// 7. 验证指定版本是否存在 (需求 8.3, 8.4)
	output.Info("验证版本 %s 是否存在...", switchVersion)
	exitCode := lambdaClient.VerifyVersionExists(ctx, functionName, switchVersion)
	if exitCode != exitcode.Success {
		// 需求 8.4: 如果版本不存在，返回资源不存在错误
		os.Exit(exitCode)
		return
	}
	output.Success("版本 %s 存在", switchVersion)

	// 8. 获取 live 别名当前版本
	output.Info("获取 live 别名当前版本...")
	liveVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "live")
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Info("live 别名: 版本 %s", liveVersion)

	// 9. 检查 live 是否已指向目标版本 (需求 8.5)
	if liveVersion == switchVersion {
		output.Success("live 已指向版本 %s，无需切换", switchVersion)
		os.Exit(exitcode.Success)
		return
	}

	// 10. 更新 live 别名指向指定版本并清除灰度配置 (需求 8.6, 8.7)
	// 注意：不更新 previous 别名 (需求 8.7)
	output.Separator()
	output.Info("更新 live 别名...")
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "live", switchVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("live 别名已更新到版本 %s", switchVersion)

	// 11. 显示注意事项 (需求 8.8)
	output.Separator()
	output.Success("Switch 完成!")
	output.Info("")
	output.Info("版本变更:")
	output.Info("  - live: 版本 %s -> 版本 %s", liveVersion, switchVersion)
	output.Info("")
	output.Warning("注意事项:")
	output.Warning("  - 此操作绕过了正常的发布流程")
	output.Warning("  - previous 别名未更新，仍指向原来的版本")
	output.Warning("  - 如需回退，请使用 rollback 命令或再次使用 switch 命令")
	output.Info("")
	output.Info("下一步操作:")
	output.Info("  查看当前状态: lad status --env %s", env)
	output.Info("  回退到上一版本: lad rollback --env %s", env)
}
