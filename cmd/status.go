// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"os"

	"github.com/aura-studio/lad/internal/aws"
	"github.com/aura-studio/lad/internal/exitcode"
	"github.com/aura-studio/lad/internal/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看当前别名状态",
	Long: `查看当前别名状态，包括 live、previous、latest 三个别名的版本信息。

该命令会显示：
1. 三个别名（live、previous、latest）的版本
2. 如果存在活跃的灰度配置，显示灰度状态和流量分配比例
3. 根据当前状态提示可用的操作`,
	Run: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. 验证环境参数
	if err := ValidateEnv(env); err != nil {
		HandleParamError(err)
		return
	}

	// 2. 获取函数名
	functionName, err := GetFunctionName(env)
	if err != nil {
		HandleParamError(err)
		return
	}

	// 3. 获取 AWS Profile
	awsProfile := GetProfile(env)

	output.Info("Lambda 别名状态")
	output.Info("环境: %s", env)
	output.Info("函数: %s", functionName)
	if awsProfile != "" {
		output.Info("Profile: %s", awsProfile)
	}
	output.Separator()

	// 4. 创建 AWS Lambda 客户端
	lambdaClient, err := aws.NewClient(ctx, awsProfile)
	if err != nil {
		output.Error("创建 AWS 客户端失败: %v", err)
		os.Exit(exitcode.AWSError)
		return
	}

	// 5. 获取三个别名版本 (需求 9.1, 9.2)
	output.Info("别名版本:")

	// 获取 live 别名版本
	liveVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "live")
	if exitCode != exitcode.Success {
		// 获取失败显示"未配置" (需求 9.2)
		liveVersion = "未配置"
	}
	output.Info("  - live: %s", liveVersion)

	// 获取 previous 别名版本
	previousVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "previous")
	if exitCode != exitcode.Success {
		// 获取失败显示"未配置" (需求 9.2)
		previousVersion = "未配置"
	}
	output.Info("  - previous: %s", previousVersion)

	// 获取 latest 别名版本
	latestVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "latest")
	if exitCode != exitcode.Success {
		// 获取失败显示"未配置" (需求 9.2)
		latestVersion = "未配置"
	}
	output.Info("  - latest: %s", latestVersion)

	output.Separator()

	// 6. 检查灰度配置 (需求 9.3)
	active, canaryVersion, weight := lambdaClient.CheckCanaryActive(ctx, functionName, "live")

	if active {
		// 存在活跃的灰度配置 (需求 9.3)
		output.Info("灰度状态: 活跃")
		output.Info("  - 主版本: %s (%.0f%%)", liveVersion, (1-weight)*100)
		output.Info("  - 灰度版本: %s (%.0f%%)", canaryVersion, weight*100)
		output.Separator()
		output.Info("可用操作:")
		output.Info("  完成灰度发布: lad promote --env %s", env)
		output.Info("  回退灰度: lad rollback --env %s", env)
		output.Info("  调整灰度比例: lad canary --env %s --strategy <strategy>", env)
	} else {
		// 不存在灰度配置
		output.Info("灰度状态: 无")

		// 判断 live 和 latest 是否相同 (需求 9.4, 9.5)
		// 注意：如果版本是"未配置"，则不进行比较
		if liveVersion != "未配置" && latestVersion != "未配置" {
			if liveVersion == latestVersion {
				// live 等于 latest，系统处于稳定状态 (需求 9.4)
				output.Separator()
				output.Success("系统处于稳定状态")
				output.Info("")
				output.Info("可用操作:")
				output.Info("  部署新版本: lad deploy --env %s", env)
				if previousVersion != "未配置" && previousVersion != liveVersion {
					output.Info("  回退到上一版本: lad rollback --env %s", env)
				}
			} else {
				// live 不等于 latest，有新版本待发布 (需求 9.5)
				output.Separator()
				output.Warning("有新版本待发布")
				output.Info("  当前版本: %s", liveVersion)
				output.Info("  待发布版本: %s", latestVersion)
				output.Info("")
				output.Info("可用操作:")
				output.Info("  开始灰度发布: lad canary --env %s --strategy canary10", env)
				output.Info("  直接发布: lad promote --env %s --skip-canary", env)
			}
		} else {
			// 有别名未配置的情况
			output.Separator()
			output.Warning("部分别名未配置")
			output.Info("")
			output.Info("可用操作:")
			output.Info("  部署新版本: lad deploy --env %s", env)
		}
	}
}
