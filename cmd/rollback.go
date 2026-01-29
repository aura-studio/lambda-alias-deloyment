// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aura-studio/lambda-alias-deployment/internal/aws"
	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
	"github.com/spf13/cobra"
)

var (
	// rollback 命令选项
	reason string
)

// RollbackLog 回退日志条目
type RollbackLog struct {
	Timestamp   time.Time
	Env         string
	FromVersion string
	ToVersion   string
	Reason      string
	Operator    string // 从 USER 环境变量获取
}

// Format 格式化日志条目
// 格式: [timestamp] ENV=env FROM_VERSION=from TO_VERSION=to REASON="reason" OPERATOR=operator
func (l *RollbackLog) Format() string {
	return fmt.Sprintf("[%s] ENV=%s FROM_VERSION=%s TO_VERSION=%s REASON=\"%s\" OPERATOR=%s",
		l.Timestamp.Format(time.RFC3339),
		l.Env,
		l.FromVersion,
		l.ToVersion,
		l.Reason,
		l.Operator,
	)
}

// AppendToFile 追加到日志文件
func (l *RollbackLog) AppendToFile(path string) error {
	// 打开文件，如果不存在则创建，追加模式
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %w", err)
	}
	defer f.Close()

	// 写入日志条目
	_, err = f.WriteString(l.Format() + "\n")
	if err != nil {
		return fmt.Errorf("无法写入日志文件: %w", err)
	}

	return nil
}

// getExecutablePath 获取可执行文件所在目录
func getExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(execPath), nil
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "紧急回退到上一个稳定版本",
	Long: `紧急回退到上一个稳定版本。

该命令会执行以下操作：
1. 获取 live 和 previous 别名的版本
2. 检查是否需要回退（版本是否相同）
3. 更新 live 别名指向 previous 版本并清除灰度配置
4. 记录回退日志到 rollback.log 文件
5. 显示回退结果和下一步操作提示`,
	Run: runRollback,
}

func init() {
	rollbackCmd.Flags().StringVar(&reason, "reason", "", "回退原因")
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) {
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

	output.Info("开始 Rollback...")
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

	// 5. 获取 live 和 previous 别名的版本 (需求 7.1)
	output.Info("获取别名版本...")
	liveVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "live")
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Info("live 别名: 版本 %s", liveVersion)

	previousVersion, exitCode := lambdaClient.GetAliasVersion(ctx, functionName, "previous")
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Info("previous 别名: 版本 %s", previousVersion)

	// 6. 检查 live 和 previous 是否指向同一版本 (需求 7.2)
	if liveVersion == previousVersion {
		output.Success("live 和 previous 已指向同一版本 (%s)，无需回退", liveVersion)
		os.Exit(exitcode.Success)
		return
	}

	// 7. 更新 live 别名指向 previous 版本并清除灰度配置 (需求 7.3)
	output.Separator()
	output.Info("更新 live 别名...")
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "live", previousVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("live 别名已更新到版本 %s", previousVersion)

	// 8. 同时更新 latest 别名，防止下次 promote 又推上问题版本
	output.Info("更新 latest 别名...")
	exitCode = lambdaClient.UpdateAlias(ctx, functionName, "latest", previousVersion)
	if exitCode != exitcode.Success {
		os.Exit(exitCode)
		return
	}
	output.Success("latest 别名已更新到版本 %s", previousVersion)

	// 8. 记录回退日志 (需求 7.4, 7.5, 7.6, 7.7)
	rollbackReason := reason
	if rollbackReason == "" {
		rollbackReason = "未指定原因" // 需求 7.7
	}

	operator := os.Getenv("USER")
	if operator == "" {
		operator = "unknown"
	}

	rollbackLog := &RollbackLog{
		Timestamp:   time.Now(),
		Env:         env,
		FromVersion: liveVersion,
		ToVersion:   previousVersion,
		Reason:      rollbackReason,
		Operator:    operator,
	}

	// 获取可执行文件所在目录
	execDir, err := getExecutablePath()
	if err != nil {
		output.Warning("无法获取可执行文件路径，日志将写入当前目录: %v", err)
		execDir = "."
	}
	logPath := filepath.Join(execDir, "rollback.log")

	if err := rollbackLog.AppendToFile(logPath); err != nil {
		output.Warning("无法写入回退日志: %v", err)
	} else {
		output.Info("回退日志已记录到: %s", logPath)
	}

	// 10. 显示回退结果和下一步操作提示 (需求 7.8)
	output.Separator()
	output.Success("Rollback 完成!")
	output.Info("")
	output.Info("版本变更:")
	output.Info("  - live: 版本 %s -> 版本 %s", liveVersion, previousVersion)
	output.Info("  - latest: -> 版本 %s", previousVersion)
	output.Info("")
	output.Info("回退信息:")
	output.Info("  - 原因: %s", rollbackReason)
	output.Info("  - 操作人: %s", operator)
	output.Info("")
	output.Info("下一步操作:")
	output.Info("  查看当前状态: lad status --env %s", env)
	output.Info("  部署新版本: lad deploy --env %s", env)
}
