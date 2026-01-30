// Package cmd implements the command line interface for the lad tool.
package cmd

import (
	"fmt"
	"os"

	"github.com/aura-studio/lad/internal/config"
	"github.com/aura-studio/lad/internal/exitcode"
	"github.com/aura-studio/lad/internal/output"
	"github.com/spf13/cobra"
)

var (
	// 全局选项
	env      string // 环境 (test/prod)，默认 test
	profile  string // AWS Profile
	function string // Lambda 函数名
)

// samconfigPath 是 samconfig.toml 的路径，可在测试中覆盖
var samconfigPath = "samconfig.toml"

var rootCmd = &cobra.Command{
	Use:   "lad",
	Short: "Lambda Alias Deployment - Lambda 函数灰度发布工具",
	Long: `Lambda Alias Deployment (lad) 是一个用于管理 AWS Lambda 函数版本和别名的命令行工具。
支持灰度发布、回退和版本切换等功能。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 没有子命令时显示帮助信息
		cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&env, "env", "test", "指定环境 (test|prod)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "AWS Profile 名称")
	rootCmd.PersistentFlags().StringVar(&function, "function", "", "Lambda 函数名称")
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

// ValidateEnv 验证环境参数
// 只接受 "test" 或 "prod"，否则返回参数错误
func ValidateEnv(envValue string) error {
	if envValue != "test" && envValue != "prod" {
		return fmt.Errorf("无效的环境值 '%s'，有效值为: test, prod", envValue)
	}
	return nil
}

// GetFunctionName 获取函数名
// 优先级: --function > samconfig.toml
func GetFunctionName(envValue string) (string, error) {
	// 优先使用 --function 选项
	if function != "" {
		return function, nil
	}

	// 尝试从 samconfig.toml 读取
	samConfig, err := config.LoadSAMConfig(samconfigPath)
	if err != nil {
		return "", fmt.Errorf("无法读取 samconfig.toml，请通过 --function 指定函数名: %w", err)
	}

	functionName := samConfig.GetFunctionName(envValue)
	if functionName == "" {
		return "", fmt.Errorf("samconfig.toml 中未找到 stack_name 配置，请通过 --function 指定函数名")
	}

	return functionName, nil
}

// GetProfile 获取 AWS Profile
// 优先级: --profile > samconfig.toml
// 如果都未指定，返回空字符串（使用默认 AWS 配置）
func GetProfile(envValue string) string {
	// 优先使用 --profile 选项
	if profile != "" {
		return profile
	}

	// 尝试从 samconfig.toml 读取
	samConfig, err := config.LoadSAMConfig(samconfigPath)
	if err != nil {
		// 无法读取配置文件，返回空字符串使用默认配置
		return ""
	}

	return samConfig.GetProfile(envValue)
}

// GetEnv 获取当前环境值
func GetEnv() string {
	return env
}

// handleError 处理错误并退出
func handleError(err error, code int) {
	output.Error("%s", err.Error())
	os.Exit(code)
}

// HandleParamError 处理参数错误
func HandleParamError(err error) {
	handleError(err, exitcode.ParamError)
}

// SetFunction 设置函数名（用于测试）
func SetFunction(f string) {
	function = f
}

// SetProfile 设置 AWS Profile（用于测试）
func SetProfile(p string) {
	profile = p
}

// SetSamconfigPath 设置 samconfig.toml 路径（用于测试）
func SetSamconfigPath(path string) {
	samconfigPath = path
}
