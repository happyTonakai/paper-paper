package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	API      APIConfig      `yaml:"api"`
	Obsidian ObsidianConfig `yaml:"obsidian"`
	UI       UIConfig       `yaml:"ui"`
}

type APIConfig struct {
	BaseURL      string `yaml:"base_url"`
	APIKey       string `yaml:"api_key"`
	DefaultModel string `yaml:"default_model"`
	LightModel   string `yaml:"light_model"`
}

type ObsidianConfig struct {
	VaultPath     string `yaml:"vault_path"`
	ExportFolder  string `yaml:"export_folder"`
}

type UIConfig struct {
	MaxRecentRounds int `yaml:"max_recent_rounds"`
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".paperpaper")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func PapersDir() string {
	return filepath.Join(ConfigDir(), "papers")
}

func PromptsDir() string {
	return filepath.Join(ConfigDir(), "prompts")
}

func Load() (*Config, error) {
	cfg := defaultConfig()

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Expand env vars in api_key
	cfg.API.APIKey = os.ExpandEnv(cfg.API.APIKey)

	// Expand ~ in paths
	cfg.Obsidian.VaultPath = expandHome(cfg.Obsidian.VaultPath)

	return cfg, nil
}

func defaultConfig() *Config {
	cfg := &Config{
		API: APIConfig{
			BaseURL:      "https://api.openai.com/v1",
			APIKey:       "${OPENAI_API_KEY}",
			DefaultModel: "gpt-4o",
			LightModel:   "gpt-4o-mini",
		},
		Obsidian: ObsidianConfig{
			VaultPath:    "~/Documents/Obsidian/MyVault",
			ExportFolder: "Papers",
		},
		UI: UIConfig{
			MaxRecentRounds: 5,
		},
	}

	// Env vars can override defaults (but config file values take precedence)
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.API.BaseURL = v
	}
	if v := os.Getenv("OPENAI_MODEL_NAME"); v != "" {
		cfg.API.DefaultModel = v
		cfg.API.LightModel = v
	}

	return cfg
}

func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Mask API key for saving
	saveCfg := *c
	if saveCfg.API.APIKey != "" && !strings.HasPrefix(saveCfg.API.APIKey, "${") {
		saveCfg.API.APIKey = "${OPENAI_API_KEY}"
	}

	data, err := yaml.Marshal(&saveCfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, path[1:])
}
