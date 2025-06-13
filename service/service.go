package service

import (
	"context"
	"log"
	"webhook-server/service/config"
	"webhook-server/service/contact"
	"webhook-server/service/rest"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitService(mongoClient *mongo.Client) *rest.RestController {
	config, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	mongoClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(config.MongoDBURI))
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.TODO())

	discord, err := discordgo.New("Bot " + config.DiscordBotToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}
	defer discord.Close()

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Discord bot is ready")
	})
	if err := discord.Open(); err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	server := &rest.RestController{
		Telegram: &contact.TelegramSender{},
		Discord: &contact.DiscordSender{
			Discord:   discord,
			ChannelID: config.DiscordChannelID,
		},
		MongoClient: mongoClient,
	}

	return server
}
