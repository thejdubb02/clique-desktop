package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	// Where the window was last left. Restarting for an update should not
	// cost somebody the size they chose.
	Window WindowBox `json:"window"`
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(base, "CLIque")
	if err := os.MkdirAll(d, 0700); err != nil {
		return "", err
	}
	return d, nil
}

func Load() (Config, error) {
	d, err := dir()
	if err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(filepath.Join(d, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(cfg Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, "config.json"), b, 0600)
}

func WebViewDataPath() string {
	d, err := dir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "webview")
}
