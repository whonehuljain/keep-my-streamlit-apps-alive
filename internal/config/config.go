package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type StreamlitApp struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Config struct {
	Apps     []StreamlitApp `json:"apps"`
	Schedule string         `json:"schedule"`
	Timeout  int            `json:"timeout_seconds"`
}

// reads configs from json file
func LoadConfig(filename string) (*Config, error) {
	// read from the file
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return createDefaultConfig(filename)
		}
		return nil, fmt.Errorf("error reading config files: %w", err)
	}

	var config Config

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing : %w", err)
	}

	return &config, nil
}

// creates default config for my app
func createDefaultConfig(filename string) (*Config, error) {
	defaultConfig := &Config{
		Apps: []StreamlitApp{
			{
				Name: "My App (F1nalyze)",
				URL:  "https://f1nalyze.streamlit.app/",
			},
		},
		Schedule: "0 */8 * * *",
		Timeout:  300,
	}

	data, err := json.MarshalIndent(defaultConfig, "", " ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling default config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return nil, fmt.Errorf("error writing default config: %w", err)
	}

	fmt.Printf("Created default config file: %s\n", filename)
	return defaultConfig, nil
}

// save configs to a file
func (c *Config) SaveConfig(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// adds new apps to config
func (c *Config) AddApp(name, url string) {
	c.Apps = append(c.Apps, StreamlitApp{
		Name: name,
		URL:  url,
	})
}

// checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Apps) == 0 {
		return fmt.Errorf("no apps configured")
	}

	for i, app := range c.Apps {
		if app.Name == "" {
			return fmt.Errorf("app %d: name cannot be empty", i)
		}
		if app.URL == "" {
			return fmt.Errorf("app %d (%s): URL cannot be empty", i, app.Name)
		}
	}

	if c.Schedule == "" {
		return fmt.Errorf("schedule cannot be empty")
	}

	if c.Timeout <= 0 {
		c.Timeout = 300 // 5 min
	}

	return nil
}
