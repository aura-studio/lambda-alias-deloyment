package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// DeployParameters 表示 deploy 参数
type DeployParameters struct {
	StackName string `toml:"stack_name"`
	Profile   string `toml:"profile"`
}

// Deploy 表示 deploy 配置
type Deploy struct {
	Parameters DeployParameters `toml:"parameters"`
}

// EnvConfig 表示环境配置
type EnvConfig struct {
	Deploy Deploy `toml:"deploy"`
}

// SAMConfig 表示 samconfig.toml 的结构
type SAMConfig struct {
	envConfigs map[string]EnvConfig
}

// LoadSAMConfig 加载 samconfig.toml 文件
func LoadSAMConfig(path string) (*SAMConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %w", err)
	}

	// 解析为通用 map 结构
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("无法解析配置文件: %w", err)
	}

	config := &SAMConfig{
		envConfigs: make(map[string]EnvConfig),
	}

	// 遍历顶层键，查找环境配置
	for key, value := range raw {
		if key == "version" {
			continue
		}

		// 尝试解析为环境配置
		envMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		var envConfig EnvConfig

		// 解析 deploy 配置
		if deployMap, ok := envMap["deploy"].(map[string]interface{}); ok {
			if paramsMap, ok := deployMap["parameters"].(map[string]interface{}); ok {
				if stackName, ok := paramsMap["stack_name"].(string); ok {
					envConfig.Deploy.Parameters.StackName = stackName
				}
				if profile, ok := paramsMap["profile"].(string); ok {
					envConfig.Deploy.Parameters.Profile = profile
				}
			}
		}

		config.envConfigs[key] = envConfig
	}

	return config, nil
}

// GetStackName 获取指定环境的 stack_name
func (c *SAMConfig) GetStackName(env string) string {
	if envConfig, ok := c.envConfigs[env]; ok {
		return envConfig.Deploy.Parameters.StackName
	}
	return ""
}

// GetProfile 获取指定环境的 AWS profile
func (c *SAMConfig) GetProfile(env string) string {
	if envConfig, ok := c.envConfigs[env]; ok {
		return envConfig.Deploy.Parameters.Profile
	}
	return ""
}

// GetFunctionName 根据 stack_name 和环境生成函数名
// 格式: {stack_name}-function-default
func (c *SAMConfig) GetFunctionName(env string) string {
	stackName := c.GetStackName(env)
	if stackName == "" {
		return ""
	}
	return fmt.Sprintf("%s-function-default", stackName)
}
