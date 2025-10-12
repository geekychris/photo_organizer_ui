package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DatabasePath  string `yaml:"database_path"`
	ThumbnailDir  string `yaml:"thumbnail_dir"`
	ThumbnailSize int    `yaml:"thumbnail_size"`
	WebPort       string `yaml:"web_port"`
}

func Load(configFile string) (*Config, error) {
	// Default configuration
	cfg := &Config{
		DatabasePath:  "photos.db",
		ThumbnailDir:  "thumbnails",
		ThumbnailSize: 300,
		WebPort:       "8080",
	}

	// If config file exists, load it
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Ensure paths are absolute
	if !filepath.IsAbs(cfg.DatabasePath) {
		cwd, _ := os.Getwd()
		cfg.DatabasePath = filepath.Join(cwd, cfg.DatabasePath)
	}

	if !filepath.IsAbs(cfg.ThumbnailDir) {
		cwd, _ := os.Getwd()
		cfg.ThumbnailDir = filepath.Join(cwd, cfg.ThumbnailDir)
	}

	// Create thumbnail directory if it doesn't exist
	if err := os.MkdirAll(cfg.ThumbnailDir, 0755); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save(configFile string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}