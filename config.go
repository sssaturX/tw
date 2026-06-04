package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Accounts []Account `json:"accounts"`
	Presets  []Preset  `json:"presets,omitempty"`
}

type Preset struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type Account struct {
	Name        string `json:"name"`
	Login       string `json:"login,omitempty"`
	Token       string `json:"token"`
	ClientID    string `json:"client_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	SOCKS5Proxy string `json:"socks5_proxy,omitempty"`
}

func loadConfig(path string) (Config, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(bs, &cfg); err != nil {
		return Config{}, err
	}
	if err := normalizeConfig(&cfg); err != nil {
		return Config{}, err
	}
	if len(cfg.Accounts) == 0 {
		return Config{}, errors.New("config has no accounts")
	}

	return cfg, nil
}

func loadOptionalConfig(path string) (Config, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(bs, &cfg); err != nil {
		return Config{}, err
	}
	if err := normalizeConfig(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func normalizeConfig(cfg *Config) error {
	seenPresets := map[string]bool{}
	for i := range cfg.Presets {
		preset := &cfg.Presets[i]
		preset.ID = strings.TrimSpace(preset.ID)
		preset.Title = strings.TrimSpace(preset.Title)
		preset.Message = strings.TrimSpace(preset.Message)
		if preset.Message == "" {
			return fmt.Errorf("preset %q has empty message", preset.Title)
		}
		if preset.ID == "" {
			preset.ID = fmt.Sprintf("preset_%d", i+1)
		}
		if preset.Title == "" {
			preset.Title = preset.Message
		}
		key := strings.ToLower(preset.ID)
		if seenPresets[key] {
			return fmt.Errorf("duplicate preset id %q", preset.ID)
		}
		seenPresets[key] = true
	}

	seen := map[string]bool{}
	for i := range cfg.Accounts {
		acc := &cfg.Accounts[i]
		acc.Name = strings.TrimSpace(acc.Name)
		acc.Login = normalizeLogin(acc.Login)
		acc.Token = sanitizeToken(acc.Token)
		acc.ClientID = strings.TrimSpace(acc.ClientID)
		acc.UserID = strings.TrimSpace(acc.UserID)
		acc.SOCKS5Proxy = strings.TrimSpace(acc.SOCKS5Proxy)

		if acc.Name == "" {
			if acc.Login != "" {
				acc.Name = acc.Login
			} else {
				acc.Name = fmt.Sprintf("account_%d", i+1)
			}
		}
		key := strings.ToLower(acc.Name)
		if seen[key] {
			return fmt.Errorf("duplicate account name %q", acc.Name)
		}
		seen[key] = true
		if acc.Token == "" {
			return fmt.Errorf("account %q has empty token", acc.Name)
		}
	}
	return nil
}

func saveConfig(path string, cfg Config) error {
	bs, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	bs = append(bs, '\n')
	return os.WriteFile(path, bs, 0600)
}

func selectAccounts(cfg Config, name string, all bool) ([]Account, error) {
	if all {
		return cfg.Accounts, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		if len(cfg.Accounts) == 1 {
			return []Account{cfg.Accounts[0]}, nil
		}
		return nil, errors.New("choose --account or --all")
	}
	for _, acc := range cfg.Accounts {
		if strings.EqualFold(acc.Name, name) || strings.EqualFold(acc.Login, name) {
			return []Account{acc}, nil
		}
	}
	return nil, fmt.Errorf("account %q not found", name)
}

func sanitizeToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "oauth:")
	token = strings.TrimPrefix(token, "OAuth ")
	token = strings.TrimPrefix(token, "Bearer ")
	return strings.TrimSpace(token)
}

func normalizeLogin(login string) string {
	login = strings.TrimSpace(strings.ToLower(login))
	return strings.TrimPrefix(login, "#")
}
