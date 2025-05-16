package llm

import (
	"encoding/json"
	"fmt"

	"xiaozhi-server-go/src/core/types"
)

// Config LLM配置结构
type Config struct {
	Type        string                 `yaml:"type"`
	ModelName   string                 `yaml:"model_name"`
	BaseURL     string                 `yaml:"base_url,omitempty"`
	APIKey      string                 `yaml:"api_key,omitempty"`
	Temperature float64                `yaml:"temperature,omitempty"`
	MaxTokens   int                    `yaml:"max_tokens,omitempty"`
	TopP        float64                `yaml:"top_p,omitempty"`
	Extra       map[string]interface{} `yaml:",inline"`
}

// Provider LLM提供者接口
type Provider interface {
	types.LLMProvider
}

// BaseProvider LLM基础实现
type BaseProvider struct {
	config *Config
}

// Config 获取配置
func (p *BaseProvider) Config() *Config {
	return p.config
}

// NewBaseProvider 创建LLM基础提供者
func NewBaseProvider(config *Config) *BaseProvider {
	return &BaseProvider{
		config: config,
	}
}

// Initialize 初始化提供者
func (p *BaseProvider) Initialize() error {
	return nil
}

// Cleanup 清理资源
func (p *BaseProvider) Cleanup() error {
	return nil
}

// Factory LLM工厂函数类型
type Factory func(config *Config) (Provider, error)

var (
	factories = make(map[string]Factory)
)

// Register 注册LLM提供者工厂
func Register(name string, factory Factory) {
	factories[name] = factory
}

// Create 创建LLM提供者实例
func Create(name string, config *Config) (Provider, error) {
	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("未知的LLM提供者: %s", name)
	}

	provider, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("创建LLM提供者失败: %v", err)
	}

	if err := provider.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化LLM提供者失败: %v", err)
	}

	return provider, nil
}

// ParseFunctionArguments 解析函数调用参数
func ParseFunctionArguments(arguments string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &result); err != nil {
		return nil, fmt.Errorf("解析函数参数失败: %v", err)
	}
	return result, nil
}

// ValidateFunctionArguments 验证函数调用参数
func ValidateFunctionArguments(params types.FunctionParams, args map[string]interface{}) error {
	// 检查必需参数
	for _, required := range params.Required {
		if _, ok := args[required]; !ok {
			return fmt.Errorf("缺少必需参数: %s", required)
		}
	}

	// 验证参数类型和值
	for name, schema := range params.Properties {
		value, ok := args[name]
		if !ok {
			continue
		}

		switch schema.Type {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("参数 %s 类型错误: 需要string", name)
			}
			if len(schema.Enum) > 0 {
				valid := false
				for _, enum := range schema.Enum {
					if value.(string) == enum {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("参数 %s 值无效: %v", name, value)
				}
			}
		case "number":
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("参数 %s 类型错误: 需要number", name)
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("参数 %s 类型错误: 需要boolean", name)
			}
		}
	}

	return nil
}
