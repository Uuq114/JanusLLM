package models

type ModelConfig struct {
	Name        string  `yaml:"name"`        // 模型名称
	Type        string  `yaml:"type"`        // 模型类型 (openai, claude, local)
	BaseURL     string  `yaml:"base_url"`    // API基础URL
	APIKey      string  `yaml:"api_key"`     // API密钥
	Weight      int     `yaml:"weight"`      // 负载均衡权重
	MaxTokens   int     `yaml:"max_tokens"`  // 最大token数
	Temperature float64 `yaml:"temperature"` // 温度参数
}

type ModelGroup struct {
	Name               string        `yaml:"name"`                  // 组名称
	Strategy           string        `yaml:"strategy"`              // 负载均衡策略 (round-robin, weighted)
	Models             []ModelConfig `yaml:"models"`                // 组内模型列表
	CostPerInputToken  float64       `yaml:"cost_per_input_token"`  // 输入token费用
	CostPerOutputToken float64       `yaml:"cost_per_output_token"` // 输出token费用
}
