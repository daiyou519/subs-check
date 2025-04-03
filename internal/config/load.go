package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bestruirui/bestsub/internal/model"
)

var defaultConfig = &model.Config{
	Server: struct {
		Port int    `json:"port"`
		Host string `json:"host"`
	}{
		Port: 8080,
		Host: "0.0.0.0",
	},
	Database: struct {
		Path string `json:"path"`
	}{
		Path: "data/bestsub.db",
	},
	JWT: struct {
		Secret    string `json:"secret"`
		ExpiresIn int    `json:"expires_in"`
	}{
		Secret:    "bestsub-jwt-secret",
		ExpiresIn: 3600,
	},
}

func Load(path string) (*model.Config, error) {
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	var cfg *model.Config

	if _, fileErr := os.Stat(path); os.IsNotExist(fileErr) {
		var err error
		cfg, err = createDefaultConfig(path)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		cfg, err = readConfig(path)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func createDefaultConfig(path string) (*model.Config, error) {
	data, err := json.MarshalIndent(defaultConfig, "", "    ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return defaultConfig, nil
}

func readConfig(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
