package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the .dfo.yaml configuration file
type Config struct {
	Registry   RegistryConfig             `yaml:"registry"`
	SigningKey string                     `yaml:"signing_key,omitempty"`
	Containers map[string]ContainerConfig `yaml:"containers,omitempty"`
}

// RegistryConfig contains registry settings
type RegistryConfig struct {
	Default string `yaml:"default"`
}

// ContainerConfig contains per-container settings
type ContainerConfig struct {
	AdditionalRegistries []string `yaml:"additional_registries,omitempty"`
}

// Load reads and parses the configuration file
func Load(configPath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return &Config{
			Registry: RegistryConfig{
				Default: "", // No default registry
			},
			Containers: make(map[string]ContainerConfig),
		}, nil
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize containers map if nil
	if config.Containers == nil {
		config.Containers = make(map[string]ContainerConfig)
	}

	return &config, nil
}

// GetRegistriesForContainer returns all registries that a container should be pushed to
func (c *Config) GetRegistriesForContainer(containerName string) []string {
	var registries []string

	// Add default registry if set
	if c.Registry.Default != "" {
		registries = append(registries, c.Registry.Default)
	}

	// Add additional registries for this container
	if containerConfig, exists := c.Containers[containerName]; exists {
		registries = append(registries, containerConfig.AdditionalRegistries...)
	}

	return registries
}

// Save writes the configuration to a file
func (c *Config) Save(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
