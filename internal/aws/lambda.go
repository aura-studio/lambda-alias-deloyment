// Package aws provides AWS Lambda client functionality.
package aws

import (
	"context"
	"strings"

	"github.com/aura-studio/lambda-alias-deployment/internal/exitcode"
	"github.com/aura-studio/lambda-alias-deployment/internal/output"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// Client 封装 Lambda API 操作
type Client struct {
	client *lambda.Client
}

// NewClient 创建新的 Lambda 客户端
// 如果 profile 为空，则使用默认的 AWS 配置
func NewClient(ctx context.Context, profile string) (*Client, error) {
	var cfg config.LoadOptions

	// 如果指定了 profile，则使用该 profile
	opts := []func(*config.LoadOptions) error{}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	_ = cfg // 避免未使用变量警告

	lambdaClient := lambda.NewFromConfig(awsCfg)
	return &Client{client: lambdaClient}, nil
}

// classifyError 根据错误信息分类返回退出码
// 网络错误关键词: unable to locate credentials, could not connect, connection refused, network, timeout, timed out, unreachable
// 资源不存在关键词: resourcenotfoundexception, does not exist, not found, cannot find
func classifyError(err error) int {
	if err == nil {
		return exitcode.Success
	}

	errMsg := strings.ToLower(err.Error())

	// 检查网络错误关键词
	networkKeywords := []string{
		"unable to locate credentials",
		"could not connect",
		"connection refused",
		"network",
		"timeout",
		"timed out",
		"unreachable",
	}
	for _, keyword := range networkKeywords {
		if strings.Contains(errMsg, keyword) {
			return exitcode.NetworkError
		}
	}

	// 检查资源不存在关键词
	resourceNotFoundKeywords := []string{
		"resourcenotfoundexception",
		"does not exist",
		"not found",
		"cannot find",
	}
	for _, keyword := range resourceNotFoundKeywords {
		if strings.Contains(errMsg, keyword) {
			return exitcode.ResourceNotFound
		}
	}

	// 其他 AWS 错误
	return exitcode.AWSError
}

// CreateVersion 创建新版本
// 返回: 版本号, 错误
func (c *Client) CreateVersion(ctx context.Context, functionName, description string) (string, error) {
	input := &lambda.PublishVersionInput{
		FunctionName: aws.String(functionName),
		Description:  aws.String(description),
	}

	result, err := c.client.PublishVersion(ctx, input)
	if err != nil {
		return "", err
	}

	return aws.ToString(result.Version), nil
}

// GetAliasVersion 获取别名指向的版本
// 返回: 版本号, 退出码
func (c *Client) GetAliasVersion(ctx context.Context, functionName, aliasName string) (string, int) {
	input := &lambda.GetAliasInput{
		FunctionName: aws.String(functionName),
		Name:         aws.String(aliasName),
	}

	result, err := c.client.GetAlias(ctx, input)
	if err != nil {
		exitCode := classifyError(err)
		output.Error("%v", err)
		return "", exitCode
	}

	return aws.ToString(result.FunctionVersion), exitcode.Success
}

// UpdateAlias 更新别名指向（清除路由配置）
// 返回: 退出码
func (c *Client) UpdateAlias(ctx context.Context, functionName, aliasName, version string) int {
	input := &lambda.UpdateAliasInput{
		FunctionName:    aws.String(functionName),
		Name:            aws.String(aliasName),
		FunctionVersion: aws.String(version),
		// 清除路由配置
		RoutingConfig: &types.AliasRoutingConfiguration{
			AdditionalVersionWeights: map[string]float64{},
		},
	}

	_, err := c.client.UpdateAlias(ctx, input)
	if err != nil {
		exitCode := classifyError(err)
		output.Error("%v", err)
		return exitCode
	}

	return exitcode.Success
}

// ConfigureCanary 配置灰度流量
// 参数: functionName, aliasName, mainVersion, canaryVersion, weight (0.0-1.0)
// 返回: 退出码
func (c *Client) ConfigureCanary(ctx context.Context, functionName, aliasName, mainVersion, canaryVersion string, weight float64) int {
	input := &lambda.UpdateAliasInput{
		FunctionName:    aws.String(functionName),
		Name:            aws.String(aliasName),
		FunctionVersion: aws.String(mainVersion),
		RoutingConfig: &types.AliasRoutingConfiguration{
			AdditionalVersionWeights: map[string]float64{
				canaryVersion: weight,
			},
		},
	}

	_, err := c.client.UpdateAlias(ctx, input)
	if err != nil {
		exitCode := classifyError(err)
		output.Error("%v", err)
		return exitCode
	}

	return exitcode.Success
}

// CheckCanaryActive 检查是否有活跃的灰度配置
// 返回: 是否活跃, 灰度版本, 权重
func (c *Client) CheckCanaryActive(ctx context.Context, functionName, aliasName string) (bool, string, float64) {
	input := &lambda.GetAliasInput{
		FunctionName: aws.String(functionName),
		Name:         aws.String(aliasName),
	}

	result, err := c.client.GetAlias(ctx, input)
	if err != nil {
		// 如果获取别名失败，返回无活跃灰度
		return false, "", 0
	}

	// 检查是否有路由配置
	if result.RoutingConfig == nil || len(result.RoutingConfig.AdditionalVersionWeights) == 0 {
		return false, "", 0
	}

	// 获取灰度版本和权重
	for version, weight := range result.RoutingConfig.AdditionalVersionWeights {
		return true, version, weight
	}

	return false, "", 0
}

// VerifyVersionExists 验证版本是否存在
// 返回: 退出码
func (c *Client) VerifyVersionExists(ctx context.Context, functionName, version string) int {
	// 使用 GetFunction 并指定 Qualifier 来验证版本是否存在
	input := &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
		Qualifier:    aws.String(version),
	}

	_, err := c.client.GetFunction(ctx, input)
	if err != nil {
		exitCode := classifyError(err)
		output.Error("%v", err)
		return exitCode
	}

	return exitcode.Success
}
