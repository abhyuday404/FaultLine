package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config is the main configuration structure.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

// Rule defines a single failure injection rule.
type Rule struct {
	Target  string  `yaml:"target"`
	Failure Failure `yaml:"failure"`
}

// Failure specifies the type and parameters of the failure.
type Failure struct {
	Type        string  `yaml:"type"` // "latency", "error", "flaky"
	LatencyMs   int     `yaml:"latency_ms,omitempty"`
	ErrorCode   int     `yaml:"error_code,omitempty"`
	Probability float64 `yaml:"probability,omitempty"` // For "flaky" type
}

// LoadConfig reads a YAML file and returns a Config struct.
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
