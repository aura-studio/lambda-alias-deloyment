// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aura-studio/lambda-alias-deployment/internal/aws"
	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
	"github.com/spf13/cobra"
)

var (
	// canary 命令选项
	strategy    string
	autoPromote bool
)

var canaryCmd = &cobra.Command{
	Use:   "canary",
	Short: "执行灰度发布",
	Long: `执行灰度发布，将部分流量路由到新版本进行验证。

该命令会执行以下操作：
1. 验证灰度策略参数
2. 获取 live 和 latest 别名的版本
3. 配置 live 别名的流量路由
4. 显示流量分配比例和下一步操作提示

支持的灰度策略：
  canary10 - 10% 流量到新版本
  canary25 - 25% 流量到新版本
  canary50 - 50% 流量到新版本
  canary75 - 75% 流量到新版本`,
	Run: runCanary,
}

func init() {
	canaryCmd.Flags().StringVar(&strategy, "strategy", "", "灰度策略 (canary10|canary25|canary50|canary75)")
	canaryCmd.Flags().BoolVar(&autoPromote, "auto-promote", false, "自动执行 promote（仅限 canary75 策略）")
	canaryCmd.MarkFlagRequired("strategy")
	rootCmd.AddCommand(canaryCmd)
}

func runCanary(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. 验证环境参数
	if err := ValidateEnv(env); err != nil {
		HandleParamError(err)
		return
	}

	// 2. 验证 --strategy 参数 (需求 5.1, 5.2, 5.3)
	canaryStrategy := CanaryStrategy(strategy)
	if !canaryStrategy.IsValid() {
		validStrategies := make([]string, len(AllStrategies))
		for i, s := range AllStrategies {
			validStrategies[i] = string(s)
		}
		HandleParamError(fmt.Errorf("无效的灰度策略 '%s'，有效策略为: %s", strategy, strings.Join(validStrategies, ", ")))
		return
	}

	// 3. 验证 --auto-promote 参数 (需求 5.7, 5.8)
	if err := ValidateAutoPromote(canaryStrategy, autoPromote); err != nil {
		HandleParamError(err)
		return
	}

	// 4. 获取函数名
	functionName, err := GetFunctionName(env)
	if err != nil {
		HandleParamError(err)
		return
	}

	// 5. 获取 AWS Profile
	awsProfile := GetProfile(env)

	output.Info("开始灰度发布...")
	output.Info("环境: %s", env)
	output.Info("函数: %s", functionName)
	output.Info("策略: %s (%.0f%% 流量到新版本)", canaryStrategy, canaryStrategy.Weight()*100)
	if awsProfile != "" {
		output.Info("Profile: %s", awsProfile)
	}
	output.Separator()

	// 6. 创建 AWS Lambda 客户端
	lambdaClient, err := aws.NewClient(ctx, awsProfile)
	if err != nil {
		output.Error("创建 AWS 客户端失败: %v", err)
		os.Exit(exitcode.AWSError)
		return
	}

	// 7. 获取 live 和 latest 别名的版本 (需求 5.4)
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

	// 8. 检查 live 和 latest 是否指向同一版本 (需求 5.5)
	if liveVersion == latestVersion {
		output.Error("live 和 latest 指向同一版本 (%s)，请先执行 deploy 部署新版本", liveVersion)
		os.Exit(exitcode.ParamError)
		return
	}

	// 9. 配置灰度流量 (需求 5.6)
	output.Separator()
	output.Info("配置灰度流量...")
	exitCode = lambdaClient.ConfigureCanary(ctx, functionName, "live", liveVersion, latestVersion, canaryStrategy.Weight())
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("灰度配置完成")

	// 10. 显示流量分配和下一步提示 (需求 5.9)
	output.Separator()
	output.Success("灰度发布配置成功!")
	output.Info("")
	output.Info("流量分配:")
	output.Info("  - 稳定版本 (v%s): %.0f%%", liveVersion, (1-canaryStrategy.Weight())*100)
	output.Info("  - 灰度版本 (v%s): %.0f%%", latestVersion, canaryStrategy.Weight()*100)
	output.Info("")

	// 11. 处理 --auto-promote 参数 (需求 5.7)
	if autoPromote && canaryStrategy == Canary75 {
		output.Info("检测到 --auto-promote 参数，自动执行 promote...")
		output.Separator()
		executePromote(ctx, lambdaClient, functionName, liveVersion, latestVersion)
		return
	}

	// 显示下一步操作提示
	output.Info("下一步操作:")
	nextStrategy := canaryStrategy.NextStrategy()
	if nextStrategy != canaryStrategy {
		output.Info("  增加灰度比例: lad canary --env %s --strategy %s", env, nextStrategy)
	}
	output.Info("  完成灰度发布: lad promote --env %s", env)
	output.Info("  回退灰度发布: lad rollback --env %s", env)
}

// executePromote 执行 promote 操作（用于 auto-promote）
func executePromote(ctx context.Context, lambdaClient *aws.Client, functionName, liveVersion, latestVersion string) {
	output.Info("执行 promote...")

	// 更新 previous 别名指向原 live 版本
	output.Info("更新 previous 别名...")
	exitCode := lambdaClient.UpdateAlias(ctx, functionName, "previous", liveVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("previous 别名已更新到版本 %s", liveVersion)

	// 更新 live 别名指向 latest 版本并清除灰度配置
	output.Info("更新 live 别名...")
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "live", latestVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("live 别名已更新到版本 %s", latestVersion)

	output.Separator()
	output.Success("Promote 完成!")
	output.Info("")
	output.Info("版本变更:")
	output.Info("  - previous: -> 版本 %s", liveVersion)
	output.Info("  - live: 版本 %s -> 版本 %s", liveVersion, latestVersion)
}
