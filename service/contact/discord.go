package contact

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"webhook-server/service/config"
)

type IDiscordSender interface {
	SendDiscordMessage(message string) ([]byte, error)
	SendDiscordMessageWithComponents(message string, components []discordgo.MessageComponent) ([]byte, error)
	UpdateMessage(channelID, messageID, content string, components []discordgo.MessageComponent) error
}

type DiscordSender struct {
	Discord   *discordgo.Session
	ChannelID string
}

func (d *DiscordSender) SendDiscordMessage(message string) ([]byte, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	msg, err := d.Discord.ChannelMessageSend(config.DiscordChannelID, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send Discord message: %w", err)
	}
	return []byte(msg.ID), nil
}

func (d *DiscordSender) SendDiscordMessageWithComponents(message string, components []discordgo.MessageComponent) ([]byte, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	msg, err := d.Discord.ChannelMessageSendComplex(config.DiscordChannelID, &discordgo.MessageSend{
		Content:    message,
		Components: components,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send Discord message with components: %w", err)
	}
	return []byte(msg.ID), nil
}

func (d *DiscordSender) UpdateMessage(channelID, messageID, content string, components []discordgo.MessageComponent) error {
	_, err := d.Discord.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelID,
		ID:         messageID,
		Content:    &content,
		Components: &components,
	})
	if err != nil {
		return fmt.Errorf("failed to update Discord message: %w", err)
	}
	return nil
}
