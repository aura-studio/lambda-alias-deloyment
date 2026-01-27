// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/aura-studio/lambda-alias-deployment/internal/aws"
	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
	"github.com/spf13/cobra"
)

// CommandExecutor 定义命令执行接口，用于测试时 mock
type CommandExecutor interface {
	Run(name string, args ...string) error
}

// RealCommandExecutor 实际的命令执行器
type RealCommandExecutor struct{}

// Run 执行实际的系统命令
func (e *RealCommandExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// 默认命令执行器
var cmdExecutor CommandExecutor = &RealCommandExecutor{}

// SetCommandExecutor 设置命令执行器（用于测试）
func SetCommandExecutor(executor CommandExecutor) {
	cmdExecutor = executor
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "部署新版本并更新 latest 别名",
	Long: `部署新版本并更新 latest 别名。

该命令会执行以下操作：
1. 检查是否存在未完成的灰度发布
2. 执行 SAM build 命令
3. 执行 SAM deploy 命令
4. 创建新的 Lambda 版本
5. 更新 latest 别名指向新版本`,
	Run: runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) {
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

	output.Info("开始部署...")
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

	// 5. 检查是否有未完成的灰度发布
	output.Info("检查灰度状态...")
	active, canaryVersion, weight := lambdaClient.CheckCanaryActive(ctx, functionName, "live")
	if active {
		output.Error("存在未完成的灰度发布 (版本 %s, 权重 %.0f%%)", canaryVersion, weight*100)
		output.Info("请先执行 promote 完成灰度发布，或执行 rollback 回退")
		os.Exit(exitcode.ParamError)
		return
	}
	output.Success("无活跃灰度发布")

	// 6. 执行 sam build
	output.Separator()
	output.Info("执行 SAM build...")
	if err := cmdExecutor.Run("sam", "build"); err != nil {
		output.Error("SAM build 失败: %v", err)
		os.Exit(exitcode.AWSError)
		return
	}
	output.Success("SAM build 完成")

	// 7. 执行 sam deploy
	output.Separator()
	output.Info("执行 SAM deploy...")
	deployTime := time.Now().Format("2006-01-02 15:04:05")
	description := fmt.Sprintf("Deployed at %s", deployTime)

	samDeployArgs := []string{
		"deploy",
		"--parameter-overrides",
		fmt.Sprintf("Runtime=provided.al2023 Description=\"%s\"", description),
	}

	if err := cmdExecutor.Run("sam", samDeployArgs...); err != nil {
		output.Error("SAM deploy 失败: %v", err)
		os.Exit(exitcode.AWSError)
		return
	}
	output.Success("SAM deploy 完成")

	// 8. 创建新的 Lambda 版本
	output.Separator()
	output.Info("创建新版本...")
	versionDescription := fmt.Sprintf("Deployed at %s", deployTime)
	newVersion, err := lambdaClient.CreateVersion(ctx, functionName, versionDescription)
	if err != nil {
		output.Error("创建版本失败: %v", err)
		os.Exit(exitcode.AWSError)
		return
	}
	output.Success("创建版本 %s", newVersion)

	// 9. 更新 latest 别名指向新版本
	output.Info("更新 latest 别名...")
	exitCode := lambdaClient.UpdateAlias(ctx, functionName, "latest", newVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("latest 别名已更新到版本 %s", newVersion)

	// 10. 显示部署结果和下一步提示
	output.Separator()
	output.Success("部署完成!")
	output.Info("")
	output.Info("部署信息:")
	output.Info("  - 新版本: %s", newVersion)
	output.Info("  - latest 别名: -> 版本 %s", newVersion)
	output.Info("")
	output.Info("下一步操作:")
	output.Info("  执行灰度发布: lad canary --env %s --strategy canary10", env)
}
