// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aura-studio/lambda-alias-deployment/internal/aws"
	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
	"github.com/spf13/cobra"
)

var (
	// canary 命令选项
	percent int
)

var canaryCmd = &cobra.Command{
	Use:   "canary",
	Short: "执行灰度发布",
	Long: `执行灰度发布，将部分流量路由到新版本进行验证。

该命令会执行以下操作：
1. 验证灰度百分比参数 (0-100)
2. 获取 live 和 latest 别名的版本
3. 配置 live 别名的流量路由
4. 显示流量分配比例和下一步操作提示

使用 --percent 参数指定新版本流量百分比：
  lad canary --percent 0    # 清除灰度配置
  lad canary --percent 10   # 10% 流量到新版本
  lad canary --percent 50   # 50% 流量到新版本
  lad canary --percent 100  # 100% 流量到新版本（不更新 previous，建议用 promote）

如需自动递进灰度，请使用 'lad auto' 命令`,
	Run: runCanary,
}

func init() {
	canaryCmd.Flags().IntVar(&percent, "percent", -1, "新版本流量百分比 (0-100)")
	canaryCmd.MarkFlagRequired("percent")
	rootCmd.AddCommand(canaryCmd)
}

func runCanary(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. 验证环境参数
	if err := ValidateEnv(env); err != nil {
		HandleParamError(err)
		return
	}

	// 2. 验证 --percent 参数
	if percent < 0 || percent > 100 {
		HandleParamError(fmt.Errorf("无效的百分比 '%d'，有效范围为 0-100", percent))
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

	output.Info("开始灰度发布...")
	output.Info("环境: %s", env)
	output.Info("函数: %s", functionName)
	output.Info("灰度比例: %d%% 流量到新版本", percent)
	if awsProfile != "" {
		output.Info("Profile: %s", awsProfile)
	}
	output.Separator()

	// 5. 创建 AWS Lambda 客户端
	lambdaClient, err := aws.NewClient(ctx, awsProfile)
	if err != nil {
		output.Error("创建 AWS 客户端失败: %v", err)
		os.Exit(exitcode.AWSError)
		return
	}

	// 6. 获取 live 和 latest 别名的版本
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

	// 7. 检查 live 和 latest 是否指向同一版本
	// percent=0 允许在同版本时执行（用于清除灰度配置）
	if liveVersion == latestVersion && percent != 0 {
		output.Error("live 和 latest 指向同一版本 (%s)，请先执行 deploy 部署新版本", liveVersion)
		os.Exit(exitcode.ParamError)
		return
	}

	// 8. percent=100 警告
	if percent == 100 {
		output.Warning("--percent 100 会将 100%% 流量切到新版本，但不会更新 previous 别名")
		output.Warning("建议使用 'lad promote' 完成正式发布")
	}

	// 9. 配置灰度流量
	weight := float64(percent) / 100.0
	output.Separator()
	if percent == 0 {
		output.Info("清除灰度配置...")
		// percent=0 直接更新别名到 liveVersion，清除路由配置
		exitCode = lambdaClient.UpdateAlias(ctx, functionName, "live", liveVersion)
	} else {
		output.Info("配置灰度流量...")
		exitCode = lambdaClient.ConfigureCanary(ctx, functionName, "live", liveVersion, latestVersion, weight)
	}
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	if percent == 0 {
		output.Success("灰度配置已清除")
	} else {
		output.Success("灰度配置完成")
	}

	// 10. 显示流量分配和下一步提示
	output.Separator()
	output.Success("灰度发布配置成功!")
	output.Info("")
	if percent == 0 {
		output.Info("流量分配:")
		output.Info("  - 稳定版本 (v%s): 100%%", liveVersion)
		output.Info("")
		output.Info("下一步操作:")
		output.Info("  重新开始灰度: lad canary --env %s --percent 10", env)
		output.Info("  部署新版本: lad deploy --env %s", env)
	} else {
		output.Info("流量分配:")
		output.Info("  - 稳定版本 (v%s): %d%%", liveVersion, 100-percent)
		output.Info("  - 灰度版本 (v%s): %d%%", latestVersion, percent)
		output.Info("")

		// 显示下一步操作提示
		output.Info("下一步操作:")
		if percent < 100 {
			output.Info("  增加灰度比例: lad canary --env %s --percent <更高百分比>", env)
		}
		output.Info("  完成灰度发布: lad promote --env %s", env)
		output.Info("  回退灰度发布: lad rollback --env %s", env)
	}
}
