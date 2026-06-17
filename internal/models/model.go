package models

type ModelConfig struct {
	Name            string  `yaml:"name"`
	Type            string  `yaml:"type"`
	BaseURL         string  `yaml:"base_url"`
	APIKey          string  `yaml:"api_key"`
	APIKeySecretRef string  `yaml:"api_key_secret_ref"`
	Weight          int     `yaml:"weight"`
	MaxTokens       int     `yaml:"max_tokens"`
	Temperature     float64 `yaml:"temperature"`
	TimeoutSeconds  int     `yaml:"timeout_seconds"`
	RetryTimes      int     `yaml:"retry_times"`
	SkipTLSVerify   bool    `yaml:"skip_tls_verify"`
}

type ModelGroup struct {
	Name string `yaml:"name"`
	// Strategy selects round-robin, weighted, latency, or client-sticky balancing.
	Strategy           string                 `yaml:"strategy"`
	Models             []ModelConfig          `yaml:"models"`
	CostPerInputToken  float64                `yaml:"cost_per_input_token"`
	CostPerOutputToken float64                `yaml:"cost_per_output_token"`
	RequestDefaults    map[string]interface{} `yaml:"request_defaults"`
}
