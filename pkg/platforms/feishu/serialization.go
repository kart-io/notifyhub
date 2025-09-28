package feishu

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kart-io/notifyhub/pkg/notifyhub/config"
	"gopkg.in/yaml.v3"
)

// ToJSON serializes the configuration to JSON
func ToJSON(cfg *config.FeishuConfig) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}

// FromJSON deserializes configuration from JSON
func FromJSON(data []byte) (*config.FeishuConfig, error) {
	cfg := &config.FeishuConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	SetDefaults(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ToYAML serializes the configuration to YAML
func ToYAML(cfg *config.FeishuConfig) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// FromYAML deserializes configuration from YAML
func FromYAML(data []byte) (*config.FeishuConfig, error) {
	cfg := &config.FeishuConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	SetDefaults(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromFile loads configuration from a JSON or YAML file
func LoadFromFile(filename string) (*config.FeishuConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine format by file extension
	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		return FromYAML(data)
	} else if strings.HasSuffix(filename, ".json") {
		return FromJSON(data)
	} else {
		// Try JSON first, then YAML
		if cfg, err := FromJSON(data); err == nil {
			return cfg, nil
		}
		return FromYAML(data)
	}
}

// SaveToFile saves configuration to a JSON or YAML file
func SaveToFile(cfg *config.FeishuConfig, filename string) error {
	var data []byte
	var err error

	// Determine format by file extension
	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		data, err = ToYAML(cfg)
	} else {
		data, err = ToJSON(cfg)
	}

	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}