// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aura-studio/lad/internal/aws"
	"github.com/aura-studio/lad/internal/exitcode"
	"github.com/aura-studio/lad/internal/output"
	"github.com/spf13/cobra"
)

var (
	// auto 命令选项
	autoPercent int
	autoWait    time.Duration
)

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "自动递进灰度发布",
	Long: `自动递进灰度发布，按指定步长逐步增加流量到新版本。

该命令会执行以下操作：
1. 获取 live 和 latest 别名的版本
2. 按 --percent 指定的步长递增灰度比例
3. 每个阶段等待 --wait 指定的时间
4. 达到 100% 后执行 promote 完成切换

示例：
  lad auto --percent 10 --wait 5m   # 每次增加 10%，每阶段等待 5 分钟
                                     # 10% → 20% → 30% → ... → 100%
  
  lad auto --percent 25 --wait 1h   # 每次增加 25%，每阶段等待 1 小时
                                     # 25% → 50% → 75% → 100%`,
	Run: runAuto,
}

func init() {
	autoCmd.Flags().IntVar(&autoPercent, "percent", 10, "每次增加的灰度百分比 (1-100)")
	autoCmd.Flags().DurationVar(&autoWait, "wait", 5*time.Minute, "每个灰度阶段的等待时间")
	rootCmd.AddCommand(autoCmd)
}

func runAuto(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. 验证环境参数
	if err := ValidateEnv(env); err != nil {
		HandleParamError(err)
		return
	}

	// 2. 验证 --percent 参数
	if autoPercent < 1 || autoPercent > 100 {
		HandleParamError(fmt.Errorf("无效的百分比 '%d'，有效范围为 1-100", autoPercent))
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

	// 5. 计算灰度步骤
	var steps []int
	for pct := autoPercent; pct < 100; pct += autoPercent {
		steps = append(steps, pct)
	}
	// 确保最后一步是 100%（由 promote 完成）

	output.Info("开始自动灰度发布...")
	output.Info("环境: %s", env)
	output.Info("函数: %s", functionName)
	output.Info("步长: %d%%", autoPercent)
	output.Info("等待时间: %v", autoWait)
	output.Info("灰度步骤: %v → promote", steps)
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

	// 7. 获取 live 和 latest 别名的版本
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

	// 8. 检查 live 和 latest 是否指向同一版本
	if liveVersion == latestVersion {
		output.Error("live 和 latest 指向同一版本 (%s)，请先执行 deploy 部署新版本", liveVersion)
		os.Exit(exitcode.ParamError)
		return
	}

	// 9. 按顺序执行灰度
	totalSteps := len(steps) + 1 // 包括最后的 promote
	for i, pct := range steps {
		output.Separator()
		output.Info("[%d/%d] 执行灰度: %d%% 流量到新版本", i+1, totalSteps, pct)

		weight := float64(pct) / 100.0
		exitCode = lambdaClient.ConfigureCanary(ctx, functionName, "live", liveVersion, latestVersion, weight)
		if exitCode != exitcode.Success {
			os.Exit(exitCode)
			return
		}

		output.Success("灰度配置完成")
		output.Info("流量分配: %d%% v%s, %d%% v%s", 100-pct, liveVersion, pct, latestVersion)

		output.Info("等待 %v...", autoWait)
		time.Sleep(autoWait)
	}

	// 10. 执行 promote
	output.Separator()
	output.Info("[%d/%d] 执行 promote，完成 100%% 切换...", totalSteps, totalSteps)

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

	// 11. 输出结果
	output.Separator()
	output.Success("自动灰度发布完成!")
	output.Info("")
	output.Info("版本变更:")
	output.Info("  - previous: -> 版本 %s", liveVersion)
	output.Info("  - live: 版本 %s -> 版本 %s (100%%)", liveVersion, latestVersion)
	output.Info("")
	output.Info("总耗时: %v", time.Duration(len(steps))*autoWait)
	output.Info("")
	output.Info("如需回退: lad rollback --env %s", env)
}
