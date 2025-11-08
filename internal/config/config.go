package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DBURL       string `json:"db_url"`
	CurrentUser string `json:"current_user_name"`
}

func Write(cfg Config) error {
	if envPath := os.Getenv("GATOR_CONFIG"); envPath != "" {
		return saveToFile(envPath, cfg)
	}

	cwd, err := os.Getwd()
	if err == nil {
		localPath := filepath.Join(cwd, ".gatorconfig.json")
		if _, err := os.Stat(localPath); err == nil {
			return saveToFile(localPath, cfg)

		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find config path: %w", err)
	}
	homePath := filepath.Join(home, ".gatorconfig.json")

	return saveToFile(homePath, cfg)
}

func Read() (Config, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	body, err := os.ReadFile(path + "/.gatorconfig.json")
	if err != nil {
		return Config{}, err
	}

	var cfg Config

	if err := json.Unmarshal(body, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.DBURL == "" {
		return Config{}, errors.New("config db_url is empty")
	}
	return cfg, nil
}

func (c *Config) SetUser(username string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}
	c.CurrentUser = username
	return Write(*c)
}

func saveToFile(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}
