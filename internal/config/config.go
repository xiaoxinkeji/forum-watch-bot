package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Telegram TelegramConfig `json:"telegram"`
	Admin    AdminConfig    `json:"admin"`
	Runtime  RuntimeConfig  `json:"runtime"`
	Web      WebConfig      `json:"web"`
	Sites    []SiteConfig   `json:"sites"`
}

type TelegramConfig struct {
	BotToken      string `json:"bot_token"`
	PushChannelID int64  `json:"push_channel_id"`
}

type AdminConfig struct {
	AdminUserIDs []int64 `json:"admin_user_ids"`
}

type RuntimeConfig struct {
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	DailyPushLimit      int    `json:"daily_push_limit"`
	DatabasePath        string `json:"database_path"`
	ProxyURL            string `json:"proxy_url"`
}

type WebConfig struct {
	Enabled  bool   `json:"enabled"`
	Listen   string `json:"listen"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type SiteConfig struct {
	ID      string            `json:"id"`
	Enabled bool              `json:"enabled"`
	Options map[string]string `json:"options"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Telegram.BotToken == "" {
		return nil, fmt.Errorf("telegram.bot_token is required")
	}
	if cfg.Telegram.PushChannelID == 0 {
		return nil, fmt.Errorf("telegram.push_channel_id is required")
	}
	if cfg.Runtime.PollIntervalSeconds <= 0 {
		cfg.Runtime.PollIntervalSeconds = 120
	}
	if cfg.Runtime.DailyPushLimit <= 0 {
		cfg.Runtime.DailyPushLimit = 10
	}
	if cfg.Runtime.DatabasePath == "" {
		cfg.Runtime.DatabasePath = "forum-watch-bot.db"
	}
	if cfg.Web.Enabled {
		if cfg.Web.Listen == "" {
			cfg.Web.Listen = ":8080"
		}
		if cfg.Web.Username == "" || cfg.Web.Password == "" {
			return nil, fmt.Errorf("web.username and web.password are required when web.enabled=true")
		}
	}
	return &cfg, nil
}
