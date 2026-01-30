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

var (
	// promote 命令选项
	skipCanary bool
)

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "完成灰度发布，将流量完全切换到新版本",
	Long: `完成灰度发布，将流量完全切换到新版本。

该命令会执行以下操作：
1. 获取 live 和 latest 别名的版本
2. 检查是否有活跃的灰度配置（可通过 --skip-canary 跳过）
3. 更新 previous 别名指向原 live 版本
4. 更新 live 别名指向 latest 版本并清除灰度配置
5. 显示版本变更信息`,
	Run: runPromote,
}

func init() {
	promoteCmd.Flags().BoolVar(&skipCanary, "skip-canary", false, "跳过灰度状态检查")
	rootCmd.AddCommand(promoteCmd)
}

func runPromote(cmd *cobra.Command, args []string) {
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

	output.Info("开始 Promote...")
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

	// 5. 获取 live 和 latest 别名的版本 (需求 6.1)
	output.Info("获取别名版本...")
	liveVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "live")
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Info("live 别名: 版本 %s", liveVersion)

	latestVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "latest")
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Info("latest 别名: 版本 %s", latestVersion)

	// 6. 检查 live 和 latest 是否指向同一版本 (需求 6.2)
	if liveVersion == latestVersion {
		output.Separator()
		output.Warning("live 和 latest 已指向同一版本 (%s)", liveVersion)
		output.Warning("没有新版本需要切换，跳过 promote 操作")
		output.Info("")
		output.Info("可能的原因:")
		output.Info("  - sam deploy 没有检测到代码变化")
		output.Info("  - 已经完成了 promote 操作")
		output.Info("")
		output.Info("如需发布新版本，请先更新代码后重新执行 sam deploy")
		return
	}

	// 7. 检查灰度状态 (需求 6.3, 6.6)
	if !skipCanary {
		active, canaryVersion, weight := lambdaClient.CheckCanaryActive(ctx, functionName, "live")
		if !active {
			// 没有活跃灰度，显示警告但继续执行 (需求 6.3)
			output.Warning("没有活跃的灰度配置，建议先执行 canary 命令进行灰度验证")
		} else {
			output.Info("检测到活跃灰度配置: 版本 %s, 权重 %.0f%%", canaryVersion, weight*100)
		}
	} else {
		// 跳过灰度状态检查 (需求 6.6)
		output.Info("已跳过灰度状态检查 (--skip-canary)")
	}

	// 8. 更新 previous 别名指向原 live 版本 (需求 6.4)
	output.Separator()
	output.Info("更新 previous 别名...")
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "previous", liveVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("previous 别名已更新到版本 %s", liveVersion)

	// 9. 更新 live 别名指向 latest 版本并清除灰度配置 (需求 6.5)
	output.Info("更新 live 别名...")
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "live", latestVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("live 别名已更新到版本 %s", latestVersion)

	// 10. 显示版本变更信息 (需求 6.7)
	output.Separator()
	output.Success("Promote 完成!")
	output.Info("")
	output.Info("版本变更:")
	output.Info("  - previous: -> 版本 %s", liveVersion)
	output.Info("  - live: 版本 %s -> 版本 %s", liveVersion, latestVersion)
	output.Info("")
	output.Info("下一步操作:")
	output.Info("  部署新版本: lad deploy --env %s", env)
	output.Info("  回退到上一版本: lad rollback --env %s", env)
}
