package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config is the main configuration structure.
type Config struct {
	Rules    []Rule      `yaml:"rules"`
	TCPRules []TCPRule   `yaml:"tcpRules"`
	OpenAPI  OpenAPIConf `yaml:"openapi"`
}

// OpenAPIConf contains OpenAPI/Swagger discovery configuration
type OpenAPIConf struct {
	Enabled     bool     `yaml:"enabled"`
	SpecFiles   []string `yaml:"specFiles"`
	SearchPaths []string `yaml:"searchPaths"`
	AutoCreate  bool     `yaml:"autoCreate"` // Automatically create rules from discovered endpoints
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

// TCPRule defines a TCP-level proxy for DB/network fault injection
type TCPRule struct {
	Listen   string    `yaml:"listen"`   // e.g., 127.0.0.1:55432
	Upstream string    `yaml:"upstream"` // e.g., localhost:5432
	Faults   TCPFaults `yaml:"faults"`
}

// TCPFaults contains knobs to simulate network failures at L4
type TCPFaults struct {
	LatencyMs         int     `yaml:"latency_ms,omitempty"`
	DropProbability   float64 `yaml:"drop_probability,omitempty"`
	ResetProbability  float64 `yaml:"reset_probability,omitempty"`
	BandwidthKbps     int     `yaml:"bandwidth_kbps,omitempty"`
	RefuseConnections bool    `yaml:"refuse_connections,omitempty"`
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
