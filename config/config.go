package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken   string
	ChatID     string
	DiscordURL string
	ProxyURL   string
	ProxyType  string
	ProxyUser  string
	ProxyPass  string
}

var (
	config     *Config
	configOnce sync.Once
)

func GetConfig() (*Config, error) {
	var err error
	configOnce.Do(func() {
		if envErr := godotenv.Load(); envErr != nil {
			err = fmt.Errorf("error loading .env file: %w", envErr)
			return
		}

		config = &Config{
			BotToken:   os.Getenv("BOT_TOKEN"),
			ChatID:     os.Getenv("CHAT_ID"),
			DiscordURL: os.Getenv("DISCORD_URL"),
			ProxyURL:   os.Getenv("PROXY_URL"),
			ProxyType:  os.Getenv("PROXY_TYPE"),
			ProxyUser:  os.Getenv("PROXY_USER"),
			ProxyPass:  os.Getenv("PROXY_PASS"),
		}

		if config.BotToken == "" {
			err = fmt.Errorf("BOT_TOKEN environment variable is required")
			return
		}
		if config.ChatID == "" {
			err = fmt.Errorf("CHAT_ID environment variable is required")
			return
		}
		if config.DiscordURL == "" {
			err = fmt.Errorf("DISCORD_URL environment variable is required")
			return
		}
	})
	return config, err
}
