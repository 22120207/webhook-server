package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken             string
	ChatID               string
	DiscordURL           string
	ProxyURL             string
	ProxyType            string
	ProxyUser            string
	ProxyPass            string
	DiscordBotToken      string
	DiscordApplicationID string
	DiscordPublicKey     string
	DiscordChannelID     string
	MongoDBURI           string
	MongoDBDatabase      string
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
			BotToken:             os.Getenv("BOT_TOKEN"),
			ChatID:               os.Getenv("CHAT_ID"),
			DiscordURL:           os.Getenv("DISCORD_URL"),
			ProxyURL:             os.Getenv("PROXY_URL"),
			ProxyType:            os.Getenv("PROXY_TYPE"),
			ProxyUser:            os.Getenv("PROXY_USER"),
			ProxyPass:            os.Getenv("PROXY_PASS"),
			DiscordBotToken:      os.Getenv("DISCORD_BOT_TOKEN"),
			DiscordApplicationID: os.Getenv("DISCORD_APPLICATION_ID"),
			DiscordPublicKey:     os.Getenv("DISCORD_PUBLIC_KEY"),
			DiscordChannelID:     os.Getenv("DISCORD_CHANNEL_ID"),
			MongoDBURI:           os.Getenv("MONGODB_URI"),
			MongoDBDatabase:      os.Getenv("MONGODB_DATABASE"),
		}

		// Validate required fields
		if config.BotToken == "" {
			err = fmt.Errorf("BOT_TOKEN environment variable is required")
			return
		}
		if config.ChatID == "" {
			err = fmt.Errorf("CHAT_ID environment variable is required")
			return
		}
		if config.DiscordBotToken == "" {
			err = fmt.Errorf("DISCORD_BOT_TOKEN environment variable is required")
			return
		}
		if config.DiscordChannelID == "" {
			err = fmt.Errorf("DISCORD_CHANNEL_ID environment variable is required")
			return
		}
		if config.DiscordPublicKey == "" {
			err = fmt.Errorf("DISCORD_PUBLIC_KEY environment variable is required")
			return
		}
		if config.MongoDBURI == "" {
			err = fmt.Errorf("MONGODB_URI environment variable is required")
			return
		}
		if config.MongoDBDatabase == "" {
			err = fmt.Errorf("MONGODB_DATABASE environment variable is required")
			return
		}
	})
	return config, err
}
