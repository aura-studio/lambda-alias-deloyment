// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"os"
	"time"

	"github.com/aura-studio/lambda-alias-deployment/internal/aws"
	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
	"github.com/spf13/cobra"
)

var (
	// auto 命令选项
	autoWait time.Duration
)

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "自动递进灰度发布",
	Long: `自动递进灰度发布，按 10% → 25% → 50% → 75% → 100% 的顺序逐步切换流量。

该命令会执行以下操作：
1. 获取 live 和 latest 别名的版本
2. 按顺序执行灰度策略：canary10 → canary25 → canary50 → canary75
3. 每个阶段等待指定时间（默认 1 分钟）
4. 最后执行 promote 完成 100% 切换

使用 --wait 参数指定每个阶段的等待时间，例如：
  lad auto --wait 5m   # 每阶段等待 5 分钟
  lad auto --wait 30s  # 每阶段等待 30 秒`,
	Run: runAuto,
}

func init() {
	autoCmd.Flags().DurationVar(&autoWait, "wait", 1*time.Minute, "每个灰度阶段的等待时间")
	rootCmd.AddCommand(autoCmd)
}

func runAuto(cmd *cobra.Command, args []string) {
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

	output.Info("开始自动灰度发布...")
	output.Info("环境: %s", env)
	output.Info("函数: %s", functionName)
	output.Info("等待时间: %v", autoWait)
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

	// 5. 获取 live 和 latest 别名的版本
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

	// 6. 检查 live 和 latest 是否指向同一版本
	if liveVersion == latestVersion {
		output.Error("live 和 latest 指向同一版本 (%s)，请先执行 deploy 部署新版本", liveVersion)
		os.Exit(exitcode.ParamError)
		return
	}

	// 7. 按顺序执行灰度策略
	strategies := []CanaryStrategy{Canary10, Canary25, Canary50, Canary75}

	for i, strategy := range strategies {
		output.Separator()
		output.Info("[%d/%d] 执行灰度策略: %s (%.0f%% 流量到新版本)", i+1, len(strategies), strategy, strategy.Weight()*100)

		exitCode = lambdaClient.ConfigureCanary(ctx, functionName, "live", liveVersion, latestVersion, strategy.Weight())
		if exitCode != exitcode.Success {
			os.Exit(exitCode)
			return
		}

		output.Success("灰度配置完成")
		output.Info("流量分配: %.0f%% v%s, %.0f%% v%s", (1-strategy.Weight())*100, liveVersion, strategy.Weight()*100, latestVersion)

		// 最后一个策略不需要等待
		if i < len(strategies)-1 {
			output.Info("等待 %v...", autoWait)
			time.Sleep(autoWait)
		}
	}

	// 8. 执行 promote
	output.Separator()
	output.Info("执行 promote，完成 100%% 切换...")

	// 更新 previous 别名
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "previous", liveVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("previous 别名已更新到版本 %s", liveVersion)

	// 更新 live 别名
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "live", latestVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("live 别名已更新到版本 %s", latestVersion)

	// 9. 输出结果
	output.Separator()
	output.Success("自动灰度发布完成!")
	output.Info("")
	output.Info("版本变更:")
	output.Info("  - previous: -> 版本 %s", liveVersion)
	output.Info("  - live: 版本 %s -> 版本 %s (100%%)", liveVersion, latestVersion)
	output.Info("")
	output.Info("总耗时: %v", time.Duration(len(strategies)-1)*autoWait)
	output.Info("")
	output.Info("如需回退: lad rollback --env %s", env)
}
